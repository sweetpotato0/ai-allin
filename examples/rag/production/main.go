package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/sweetpotato0/ai-allin/contrib/chunking/markdown"
	"github.com/sweetpotato0/ai-allin/contrib/chunking/token"
	"github.com/sweetpotato0/ai-allin/contrib/reranker/cohere"
	"github.com/sweetpotato0/ai-allin/contrib/reranker/mmr"
	"github.com/sweetpotato0/ai-allin/contrib/retrieval/hybrid"
	inmemory "github.com/sweetpotato0/ai-allin/contrib/vector/inmemory"
	"github.com/sweetpotato0/ai-allin/message"
	"github.com/sweetpotato0/ai-allin/rag/agentic"
	"github.com/sweetpotato0/ai-allin/rag/embedder"
)

func main() {
	ctx := context.Background()

	llm := &consoleLLM{}
	store := inmemory.NewInMemoryVectorStore()
	baseEmbedder := newKeywordEmbedder()

	mdChunker := markdown.New(
		markdown.WithFallbackChunker(token.New(token.WithMaxTokens(300), token.WithOverlapTokens(50))),
		markdown.WithMaxHeadingLevel(3),
	)

	cohereKey := os.Getenv("COHERE_API_KEY")
	cohereReranker := cohere.New(cohereKey, cohere.WithFallback(mmr.New()))

	retrievalEngine, err := hybrid.New(
		store,
		embedder.NewVectorAdapter(baseEmbedder),
		hybrid.WithChunker(mdChunker),
		hybrid.WithReranker(cohereReranker),
	)
	if err != nil {
		log.Fatalf("build hybrid engine: %v", err)
	}

	pipeline, err := agentic.NewPipeline(
		agentic.Clients{Default: llm},
		baseEmbedder,
		store,
		agentic.WithRetriever(retrievalEngine),
	)
	if err != nil {
		log.Fatalf("build pipeline: %v", err)
	}

	if err := pipeline.IndexDocuments(ctx, sampleDocs()...); err != nil {
		log.Fatalf("index docs: %v", err)
	}

	start := time.Now()
	resp, err := pipeline.Run(ctx, "AADDCC 吃多了会怎样？")
	if err != nil {
		log.Fatalf("run failed: %v", err)
	}

	fmt.Println("Hybrid Agentic RAG finished in", time.Since(start))
	fmt.Println("Final answer:\n", resp.FinalAnswer)
}

func sampleDocs() []agentic.Document {
	return []agentic.Document{
		{
			ID:    "aaddcc",
			Title: "AADDCC 介绍",
			Content: `# AADDCC

AADDCC 是一种万能药物，可以治疗任何疾病。

## 副作用

吃多了会让人精神异常，呆滞。建议严格按照处方使用。`,
		},
		{
			ID:      "faq",
			Title:   "FAQ",
			Content: "常规感冒无需使用 AADDCC。",
		},
	}
}

// --- Supporting stubs for the example ---

type consoleLLM struct{}

func (c *consoleLLM) Generate(ctx context.Context, msgs []*message.Message, tools []map[string]any) (*message.Message, error) {
	return message.NewMessage(message.RoleAssistant, "示例：根据混合检索，AADDCC 虽然万能，但过量会导致精神异常。"), nil
}

func (c *consoleLLM) SetTemperature(float64) {}
func (c *consoleLLM) SetMaxTokens(int64)     {}
func (c *consoleLLM) SetModel(string)        {}

type keywordEmbedder struct{}

func newKeywordEmbedder() *keywordEmbedder { return &keywordEmbedder{} }

func (k *keywordEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	vec := make([]float32, 4)
	lower := strings.ToLower(text)
	if strings.Contains(lower, "aaddcc") {
		vec[0] = 1
	}
	if strings.Contains(lower, "副作用") || strings.Contains(lower, "risk") {
		vec[1] = 1
	}
	if strings.Contains(lower, "万能") {
		vec[2] = 1
	}
	return vec, nil
}

func (k *keywordEmbedder) Dimension() int { return 4 }

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
