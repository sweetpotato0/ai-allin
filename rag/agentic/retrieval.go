package agentic

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"

	"github.com/sweetpotato0/ai-allin/pkg/logging"
	"github.com/sweetpotato0/ai-allin/pkg/telemetry"
	"github.com/sweetpotato0/ai-allin/rag/chunking"
	"github.com/sweetpotato0/ai-allin/rag/document"
	"github.com/sweetpotato0/ai-allin/rag/embedder"
	"github.com/sweetpotato0/ai-allin/rag/reranker"
	"github.com/sweetpotato0/ai-allin/rag/retriever"
	"github.com/sweetpotato0/ai-allin/rag/tokenizer"
	"github.com/sweetpotato0/ai-allin/vector"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
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
	logger   *slog.Logger
}

var agenticRetrievalTracer = otel.Tracer("github.com/sweetpotato0/ai-allin/rag/agentic/retrieval")

func (d *defaultRetrieval) IndexDocuments(ctx context.Context, docs ...document.Document) error {
	ctx, span := agenticRetrievalTracer.Start(ctx, "DefaultRetrieval.IndexDocuments",
		oteltrace.WithAttributes(attribute.Int("docs.count", len(docs))))
	var spanErr error
	defer func() { telemetry.End(span, spanErr) }()
	if d.logger != nil {
		d.logger.Info("default retrieval indexing documents", "count", len(docs))
	}
	if err := d.base.IndexDocuments(ctx, docs...); err != nil {
		if d.logger != nil {
			d.logger.Error("base retriever index failed", "error", err)
		}
		spanErr = err
		return err
	}
	d.keywords.add(docs...)
	return nil
}

func (d *defaultRetrieval) Search(ctx context.Context, query string) ([]RetrievalResult, error) {
	ctx, span := agenticRetrievalTracer.Start(ctx, "DefaultRetrieval.Search",
		oteltrace.WithAttributes(attribute.String("query", trimLogString(query, 80))))
	var spanErr error
	defer func() { telemetry.End(span, spanErr) }()
	if d.logger != nil {
		d.logger.Debug("default retrieval search started", "query", trimLogString(query, 80))
	}
	results, err := d.base.Search(ctx, query)
	if err != nil {
		if d.logger != nil {
			d.logger.Error("base retrieval search failed", "error", err)
		}
		spanErr = err
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
		if d.logger != nil {
			d.logger.Debug("hybrid search fallback triggered", "missing", target-len(out))
		}
		extras := d.keywords.search(query, target-len(out), seen)
		out = append(out, extras...)
	}
	if d.logger != nil {
		d.logger.Debug("default retrieval search completed", "query", trimLogString(query, 80), "hits", len(out))
	}
	span.SetAttributes(attribute.Int("results.count", len(out)))
	return out, nil
}

func (d *defaultRetrieval) Document(id string) (document.Document, bool) {
	return d.base.Document(id)
}

func (d *defaultRetrieval) Clear(ctx context.Context) error {
	if d.logger != nil {
		d.logger.Warn("clearing default retrieval index")
	}
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

	overlap := cfg.ChunkOverlap
	if overlap < 0 {
		overlap = 120
	}

	chunker := cfg.chunker
	if chunker == nil {
		tokenizer := tokenizer.NewSimpleTokenizer()
		if cfg.tokenizer != nil {
			tokenizer = cfg.tokenizer
		}
		chunker = chunking.NewSimpleChunker(
			chunking.WithTokenizer(tokenizer),
			chunking.WithOverlap(overlap),
		)
	}

	rer := cfg.reranker
	if rer == nil {
		rer = reranker.NewCosineReranker()
	}

	summar := cfg.summarizer
	adapter := embedder.NewVectorAdapterWithNormalization(emb, cfg.NormalizeEmbeddings)
	retrLogger := logging.WithComponent("retriever").With("pipeline", cfg.Name)
	opts := []retriever.Option{
		retriever.WithSearchTopK(cfg.TopK),
		retriever.WithRerankTopK(cfg.RerankTopK),
		retriever.WithLogger(retrLogger),
	}
	if cfg.preprocess != nil {
		opts = append(opts, retriever.WithPreprocessor(func(ctx context.Context, doc document.Document) (document.Document, error) {
			processed := doc.Clone()
			processed.Content = cfg.preprocess(processed.Content)
			return processed, nil
		}))
	}
	base := retriever.New(
		vec,
		adapter,
		chunker,
		summar,
		rer,
		opts...,
	)
	return &defaultRetrieval{
		base:     base,
		cfg:      cfg,
		keywords: newKeywordIndex(),
		logger:   logging.WithComponent("agentic_retrieval").With("pipeline", cfg.Name),
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

func trimLogString(text string, limit int) string {
	text = strings.TrimSpace(text)
	if limit <= 0 || len([]rune(text)) <= limit {
		return text
	}
	return string([]rune(text)[:limit]) + "..."
}
