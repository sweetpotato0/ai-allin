package agentic

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/sweetpotato0/ai-allin/rag/chunking"
	"github.com/sweetpotato0/ai-allin/rag/document"
	"github.com/sweetpotato0/ai-allin/rag/embedder"
	"github.com/sweetpotato0/ai-allin/rag/reranker"
	"github.com/sweetpotato0/ai-allin/rag/retriever"
	"github.com/sweetpotato0/ai-allin/vector"
)

// RetrievalResult captures a single chunk returned from the retrieval stage.
type RetrievalResult struct {
	Chunk document.Chunk
	Score float32
}

// RetrievalEngine represents the contract the pipeline relies on for indexing/search.
// Implementations may wrap the default chunk/embed/rerank pipeline provided here or
// delegate to an entirely different system (e.g. hybrid BM25 + vector service).
type RetrievalEngine interface {
	IndexDocuments(ctx context.Context, docs ...document.Document) error
	Search(ctx context.Context, query string) ([]RetrievalResult, error)
	Document(id string) (document.Document, bool)
	Clear(ctx context.Context) error
	Count(ctx context.Context) (int, error)
}

// defaultRetrieval composes semantic + keyword retrieval strategies.
type defaultRetrieval struct {
	base     *retriever.Retriever
	cfg      *Config
	keywords *keywordIndex
}

func (d *defaultRetrieval) IndexDocuments(ctx context.Context, docs ...document.Document) error {
	if err := d.base.IndexDocuments(ctx, docs...); err != nil {
		return err
	}
	d.keywords.add(docs...)
	return nil
}

func (d *defaultRetrieval) Search(ctx context.Context, query string) ([]RetrievalResult, error) {
	results, err := d.base.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	out := make([]RetrievalResult, 0, len(results))
	seen := make(map[string]struct{}, len(results))
	for _, res := range results {
		score := d.adjustScore(res.Chunk, res.Score)
		if score < d.cfg.MinSearchScore {
			continue
		}
		seen[res.Chunk.ID] = struct{}{}
		out = append(out, RetrievalResult{
			Chunk: res.Chunk,
			Score: score,
		})
	}
	target := d.cfg.RerankTopK
	if d.cfg.HybridTopK > 0 {
		target = d.cfg.HybridTopK
	}
	if d.cfg.EnableHybridSearch && len(out) < target {
		extras := d.keywords.search(query, target-len(out), seen)
		out = append(out, extras...)
	}
	return out, nil
}

func (d *defaultRetrieval) Document(id string) (document.Document, bool) {
	return d.base.Document(id)
}

func (d *defaultRetrieval) Clear(ctx context.Context) error {
	if err := d.base.Clear(ctx); err != nil {
		return err
	}
	d.keywords.reset()
	return nil
}

func (d *defaultRetrieval) Count(ctx context.Context) (int, error) {
	return d.base.Count(ctx)
}

func (d *defaultRetrieval) adjustScore(chunk document.Chunk, score float32) float32 {
	if d.cfg == nil {
		return score
	}
	if d.cfg.TitleScorePenalty > 0 && d.cfg.TitleScorePenalty < 1 {
		if section, ok := chunk.Metadata["section"]; ok {
			if str, ok := section.(string); ok && str == "title" {
				return score * d.cfg.TitleScorePenalty
			}
		}
	}
	return score
}

