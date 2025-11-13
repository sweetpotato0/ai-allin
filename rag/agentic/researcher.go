package agentic

import (
	"context"
	"fmt"
	"strings"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/message"
)

type queryPlan struct {
	Queries []string `json:"queries"`
}

type researcher struct {
	llm    agent.LLMClient
	cfg    *Config
	prompt string
}

func newResearcher(llm agent.LLMClient, cfg *Config) *researcher {
	return &researcher{
		llm:    llm,
		prompt: cfg.QueryPrompt,
		cfg:    cfg,
	}
}

func (r *researcher) buildQueries(ctx context.Context, question string, step PlanStep) ([]string, error) {

	var queries []string
	var err error
	if r.llm != nil {
		queries, err = r.generateWithLLM(ctx, question, step)
	}
	if len(queries) == 0 {
		queries = r.syntheticQueries(question, step)
	}
	queries = limitQueries(dedupeQueries(queries), r.cfg.QueryMaxResults)
	if len(queries) == 0 {
		queries = []string{strings.TrimSpace(step.Goal)}
	}
	return queries, err
}

func (r *researcher) generateWithLLM(ctx context.Context, question string, step PlanStep) ([]string, error) {
	userPrompt := fmt.Sprintf("Original question:\n%s\n\nPlan step goal:\n%s\n\nExpected evidence:\n%s\n\nKnown sub-questions:\n%s\n\nReturn strict JSON matching {\"queries\": [\"...\"], \"question\": \"original question\"}. Provide diverse, concrete search queries (max %d).",
		question,
		step.Goal,
		step.ExpectedEvidence,
		strings.Join(step.Questions, "\n"),
		max(1, r.cfg.QueryMaxResults),
	)
	msgs := []*message.Message{
		message.NewMessage(message.RoleSystem, r.prompt),
		message.NewMessage(message.RoleUser, userPrompt),
	}

	attempts := 1
	if r.cfg.QueryLLMRetries > 0 {
		attempts += r.cfg.QueryLLMRetries
	}

	var lastErr error
	for i := 0; i < attempts; i++ {
		genResp, err := r.llm.Generate(ctx, &agent.GenerateRequest{
			Messages: msgs,
		})
		if err != nil {
			lastErr = fmt.Errorf("query agent failed: %w", err)
			continue
		}
		if genResp == nil || genResp.Message == nil {
			lastErr = fmt.Errorf("query agent returned empty response")
			continue
		}
		plan, err := decodeJSON[queryPlan](genResp.Message.Text())
		if err != nil {
			lastErr = fmt.Errorf("query agent invalid output: %w", err)
			continue
		}
		if len(plan.Queries) == 0 {
			lastErr = fmt.Errorf("query agent returned 0 queries")
			continue
		}
		return plan.Queries, nil
	}
	return nil, lastErr
}

func (r *researcher) syntheticQueries(question string, step PlanStep) []string {
	maxResults := max(1, r.cfg.QueryMaxResults)
	candidates := make([]string, 0, maxResults)

	for _, q := range step.Questions {
		candidates = append(candidates, q)
	}
	if step.ExpectedEvidence != "" {
		candidates = append(candidates, fmt.Sprintf("%s %s", step.Goal, step.ExpectedEvidence))
	}
	if question != "" {
		candidates = append(candidates, fmt.Sprintf("%s %s", step.Goal, question))
	}
	candidates = append(candidates, step.Goal)

	if len(candidates) < maxResults {
		keywords := keywordVariants(step.Goal, question)
		candidates = append(candidates, keywords...)
	}

	return limitQueries(dedupeQueries(candidates), maxResults)
}

// --- helpers ---

func limitQueries(queries []string, maxResults int) []string {
	if maxResults <= 0 || len(queries) <= maxResults {
		return queries
	}
	return queries[:maxResults]
}

func dedupeQueries(queries []string) []string {
	seen := make(map[string]struct{}, len(queries))
	out := make([]string, 0, len(queries))
	for _, q := range queries {
		clean := strings.TrimSpace(q)
		if clean == "" {
			continue
		}
		lower := strings.ToLower(clean)
		if _, ok := seen[lower]; ok {
			continue
		}
		seen[lower] = struct{}{}
		out = append(out, clean)
	}
	return out
}

func keywordVariants(goal, question string) []string {
	var variants []string
	base := strings.Fields(goal)
	qwords := strings.Fields(question)
	if len(base) > 0 {
		variants = append(variants, strings.Join(base, " "))
	}
	if len(qwords) > 0 && len(base) > 0 {
		variants = append(variants, fmt.Sprintf("%s %s", base[0], qwords[0]))
	}
	return variants
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
