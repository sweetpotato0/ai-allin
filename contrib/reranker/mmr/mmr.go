package mmr

import (
	"context"
	"math"

	"github.com/sweetpotato0/ai-allin/rag/reranker"
	"github.com/sweetpotato0/ai-allin/vector"
)

// Reranker implements Max Marginal Relevance to reduce redundancy.
type Reranker struct {
	Lambda float32
	Limit  int
}

// New returns an MMR reranker with sensible defaults.
func New() *Reranker {
	return &Reranker{
		Lambda: 0.7,
		Limit:  8,
	}
}

// Rank implements reranker.Reranker.
func (m *Reranker) Rank(ctx context.Context, queryVec []float32, candidates []reranker.Candidate) ([]reranker.Result, error) {
	if len(candidates) == 0 {
		return nil, nil
	}
	type item struct {
		cand  reranker.Candidate
		score float32
	}
	remaining := make([]item, len(candidates))
	for i, cand := range candidates {
		score := cand.Score
		if len(queryVec) > 0 && len(cand.Vector) == len(queryVec) {
			score = vector.CosineSimilarity(queryVec, cand.Vector)
		}
		remaining[i] = item{cand: cand, score: score}
	}

	selected := make([]reranker.Result, 0, len(candidates))
	selectedCandidates := make([]reranker.Candidate, 0, len(candidates))
	for len(remaining) > 0 && (m.Limit <= 0 || len(selected) < m.Limit) {
		bestIdx := -1
		bestScore := float32(math.Inf(-1))
		for idx, candidate := range remaining {
			diversityPenalty := float32(0)
			for _, picked := range selectedCandidates {
				if len(candidate.cand.Vector) == 0 || len(picked.Vector) != len(candidate.cand.Vector) {
					continue
				}
				diversityPenalty = max(diversityPenalty, vector.CosineSimilarity(candidate.cand.Vector, picked.Vector))
			}
			score := m.Lambda*candidate.score - (1-m.Lambda)*diversityPenalty
			if score > bestScore {
				bestScore = score
				bestIdx = idx
			}
		}
		if bestIdx == -1 {
			break
		}
		best := remaining[bestIdx]
		selected = append(selected, reranker.Result{
			Chunk: best.cand.Chunk,
			Score: best.score,
		})
		selectedCandidates = append(selectedCandidates, best.cand)
		remaining = append(remaining[:bestIdx], remaining[bestIdx+1:]...)
	}

	return selected, nil
}

func max(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