func newDefaultRetrievalEngine(vec vector.VectorStore, emb vector.Embedder, cfg *Config) (RetrievalEngine, error) {
	if vec == nil {
		return nil, fmt.Errorf("vector store is required")
	}
	if emb == nil {
		return nil, fmt.Errorf("embedder is required")
	}

	chunkSize := cfg.ChunkSize
	if chunkSize <= 0 {
		chunkSize = 800
	}
	overlap := cfg.ChunkOverlap
	if overlap < 0 {
		overlap = 120
	}

	chunker := cfg.chunker
	if chunker == nil {
		separator := cfg.ChunkSeparator
		if strings.TrimSpace(separator) == "" {
			separator = "\n\n"
		}
		minSize := cfg.ChunkMinSize
		if minSize < 0 {
			minSize = 0
		}
		chunker = chunking.NewSimpleChunker(
			chunking.WithChunkSize(chunkSize),
			chunking.WithOverlap(overlap),
			chunking.WithSeparator(separator),
			chunking.WithMinChunkSize(minSize),
			chunking.WithSectionTagging(true),
		)
	}

	rer := cfg.reranker
	if rer == nil {
		rer = reranker.NewCosineReranker()
	}

	adapter := embedder.NewVectorAdapterWithNormalization(emb, cfg.NormalizeEmbeddings)
	base := retriever.New(
		vec,
		adapter,
		chunker,
		rer,
		retriever.WithSearchTopK(cfg.TopK),
		retriever.WithRerankTopK(cfg.RerankTopK),
	)
	return &defaultRetrieval{
		base:     base,
		cfg:      cfg,
		keywords: newKeywordIndex(),
	}, nil
}

type keywordIndex struct {
	mu   sync.RWMutex
	docs map[string]document.Document
}

func newKeywordIndex() *keywordIndex {
	return &keywordIndex{
		docs: make(map[string]document.Document),
	}
}

func (k *keywordIndex) add(docs ...document.Document) {
	if k == nil {
		return
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	for _, doc := range docs {
		if strings.TrimSpace(doc.Content) == "" || doc.ID == "" {
			continue
		}
		k.docs[doc.ID] = doc.Clone()
	}
}

func (k *keywordIndex) reset() {
	if k == nil {
		return
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	k.docs = make(map[string]document.Document)
}

func (k *keywordIndex) search(query string, limit int, seen map[string]struct{}) []RetrievalResult {
	if k == nil || limit <= 0 {
		return nil
	}
	tokens := tokenize(query)
	if len(tokens) == 0 {
		return nil
	}
	k.mu.RLock()
	defer k.mu.RUnlock()
	type candidate struct {
		doc   document.Document
		score float32
	}
	matches := make([]candidate, 0, len(k.docs))
	for _, doc := range k.docs {
		lower := strings.ToLower(doc.Content)
		var hits int
		for _, token := range tokens {
			if strings.Contains(lower, token) {
				hits++
			}
		}
		if hits == 0 {
			continue
		}
		score := float32(hits) / float32(len(tokens))
		matches = append(matches, candidate{doc: doc.Clone(), score: score})
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].score > matches[j].score
	})
	results := make([]RetrievalResult, 0, limit)
	for _, match := range matches {
		if len(results) >= limit {
			break
		}
		chunk := keywordChunk(match.doc, seen)
		if chunk.ID == "" {
			continue
		}
		if _, exists := seen[chunk.ID]; exists {
			continue
		}
		seen[chunk.ID] = struct{}{}
		results = append(results, RetrievalResult{
			Chunk: chunk,
			Score: match.score,
		})
	}
	return results
}

func tokenize(query string) []string {
	lower := strings.ToLower(query)
	raw := strings.Fields(lower)
	dedup := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, token := range raw {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		dedup = append(dedup, token)
	}
	return dedup
}

func keywordChunk(doc document.Document, seen map[string]struct{}) document.Chunk {
	if doc.ID == "" {
		return document.Chunk{}
	}
	id := doc.ID + "_kw"
	if seen != nil {
		if _, exists := seen[id]; exists {
			return document.Chunk{}
		}
	}
	content := strings.TrimSpace(doc.Content)
	if len([]rune(content)) > 480 {
		content = string([]rune(content)[:480])
	}
	chunk := document.Chunk{
		ID:         id,
		DocumentID: doc.ID,
		Content:    content,
		Metadata:   cloneMetadata(doc.Metadata),
	}
	if chunk.Metadata == nil {
		chunk.Metadata = make(map[string]any)
	}
	chunk.Metadata["section"] = "body"
	chunk.Metadata["retrieval"] = "keyword"
	return chunk
}
