package openai

import (
	"context"
	"errors"
	"fmt"
	"strings"

	openaisdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/sweetpotato0/ai-allin/vector"
)

// OpenAIEmbedder implements vector.Embedder by using openai.
type OpenAIEmbedder struct {
	client    openaisdk.Client
	model     openaisdk.EmbeddingModel
	dimension int
}

// New create OpenAIEmbedder.
func New(apiKey, baseURL string, model openaisdk.EmbeddingModel, dimension int) vector.Embedder {
	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if strings.TrimSpace(baseURL) != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	client := openaisdk.NewClient(opts...)
	return &OpenAIEmbedder{
		client:    client,
		model:     model,
		dimension: dimension,
	}
}

// Dimension return number of embedding dimensions
func (e *OpenAIEmbedder) Dimension() int {
	return e.dimension
}

// Embed converts text to a vector embedding
func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	vectors, err := e.embedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vectors) == 0 {
		return nil, errors.New("no embedding returned")
	}
	return vectors[0], nil
}

// EmbedBatch converts multiple texts to embeddings
func (e *OpenAIEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	return e.embedBatch(ctx, texts)
}

func (e *OpenAIEmbedder) embedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	params := openaisdk.EmbeddingNewParams{
		Model: e.model,
		Input: openaisdk.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: texts,
		},
	}

	resp, err := e.client.Embeddings.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create embeddings: %w", err)
	}
	if len(resp.Data) != len(texts) {
		return nil, fmt.Errorf("expected %d embeddings, got %d", len(texts), len(resp.Data))
	}

	out := make([][]float32, len(resp.Data))
	for i, emb := range resp.Data {
		out[i] = convertVector(emb.Embedding, e.dimension)
	}
	return out, nil
}

func convertVector(input []float64, expected int) []float32 {
	vec := make([]float32, expected)
	for i := 0; i < len(input) && i < expected; i++ {
		vec[i] = float32(input[i])
	}
	return vec
}
