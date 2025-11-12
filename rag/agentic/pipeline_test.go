package agentic

import (
	"context"
	"strings"
	"testing"

	"github.com/sweetpotato0/ai-allin/contrib/vector/inmemory"
	"github.com/sweetpotato0/ai-allin/message"
)

func TestPipelineRunProducesResponse(t *testing.T) {
	ctx := context.Background()

	planLLM := &stubLLM{
		response: `{"strategy":"baseline","steps":[{"id":"step-1","goal":"Check shipping policy","questions":["shipping policy details"],"expected_evidence":"official policy"}]}`,
	}
	writerLLM := &stubLLM{
		response: "Draft answer referencing [Doc:shipping-policy].",
	}
	criticLLM := &stubLLM{
		response: `{"verdict":"approve","issues":[],"final_answer":"Approved final answer with [Doc:shipping-policy]."}`,
	}

	store := inmemory.NewInMemoryVectorStore()
	embedder := &keywordEmbedder{}

	pipe, err := NewPipeline(
		Clients{
			Planner: planLLM,
			Writer:  writerLLM,
			Critic:  criticLLM,
		},
		embedder,
		store,
		WithTopK(2),
	)
	if err != nil {
		t.Fatalf("NewPipeline error: %v", err)
	}

	err = pipe.IndexDocuments(ctx,
		Document{ID: "shipping-policy", Title: "Shipping Policy", Content: "All shipping policy details and timelines.", Metadata: map[string]any{"source": "intranet"}},
		Document{ID: "returns", Title: "Return Policy", Content: "Return windows and shipping labels.", Metadata: map[string]any{"source": "help_center"}},
	)
	if err != nil {
		t.Fatalf("IndexDocuments error: %v", err)
	}

	resp, err := pipe.Run(ctx, "Tell me the shipping policy timeline.")
	if err != nil {
		t.Fatalf("pipeline run failed: %v", err)
	}

	if resp.Plan == nil || len(resp.Plan.Steps) != 1 {
		t.Fatalf("expected plan with 1 step, got %#v", resp.Plan)
	}
	if len(resp.Evidence) == 0 {
		t.Fatalf("expected evidence, got 0")
	}
	if resp.DraftAnswer == "" || resp.FinalAnswer == "" {
		t.Fatalf("expected non-empty answers")
	}
	if resp.Critic == nil || resp.Critic.Verdict != "approve" {
		t.Fatalf("expected critic approval")
	}
}

func TestPipelineWithoutCritic(t *testing.T) {
	ctx := context.Background()

	planLLM := &stubLLM{
		response: `{"strategy":"baseline","steps":[{"id":"","goal":"Understand returns","questions":["returns policy"],"expected_evidence":"policy doc"}]}`,
	}
	writerLLM := &stubLLM{
		response: "Return answer referencing [Doc:returns].",
	}

	store := inmemory.NewInMemoryVectorStore()
	embedder := &keywordEmbedder{}

	pipe, err := NewPipeline(
		Clients{
			Planner: planLLM,
			Writer:  writerLLM,
		},
		embedder,
		store,
		WithCritic(false),
	)
	if err != nil {
		t.Fatalf("NewPipeline error: %v", err)
	}

	if err := pipe.IndexDocuments(ctx, Document{ID: "returns", Title: "Return Policy", Content: "Return policy details."}); err != nil {
		t.Fatalf("IndexDocuments error: %v", err)
	}

	resp, err := pipe.Run(ctx, "What is the return policy?")
	if err != nil {
		t.Fatalf("pipeline run failed: %v", err)
	}
	if resp.Critic != nil {
		t.Fatalf("expected no critic feedback")
	}
	if resp.FinalAnswer != resp.DraftAnswer {
		t.Fatalf("expected final answer to equal draft when critic disabled")
	}
}

func TestPipelineSkipsWriterWithoutEvidence(t *testing.T) {
	ctx := context.Background()

	planLLM := &stubLLM{
		response: `{"strategy":"baseline","steps":[{"id":"step-1","goal":"Find escalation policy","questions":["escalation policy details"],"expected_evidence":"policy doc"}]}`,
	}
	writerLLM := &stubLLM{
		response: "This should never be returned.",
	}

	store := inmemory.NewInMemoryVectorStore()
	embedder := &keywordEmbedder{}

	fallback := "没有检索到相关内容"
	pipe, err := NewPipeline(
		Clients{
			Planner: planLLM,
			Writer:  writerLLM,
		},
		embedder,
		store,
		WithMinEvidenceCount(1),
		WithNoAnswerMessage(fallback),
	)
	if err != nil {
		t.Fatalf("NewPipeline error: %v", err)
	}

	resp, err := pipe.Run(ctx, "请告诉我最新的升级流程？")
	if err != nil {
		t.Fatalf("pipeline run failed: %v", err)
	}

	if len(resp.Evidence) != 0 {
		t.Fatalf("expected no evidence, got %d items", len(resp.Evidence))
	}
	if resp.DraftAnswer != fallback || resp.FinalAnswer != fallback {
		t.Fatalf("expected fallback answer %q, got draft=%q final=%q", fallback, resp.DraftAnswer, resp.FinalAnswer)
	}
	if writerLLM.calls != 0 {
		t.Fatalf("expected writer LLM to be skipped, got %d calls", writerLLM.calls)
	}
}

type stubLLM struct {
	response string
	calls    int
}

func (s *stubLLM) Generate(ctx context.Context, messages []*message.Message, tools []map[string]any) (*message.Message, error) {
	s.calls++
	return message.NewMessage(message.RoleAssistant, s.response), nil
}

func (s *stubLLM) SetTemperature(float64) {}
func (s *stubLLM) SetMaxTokens(int64)     {}
func (s *stubLLM) SetModel(string)        {}

type keywordEmbedder struct{}

var keywordSpace = []string{"shipping", "policy", "return", "timeline"}

func (k *keywordEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	vec := make([]float32, len(keywordSpace))
	lower := strings.ToLower(text)
	for idx, kw := range keywordSpace {
		if strings.Contains(lower, kw) {
			vec[idx] = 1
		}
	}
	return vec, nil
}

func (k *keywordEmbedder) Dimension() int {
	return 1024
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
