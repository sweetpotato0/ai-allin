package agentic

import (
	"context"
	"fmt"

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

// defaultRetrieval is a thin adapter around rag/retriever that satisfies RetrievalEngine.
type defaultRetrieval struct {
	base *retriever.Retriever
}

func (d *defaultRetrieval) IndexDocuments(ctx context.Context, docs ...document.Document) error {
	return d.base.IndexDocuments(ctx, docs...)
}

func (d *defaultRetrieval) Search(ctx context.Context, query string) ([]RetrievalResult, error) {
	results, err := d.base.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	out := make([]RetrievalResult, 0, len(results))
	for _, res := range results {
		out = append(out, RetrievalResult{
			Chunk: res.Chunk,
			Score: res.Score,
		})
	}
	return out, nil
}

func (d *defaultRetrieval) Document(id string) (document.Document, bool) {
	return d.base.Document(id)
}

func (d *defaultRetrieval) Clear(ctx context.Context) error {
	return d.base.Clear(ctx)
}

func (d *defaultRetrieval) Count(ctx context.Context) (int, error) {
	return d.base.Count(ctx)
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
		chunker = chunking.NewSimpleChunker(
			chunking.WithChunkSize(chunkSize),
			chunking.WithOverlap(overlap),
		)
	}

	rer := cfg.reranker
	if rer == nil {
		rer = reranker.NewCosineReranker()
	}

	adapter := embedder.NewVectorAdapter(emb)
	base := retriever.New(
		vec,
		adapter,
		chunker,
		rer,
		retriever.WithSearchTopK(cfg.TopK),
		retriever.WithRerankTopK(cfg.RerankTopK),
	)
	return &defaultRetrieval{base: base}, nil
}
