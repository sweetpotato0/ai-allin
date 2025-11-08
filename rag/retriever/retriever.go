package retriever

import (
	"context"
	"fmt"
	"sync"

	"github.com/sweetpotato0/ai-allin/rag/chunking"
	"github.com/sweetpotato0/ai-allin/rag/document"
	"github.com/sweetpotato0/ai-allin/rag/embedder"
	"github.com/sweetpotato0/ai-allin/rag/reranker"
	"github.com/sweetpotato0/ai-allin/vector"
)

// Config controls retrieval behaviour.
type Config struct {
	SearchTopK int
	RerankTopK int
}

// Option customizes retriever config.
type Option func(*Config)

// WithSearchTopK sets the number of neighbors fetched from the vector store.
func WithSearchTopK(k int) Option {
	return func(cfg *Config) {
		if k > 0 {
			cfg.SearchTopK = k
		}
	}
}

// WithRerankTopK sets how many documents survive reranking.
func WithRerankTopK(k int) Option {
	return func(cfg *Config) {
		if k > 0 {
			cfg.RerankTopK = k
		}
	}
}

// Retriever coordinates chunking, embedding, similarity search, and reranking.
type Retriever struct {
	store    vector.VectorStore
	embedder embedder.Embedder
	chunker  chunking.Chunker
	reranker reranker.Reranker
	cfg      Config

	mu        sync.RWMutex
	documents map[string]document.Document
	chunks    map[string]document.Chunk
}

// New creates a retriever.
func New(store vector.VectorStore, emb embedder.Embedder, chunker chunking.Chunker, rer reranker.Reranker, opts ...Option) *Retriever {
	cfg := Config{
		SearchTopK: 8,
		RerankTopK: 4,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return &Retriever{
		store:     store,
		embedder:  emb,
		chunker:   chunker,
		reranker:  rer,
		cfg:       cfg,
		documents: make(map[string]document.Document),
		chunks:    make(map[string]document.Chunk),
	}
}

// IndexDocuments ingests documents -> chunks -> embeddings -> vector store.
func (r *Retriever) IndexDocuments(ctx context.Context, docs ...document.Document) error {
	if r.store == nil || r.embedder == nil || r.chunker == nil {
		return fmt.Errorf("retriever not fully configured")
	}

	for _, doc := range docs {
		document.EnsureDocumentID(&doc)
		chunks, err := r.chunker.Chunk(ctx, doc)
		if err != nil {
			return fmt.Errorf("chunk document %s: %w", doc.ID, err)
		}

		for _, chunk := range chunks {
			vec, err := r.embedder.EmbedDocument(ctx, chunk)
			if err != nil {
				return fmt.Errorf("embed chunk %s: %w", chunk.ID, err)
			}
			embedding := &vector.Embedding{
				ID:     chunk.ID,
				Vector: vec,
				Text:   chunk.Content,
			}
			if err := r.store.AddEmbedding(ctx, embedding); err != nil {
				return fmt.Errorf("store chunk %s: %w", chunk.ID, err)
			}

			r.mu.Lock()
			r.chunks[chunk.ID] = chunk.Clone()
			r.documents[doc.ID] = doc.Clone()
			r.mu.Unlock()
		}
	}
	return nil
}

// Search executes semantic search followed by reranking.
func (r *Retriever) Search(ctx context.Context, query string) ([]reranker.Result, error) {
	queryVec, err := r.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	results, err := r.store.Search(ctx, queryVec, r.cfg.SearchTopK)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}

	candidates := make([]reranker.Candidate, 0, len(results))
	for _, hit := range results {
		chunk, ok := r.lookupChunk(hit.ID)
		if !ok {
			continue
		}
		candidates = append(candidates, reranker.Candidate{
			Chunk:  chunk,
			Vector: hit.Vector,
			Score:  0,
		})
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	if r.reranker == nil {
		output := make([]reranker.Result, 0, len(candidates))
		for _, cand := range candidates {
			output = append(output, reranker.Result{
				Chunk: cand.Chunk,
				Score: 0,
			})
		}
		return output, nil
	}

	reranked, err := r.reranker.Rank(ctx, queryVec, candidates)
	if err != nil {
		return nil, err
	}

	if r.cfg.RerankTopK > 0 && len(reranked) > r.cfg.RerankTopK {
		reranked = reranked[:r.cfg.RerankTopK]
	}
	return reranked, nil
}

// Document fetches a document by ID.
func (r *Retriever) Document(id string) (document.Document, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	doc, ok := r.documents[id]
	return doc.Clone(), ok
}

// lookupChunk retrieves chunk metadata.
func (r *Retriever) lookupChunk(id string) (document.Chunk, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	chunk, ok := r.chunks[id]
	if !ok {
		return document.Chunk{}, false
	}
	return chunk.Clone(), true
}

// Clear drops all indexed state.
func (r *Retriever) Clear(ctx context.Context) error {
	if r.store != nil {
		if err := r.store.Clear(ctx); err != nil {
			return err
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.chunks = make(map[string]document.Chunk)
	r.documents = make(map[string]document.Document)
	return nil
}

// Count returns number of chunks indexed.
func (r *Retriever) Count(ctx context.Context) (int, error) {
	if r.store == nil {
		return 0, nil
	}
	return r.store.Count(ctx)
}
