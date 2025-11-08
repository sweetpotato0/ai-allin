package reranker

import (
	"context"
	"sort"

	"github.com/sweetpotato0/ai-allin/rag/document"
	"github.com/sweetpotato0/ai-allin/vector"
)

// Candidate represents a retrieved chunk and its stored vector.
type Candidate struct {
	Chunk  document.Chunk
	Vector []float32
	Score  float32
}

// Result is the final reranked chunk.
type Result struct {
	Chunk document.Chunk
	Score float32
}

// Reranker reorders retrieval candidates, optionally refining similarity.
type Reranker interface {
	Rank(ctx context.Context, queryVector []float32, candidates []Candidate) ([]Result, error)
}

// CosineReranker sorts candidates by cosine similarity with the query vector.
type CosineReranker struct{}

// NewCosineReranker creates a reranker based on cosine similarity.
func NewCosineReranker() *CosineReranker {
	return &CosineReranker{}
}

// Rank implements the Reranker interface.
func (c *CosineReranker) Rank(ctx context.Context, queryVector []float32, candidates []Candidate) ([]Result, error) {
	results := make([]Result, 0, len(candidates))
	for _, cand := range candidates {
		score := cand.Score
		if len(cand.Vector) > 0 && len(queryVector) == len(cand.Vector) {
			score = vector.CosineSimilarity(queryVector, cand.Vector)
		}
		results = append(results, Result{
			Chunk: cand.Chunk,
			Score: score,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results, nil
}
