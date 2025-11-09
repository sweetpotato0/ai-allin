package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/sweetpotato0/ai-allin/contrib/provider/openai"
	inmemory "github.com/sweetpotato0/ai-allin/contrib/vector/inmemory"
	"github.com/sweetpotato0/ai-allin/rag/agentic"
)

func main() {
	ctx := context.Background()
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY is required to run the Agentic RAG example")
	}

	baseURL := os.Getenv("OPENAI_API_BASE_URL")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY is required to run the Agentic RAG example")
	}

	llm := openai.New(openai.DefaultConfig().WithAPIKey(apiKey).WithBaseURL(baseURL))

	// In production, replace keywordEmbedder with a proper embedding provider (OpenAI, Cohere, etc).
	embedder := newKeywordEmbedder()
	store := inmemory.NewInMemoryVectorStore()

	pipeline, err := agentic.NewPipeline(
		agentic.Clients{Default: llm},
		embedder,
		store,
		agentic.WithTopK(3),
	)
	if err != nil {
		log.Fatalf("build pipeline: %v", err)
	}

	if err := pipeline.IndexDocuments(ctx, sampleDocuments()...); err != nil {
		log.Fatalf("index documents: %v", err)
	}

	start := time.Now()
	response, err := pipeline.Run(ctx, "What should I know about shipping timelines and return policy?")
	if err != nil {
		log.Fatalf("pipeline run failed: %v", err)
	}
	fmt.Println("Agentic RAG finished in", time.Since(start))

	fmt.Printf("\nPlan (%d steps):\n", len(response.Plan.Steps))
	for _, step := range response.Plan.Steps {
		fmt.Printf(" - %s: %s\n", step.ID, step.Goal)
	}

	fmt.Printf("\nEvidence (%d docs):\n", len(response.Evidence))
	for _, ev := range response.Evidence {
		fmt.Printf(" • Step %s matched %s (score %.2f)\n", ev.StepID, ev.Document.Title, ev.Score)
	}

	fmt.Println("\nDraft answer:\n", response.DraftAnswer)
	fmt.Println("\nFinal answer:\n", response.FinalAnswer)
}

func sampleDocuments() []agentic.Document {
	return []agentic.Document{
		{
			ID:      "shipping-policy",
			Title:   "Shipping Policy",
			Content: "Orders ship within 2 business days. Expedited shipping delivers in 48 hours. International routes take 5-7 days.",
			Metadata: map[string]any{
				"source": "knowledge-base",
			},
		},
		{
			ID:      "return-policy",
			Title:   "Return Policy",
			Content: "Customers have 30 days to return items. Damaged goods qualify for free replacement labels.",
			Metadata: map[string]any{
				"source": "knowledge-base",
			},
		},
		{
			ID:      "faq",
			Title:   "FAQ Snippets",
			Content: "Shipping and returns apply to all regions except Alaska and Hawaii.",
			Metadata: map[string]any{
				"source": "faq",
			},
		},
	}
}

// keywordEmbedder is a toy embedder used for the example. Replace with a real embedding model.
type keywordEmbedder struct {
	keywords []string
}

func newKeywordEmbedder() *keywordEmbedder {
	return &keywordEmbedder{
		keywords: []string{"shipping", "return", "policy", "timeline", "days", "international"},
	}
}

func (k *keywordEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	vec := make([]float32, len(k.keywords))
	lower := strings.ToLower(text)
	for idx, kw := range k.keywords {
		if strings.Contains(lower, kw) {
			vec[idx] = 1
		}
	}
	return vec, nil
}

func (k *keywordEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i, text := range texts {
		vec, err := k.Embed(ctx, text)
		if err != nil {
			return nil, err
		}
		out[i] = vec
	}
	return out, nil
}
