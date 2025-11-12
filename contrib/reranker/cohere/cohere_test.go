package cohere

import (
	"context"
	"testing"

	"github.com/sweetpotato0/ai-allin/rag/document"
	"github.com/sweetpotato0/ai-allin/rag/reranker"
)

type stubReranker struct {
	called bool
}

func (s *stubReranker) Rank(ctx context.Context, q []float32, c []reranker.Candidate) ([]reranker.Result, error) {
	s.called = true
	return []reranker.Result{
		{Chunk: c[0].Chunk, Score: 0.5},
	}, nil
}

func TestCohereRerankerFallsBack(t *testing.T) {
	fallback := &stubReranker{}
	client := New("", WithFallback(fallback))

	ctx := reranker.ContextWithQuery(context.Background(), "测试 query")
	candidates := []reranker.Candidate{
		{Chunk: document.Chunk{ID: "chunk-1", Content: "AADDCC"}},
	}

	results, err := client.Rank(ctx, nil, candidates)
	if err != nil {
		t.Fatalf("Rank error: %v", err)
	}
	if len(results) != 1 || !fallback.called {
		t.Fatalf("expected fallback path")
	}
}
