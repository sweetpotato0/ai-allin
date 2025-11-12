package mmr

import (
	"context"
	"testing"

	"github.com/sweetpotato0/ai-allin/rag/document"
	"github.com/sweetpotato0/ai-allin/rag/reranker"
)

func TestMMRRanksWithoutDuplicates(t *testing.T) {
	r := New()
	query := []float32{1, 0}
	candidates := []reranker.Candidate{
		{Chunk: document.Chunk{ID: "c1"}, Vector: []float32{1, 0}, Score: 0.9},
		{Chunk: document.Chunk{ID: "c2"}, Vector: []float32{0.9, 0.1}, Score: 0.85},
		{Chunk: document.Chunk{ID: "c3"}, Vector: []float32{0, 1}, Score: 0.4},
	}
	results, err := r.Rank(context.Background(), query, candidates)
	if err != nil {
		t.Fatalf("rank error: %v", err)
	}
	if len(results) != len(candidates) {
		t.Fatalf("expected %d results, got %d", len(candidates), len(results))
	}
	if results[2].Chunk.ID != "c3" {
		t.Fatalf("expected diverse chunk last, got %s", results[2].Chunk.ID)
	}
}
