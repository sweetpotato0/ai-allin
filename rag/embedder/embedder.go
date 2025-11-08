package embedder

import (
	"context"

	"github.com/sweetpotato0/ai-allin/rag/document"
	"github.com/sweetpotato0/ai-allin/vector"
)

// Embedder exposes methods tailored for RAG components.
type Embedder interface {
	EmbedDocument(ctx context.Context, chunk document.Chunk) ([]float32, error)
	EmbedQuery(ctx context.Context, query string) ([]float32, error)
}

// VectorAdapter bridges the generic vector.Embedder interface into a rag Embedder.
type VectorAdapter struct {
	base vector.Embedder
}

// NewVectorAdapter creates a new adapter.
func NewVectorAdapter(base vector.Embedder) *VectorAdapter {
	return &VectorAdapter{base: base}
}

// EmbedDocument embeds the chunk content using the base embedder.
func (v *VectorAdapter) EmbedDocument(ctx context.Context, chunk document.Chunk) ([]float32, error) {
	if v == nil || v.base == nil {
		return nil, nil
	}
	return v.base.Embed(ctx, chunk.Content)
}

// EmbedQuery embeds the query string.
func (v *VectorAdapter) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	if v == nil || v.base == nil {
		return nil, nil
	}
	return v.base.Embed(ctx, query)
}
