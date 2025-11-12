package vector

import (
	"context"
	"math"
)

// Embedding represents a vector embedding
type Embedding struct {
	ID     string
	Vector []float32
	Text   string
}

// VectorStore defines the interface for vector storage and similarity search
type VectorStore interface {
	// AddEmbedding adds a new embedding to the store
	AddEmbedding(ctx context.Context, embedding *Embedding) error

	// Search finds embeddings similar to the query vector
	Search(ctx context.Context, queryVector []float32, topK int) ([]*Embedding, error)

	// DeleteEmbedding removes an embedding by ID
	DeleteEmbedding(ctx context.Context, id string) error

	// GetEmbedding retrieves a specific embedding by ID
	GetEmbedding(ctx context.Context, id string) (*Embedding, error)

	// Clear removes all embeddings
	Clear(ctx context.Context) error

	// Count returns the number of embeddings
	Count(ctx context.Context) (int, error)
}

// Embedder defines the interface for creating embeddings from text
type Embedder interface {
	// Embed converts text to a vector embedding
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch converts multiple texts to embeddings
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

	// Dimension return number of embedding dimensions
	Dimension() int
}

// CosineSimilarityOperator returns the PostgreSQL operator for cosine similarity
func CosineSimilarityOperator() string {
	return "<->"
}

// CosineSimilarity calculates the cosine similarity between two vectors
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float32
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (normA*normB + 1e-8)
}

// EuclideanDistance calculates the Euclidean distance between two vectors
func EuclideanDistance(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var sum float32
	for i := 0; i < len(a); i++ {
		diff := a[i] - b[i]
		sum += diff * diff
	}

	// Use a simple square root approximation for performance
	if sum == 0 {
		return 0
	}

	x := sum
	y := x
	for i := 0; i < 10; i++ {
		y = (y + x/y) / 2
	}
	return y
}

// Normalize scales the vector to unit length (L2 norm).
func Normalize(vec []float32) []float32 {
	if len(vec) == 0 {
		return vec
	}
	var sum float64
	for _, v := range vec {
		sum += float64(v) * float64(v)
	}
	if sum == 0 {
		return vec
	}
	inv := float32(1 / math.Sqrt(sum))
	for i := range vec {
		vec[i] *= inv
	}
	return vec
}
