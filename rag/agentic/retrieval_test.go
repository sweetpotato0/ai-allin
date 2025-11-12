package agentic

import (
	"context"
	"strings"
	"testing"

	"github.com/sweetpotato0/ai-allin/contrib/vector/inmemory"
	"github.com/sweetpotato0/ai-allin/rag/document"
)

func TestDefaultRetrievalFallsBackToKeyword(t *testing.T) {
	ctx := context.Background()
	store := inmemory.NewInMemoryVectorStore()
	emb := &constantEmbedder{}
	cfg := defaultConfig()
	cfg.MinSearchScore = 1.1
	cfg.HybridTopK = 2
	cfg.EnableHybridSearch = true

	engine, err := newDefaultRetrievalEngine(store, emb, cfg)
	if err != nil {
		t.Fatalf("newDefaultRetrievalEngine error: %v", err)
	}

	doc := document.Document{
		ID:      "aaddcc",
		Title:   "AADDCC",
		Content: "# AADDCC\n\nAADDCC 是一种万能药物，可以治疗任何疾病。但是吃多了会让人精神异常，呆滞。",
	}
	if err := engine.IndexDocuments(ctx, doc); err != nil {
		t.Fatalf("IndexDocuments error: %v", err)
	}

	results, err := engine.Search(ctx, "AADDCC 万能药物 副作用")
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("expected hybrid fallback results")
	}

	var keywordHit bool
	for _, res := range results {
		if res.Chunk.Metadata["retrieval"] == "keyword" {
			keywordHit = true
			if !strings.Contains(res.Chunk.Content, "万能药物") {
				t.Fatalf("expected keyword chunk to include body text, got %q", res.Chunk.Content)
			}
		}
	}
	if !keywordHit {
		t.Fatalf("expected at least one keyword fallback chunk, got %#v", results)
	}
}

type constantEmbedder struct{}

func (c *constantEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	return []float32{1, 0, 0, 0}, nil
}

func (c *constantEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = []float32{1, 0, 0, 0}
	}
	return out, nil
}

func (c *constantEmbedder) Dimension() int {
	return 4
}
