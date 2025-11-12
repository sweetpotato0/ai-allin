package hybrid

import (
	"context"
	"errors"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/sweetpotato0/ai-allin/rag/agentic"
	"github.com/sweetpotato0/ai-allin/rag/chunking"
	"github.com/sweetpotato0/ai-allin/rag/document"
	"github.com/sweetpotato0/ai-allin/rag/embedder"
	"github.com/sweetpotato0/ai-allin/rag/reranker"
	"github.com/sweetpotato0/ai-allin/vector"
)

// Config configures the hybrid retrieval engine.
type Config struct {
	VectorTopK    int
	RerankTopK    int
	KeywordTopK   int
	VectorWeight  float32
	KeywordWeight float32
	Chunker       chunking.Chunker
	Reranker      reranker.Reranker
}

// Option customises the engine config.
type Option func(*Config)

// WithVectorTopK sets how many vector hits are pulled from the store.
func WithVectorTopK(k int) Option {
	return func(cfg *Config) {
		if k > 0 {
			cfg.VectorTopK = k
		}
	}
}

// WithRerankTopK limits how many results survive the vector reranker.
func WithRerankTopK(k int) Option {
	return func(cfg *Config) {
		if k > 0 {
			cfg.RerankTopK = k
		}
	}
}

// WithKeywordTopK caps keyword/BM25 results that merge into the final list.
func WithKeywordTopK(k int) Option {
	return func(cfg *Config) {
		if k > 0 {
			cfg.KeywordTopK = k
		}
	}
}

// WithWeights customises the contribution of vector vs. keyword search (defaults 0.7/0.3).
func WithWeights(vectorWeight, keywordWeight float32) Option {
	return func(cfg *Config) {
		if vectorWeight >= 0 && keywordWeight >= 0 {
			cfg.VectorWeight = vectorWeight
			cfg.KeywordWeight = keywordWeight
		}
	}
}

// WithChunker overrides the sectioning strategy.
func WithChunker(ch chunking.Chunker) Option {
	return func(cfg *Config) {
		if ch != nil {
			cfg.Chunker = ch
		}
	}
}

// WithReranker overrides the reranker implementation.
func WithReranker(r reranker.Reranker) Option {
	return func(cfg *Config) {
		if r != nil {
			cfg.Reranker = r
		}
	}
}

// Engine composes semantic vector search with a lightweight BM25 index.
type Engine struct {
	store    vector.VectorStore
	embedder embedder.Embedder
	cfg      Config

	mu        sync.RWMutex
	chunker   chunking.Chunker
	reranker  reranker.Reranker
	documents map[string]document.Document
	chunks    map[string]document.Chunk
	keyword   *bm25Index
}

