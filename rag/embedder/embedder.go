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
	base      vector.Embedder
	normalize bool
}

// NewVectorAdapter creates a new adapter.
func NewVectorAdapter(base vector.Embedder) *VectorAdapter {
	return &VectorAdapter{base: base}
}

// NewVectorAdapterWithNormalization toggles L2-normalisation for all embeddings.
func NewVectorAdapterWithNormalization(base vector.Embedder, normalize bool) *VectorAdapter {
	return &VectorAdapter{
		base:      base,
		normalize: normalize,
	}
}

func (v *VectorAdapter) maybeNormalize(vec []float32) []float32 {
	if !v.normalize || len(vec) == 0 {
		return vec
	}
	return vector.Normalize(vec)
}

// EmbedDocument embeds the chunk content using the base embedder.
func (v *VectorAdapter) EmbedDocument(ctx context.Context, chunk document.Chunk) ([]float32, error) {
	if v == nil || v.base == nil {
		return nil, nil
	}
	vec, err := v.base.Embed(ctx, chunk.Content)
	if err != nil {
		return nil, err
	}
	return v.maybeNormalize(vec), nil
}

// EmbedQuery embeds the query string.
func (v *VectorAdapter) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	if v == nil || v.base == nil {
		return nil, nil
	}
	vec, err := v.base.Embed(ctx, query)
	if err != nil {
		return nil, err
	}
	return v.maybeNormalize(vec), nil
}
