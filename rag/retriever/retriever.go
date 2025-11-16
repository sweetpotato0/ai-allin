package retriever

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/sweetpotato0/ai-allin/pkg/logging"
	"github.com/sweetpotato0/ai-allin/rag/chunking"
	"github.com/sweetpotato0/ai-allin/rag/document"
	"github.com/sweetpotato0/ai-allin/rag/embedder"
	"github.com/sweetpotato0/ai-allin/rag/reranker"
	"github.com/sweetpotato0/ai-allin/rag/summarizer"
	"github.com/sweetpotato0/ai-allin/vector"
)

// Config controls retrieval behaviour.
type Config struct {
	SearchTopK   int
	RerankTopK   int
	Preprocessor PreprocessFunc
	Logger       *slog.Logger
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

// WithPreprocessor wires a document preprocessor that runs before chunking.
func WithPreprocessor(fn PreprocessFunc) Option {
	return func(cfg *Config) {
		if fn != nil {
			cfg.Preprocessor = fn
		}
	}
}

// WithLogger injects a structured logger.
func WithLogger(logger *slog.Logger) Option {
	return func(cfg *Config) {
		cfg.Logger = logger
	}
}

// Retriever coordinates chunking, embedding, similarity search, and reranking.
type Retriever struct {
	store      vector.VectorStore
	embedder   embedder.Embedder
	chunker    chunking.Chunker
	summaryer  summarizer.Summarizer
	reranker   reranker.Reranker
	preprocess PreprocessFunc
	cfg        Config
	logger     *slog.Logger

	mu        sync.RWMutex
	documents map[string]document.Document
	chunks    map[string]document.Chunk
}

// PreprocessFunc transforms documents before chunking (e.g. cleaning HTML).
type PreprocessFunc func(ctx context.Context, doc document.Document) (document.Document, error)

// New creates a retriever.
func New(store vector.VectorStore, emb embedder.Embedder, chunker chunking.Chunker, summaryer summarizer.Summarizer, rer reranker.Reranker, opts ...Option) *Retriever {
	cfg := Config{
		SearchTopK: 8,
		RerankTopK: 4,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	logger := cfg.Logger
	if logger == nil {
		logger = logging.WithComponent("retriever")
	}
	return &Retriever{
		store:      store,
		embedder:   emb,
		chunker:    chunker,
		reranker:   rer,
		summaryer:  summaryer,
		preprocess: cfg.Preprocessor,
		cfg:        cfg,
		logger:     logger,
		documents:  make(map[string]document.Document),
		chunks:     make(map[string]document.Chunk),
	}
}

// IndexDocuments ingests documents -> preprocess -> chunks -> embeddings -> vector store.
func (r *Retriever) IndexDocuments(ctx context.Context, docs ...document.Document) error {
	if r.store == nil || r.embedder == nil || r.chunker == nil {
		return fmt.Errorf("retriever not fully configured")
	}

	if r.logger != nil {
		r.logger.Info("retriever indexing documents", "count", len(docs))
	}

	for _, input := range docs {
		doc := input.Clone()
		var err error
		if r.preprocess != nil {
			doc, err = r.preprocess(ctx, doc)
			if err != nil {
				if r.logger != nil {
					r.logger.Error("document preprocess failed", "doc_id", doc.ID, "error", err)
				}
				return fmt.Errorf("preprocess document %s: %w", doc.ID, err)
			}
		}

		if strings.TrimSpace(doc.ID) == "" {
			doc.ID = document.GenDocumentID(doc.Source, doc.Content)
		}

		chunks, err := r.chunker.Chunk(ctx, doc)
		if err != nil {
			if r.logger != nil {
				r.logger.Error("chunking document failed", "doc_id", doc.ID, "error", err)
			}
			return fmt.Errorf("chunk document %s: %w", doc.ID, err)
		}
		if r.logger != nil {
			r.logger.Debug("document chunked", "doc_id", doc.ID, "chunks", len(chunks))
		}

		// generate summaries in batch
		summaries := []document.Summary{}
		if r.summaryer != nil {
			summaries, err = r.summaryer.SummarizeChunks(ctx, chunks)
			if err != nil {
				if r.logger != nil {
					r.logger.Error("chunk summarisation failed", "doc_id", doc.ID, "error", err)
				}
				return fmt.Errorf("summaryer chunks %s: %w", doc.ID, err)
			}
		}

		for i, chunk := range chunks {
			vec, err := r.embedder.EmbedDocument(ctx, chunk)
			if err != nil {
				if r.logger != nil {
					r.logger.Error("chunk embedding failed", "chunk_id", chunk.ID, "error", err)
				}
				return fmt.Errorf("embed chunk %s: %w", chunk.ID, err)
			}
			embedding := &vector.Embedding{
				ID:     chunk.ID,
				Vector: vec,
				Text:   chunk.Content,
			}
			if err := r.store.AddEmbedding(ctx, embedding); err != nil {
				if r.logger != nil {
					r.logger.Error("storing chunk embedding failed", "chunk_id", chunk.ID, "error", err)
				}
				return fmt.Errorf("store chunk %s: %w", chunk.ID, err)
			}

			r.mu.Lock()
			r.chunks[chunk.ID] = chunk.Clone()
			r.documents[doc.ID] = doc.Clone()
			r.mu.Unlock()

			if len(summaries) != 0 && i < len(summaries) {
				summaryChunk := chunk.Clone()
				summaryChunk.ID = chunk.ID + "_summary"
				summaryChunk.Content = summaries[i].Summary
				if summaryChunk.Metadata == nil {
					summaryChunk.Metadata = make(map[string]any)
				}
				summaryChunk.Metadata["section"] = "summary"
				summaryChunk.Metadata["source_chunk"] = chunk.ID

				vec, err = r.embedder.EmbedDocument(ctx, summaryChunk)
				if err != nil {
					if r.logger != nil {
						r.logger.Error("summary chunk embedding failed", "chunk_id", summaryChunk.ID, "error", err)
					}
					return fmt.Errorf("embed summary chunk %s: %w", chunk.ID, err)
				}
				summary := &vector.Embedding{
					ID:     summaryChunk.ID,
					Vector: vec,
					Text:   summaries[i].Summary,
				}
				if err := r.store.AddEmbedding(ctx, summary); err != nil {
					if r.logger != nil {
						r.logger.Error("storing summary embedding failed", "chunk_id", summaryChunk.ID, "error", err)
					}
					return fmt.Errorf("store summary chunk %s: %w", chunk.ID, err)
				}

				r.mu.Lock()
				r.chunks[summaryChunk.ID] = summaryChunk.Clone()
				r.mu.Unlock()
			}
		}
	}
	return nil
}

// Search executes semantic search followed by reranking.
func (r *Retriever) Search(ctx context.Context, query string) ([]reranker.Result, error) {
	if r.logger != nil {
		r.logger.Debug("retriever search started", "query", trimLogText(query, 80))
	}
	queryVec, err := r.embedder.EmbedQuery(ctx, query)
	if err != nil {
		if r.logger != nil {
			r.logger.Error("query embedding failed", "error", err)
		}
		return nil, fmt.Errorf("embed query: %w", err)
	}
	results, err := r.store.Search(ctx, queryVec, r.cfg.SearchTopK)
	if err != nil {
		if r.logger != nil {
			r.logger.Error("vector search failed", "error", err)
		}
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

	ctxWithQuery := reranker.ContextWithQuery(ctx, query)
	reranked, err := r.reranker.Rank(ctxWithQuery, queryVec, candidates)
	if err != nil {
		return nil, err
	}

	if r.cfg.RerankTopK > 0 && len(reranked) > r.cfg.RerankTopK {
		reranked = reranked[:r.cfg.RerankTopK]
	}
	if r.logger != nil {
		r.logger.Debug("retriever search completed", "query", trimLogText(query, 80), "hits", len(reranked))
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

func trimLogText(text string, limit int) string {
	text = strings.TrimSpace(text)
	if limit <= 0 || len([]rune(text)) <= limit {
		return text
	}
	return string([]rune(text)[:limit]) + "..."
}