// New creates a hybrid retrieval engine.
func New(store vector.VectorStore, emb embedder.Embedder, opts ...Option) (*Engine, error) {
	if store == nil || emb == nil {
		return nil, errors.New("store and embedder are required")
	}
	cfg := Config{
		VectorTopK:    12,
		RerankTopK:    6,
		KeywordTopK:   6,
		VectorWeight:  0.7,
		KeywordWeight: 0.3,
		Chunker: chunking.NewSimpleChunker(
			chunking.WithChunkSize(900),
			chunking.WithOverlap(150),
		),
		Reranker: reranker.NewCosineReranker(),
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &Engine{
		store:     store,
		embedder:  emb,
		cfg:       cfg,
		chunker:   cfg.Chunker,
		reranker:  cfg.Reranker,
		documents: make(map[string]document.Document),
		chunks:    make(map[string]document.Chunk),
		keyword:   newBM25(),
	}, nil
}

// IndexDocuments ingests the provided documents.
func (e *Engine) IndexDocuments(ctx context.Context, docs ...document.Document) error {
	if e.chunker == nil {
		return errors.New("chunker not configured")
	}
	for _, doc := range docs {
		document.EnsureDocumentID(&doc)
		chunks, err := e.chunker.Chunk(ctx, doc)
		if err != nil {
			return err
		}
		for _, chunk := range chunks {
			vec, err := e.embedder.EmbedDocument(ctx, chunk)
			if err != nil {
				return err
			}
			if err := e.store.AddEmbedding(ctx, &vector.Embedding{
				ID:     chunk.ID,
				Vector: vec,
				Text:   chunk.Content,
			}); err != nil {
				return err
			}
			e.keyword.add(chunk)
			e.mu.Lock()
			e.chunks[chunk.ID] = chunk.Clone()
			e.documents[doc.ID] = doc.Clone()
			e.mu.Unlock()
		}
	}
	return nil
}

// Search returns retrieval results blending vector and keyword matches.
func (e *Engine) Search(ctx context.Context, query string) ([]agentic.RetrievalResult, error) {
	queryVec, err := e.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	vecHits, err := e.store.Search(ctx, queryVec, e.cfg.VectorTopK)
	if err != nil {
		return nil, err
	}

	vecCandidates := make([]reranker.Candidate, 0, len(vecHits))
	for _, hit := range vecHits {
		chunk, ok := e.chunk(hit.ID)
		if !ok {
			continue
		}
		vecCandidates = append(vecCandidates, reranker.Candidate{
			Chunk:  chunk,
			Vector: hit.Vector,
			Score:  0,
		})
	}

	var vecResults []reranker.Result
	if len(vecCandidates) > 0 && e.reranker != nil {
		vecResults, err = e.reranker.Rank(ctx, queryVec, vecCandidates)
		if err != nil {
			return nil, err
		}
		if e.cfg.RerankTopK > 0 && len(vecResults) > e.cfg.RerankTopK {
			vecResults = vecResults[:e.cfg.RerankTopK]
		}
	}

	keywordHits := e.keyword.search(query, e.cfg.KeywordTopK)

	type scoredChunk struct {
		chunk document.Chunk
		score float32
	}

	scoreMap := make(map[string]scoredChunk)
	for _, hit := range keywordHits {
		chunk, ok := e.chunk(hit.ID)
		if !ok {
			continue
		}
		entry := scoreMap[chunk.ID]
		entry.chunk = chunk
		entry.score += hit.Score * e.cfg.KeywordWeight
		scoreMap[chunk.ID] = entry
	}
	for _, res := range vecResults {
		entry := scoreMap[res.Chunk.ID]
		entry.chunk = res.Chunk
		entry.score += res.Score * e.cfg.VectorWeight
		scoreMap[res.Chunk.ID] = entry
	}

	final := make([]scoredChunk, 0, len(scoreMap))
	for _, sc := range scoreMap {
		final = append(final, sc)
	}
	sort.Slice(final, func(i, j int) bool {
		return final[i].score > final[j].score
	})

	results := make([]agentic.RetrievalResult, 0, len(final))
	for _, sc := range final {
		results = append(results, agentic.RetrievalResult{
			Chunk: sc.chunk,
			Score: sc.score,
		})
	}
	return results, nil
}

// Document returns a cloned document by ID.
func (e *Engine) Document(id string) (document.Document, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	doc, ok := e.documents[id]
	return doc.Clone(), ok
}

// Clear removes all indexed state.
func (e *Engine) Clear(ctx context.Context) error {
	if err := e.store.Clear(ctx); err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.documents = make(map[string]document.Document)
	e.chunks = make(map[string]document.Chunk)
	e.keyword = newBM25()
	return nil
}

// Count returns the number of indexed chunks.
func (e *Engine) Count(ctx context.Context) (int, error) {
	return e.store.Count(ctx)
}

func (e *Engine) chunk(id string) (document.Chunk, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	chunk, ok := e.chunks[id]
	return chunk.Clone(), ok
}

// --- BM25 implementation ---

type bm25Index struct {
	mu          sync.RWMutex
	docFreq     map[string]int
	postings    map[string]map[string]int
	chunkLength map[string]int
	totalLength int
	docCount    int
	k1          float64
	b           float64
}

var bm25Regex = regexp.MustCompile(`\p{L}[\p{L}\p{M}]*|\p{N}+`)

func newBM25() *bm25Index {
	return &bm25Index{
		docFreq:     make(map[string]int),
		postings:    make(map[string]map[string]int),
		chunkLength: make(map[string]int),
		k1:          1.6,
		b:           0.75,
	}
}

func (b *bm25Index) add(chunk document.Chunk) {
	terms := tokenize(chunk.Content)
	if len(terms) == 0 {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.docCount++
	b.chunkLength[chunk.ID] = len(terms)
	b.totalLength += len(terms)

	seen := make(map[string]struct{})
	for _, term := range terms {
		if _, ok := b.postings[term]; !ok {
			b.postings[term] = make(map[string]int)
		}
		b.postings[term][chunk.ID]++
		if _, exists := seen[term]; !exists {
			b.docFreq[term]++
			seen[term] = struct{}{}
		}
	}
}

type keywordResult struct {
	ID    string
	Score float32
}

func (b *bm25Index) search(query string, limit int) []keywordResult {
	terms := unique(tokenize(query))
	if len(terms) == 0 {
		return nil
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.docCount == 0 {
		return nil
	}
	avgLen := float64(b.totalLength) / float64(b.docCount)
	scores := make(map[string]float64)
	for _, term := range terms {
		postings := b.postings[term]
		if len(postings) == 0 {
			continue
		}
		df := b.docFreq[term]
		idf := math.Log((float64(b.docCount)-float64(df)+0.5)/(float64(df)+0.5) + 1)
		for chunkID, tf := range postings {
			docLen := float64(b.chunkLength[chunkID])
			numerator := float64(tf) * (b.k1 + 1)
			denominator := float64(tf) + b.k1*(1-b.b+b.b*(docLen/avgLen))
			scores[chunkID] += idf * (numerator / denominator)
		}
	}
	results := make([]keywordResult, 0, len(scores))
	for id, score := range scores {
		results = append(results, keywordResult{ID: id, Score: float32(score)})
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results
}

func tokenize(content string) []string {
	lower := strings.ToLower(content)
	matches := bm25Regex.FindAllString(lower, -1)
	return matches
}

func unique(tokens []string) []string {
	if len(tokens) == 0 {
		return tokens
	}
	seen := make(map[string]struct{}, len(tokens))
	out := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		if _, ok := seen[tok]; ok {
			continue
		}
		seen[tok] = struct{}{}
		out = append(out, tok)
	}
	return out
}
