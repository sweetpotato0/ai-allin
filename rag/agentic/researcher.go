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
	prompt string
}

func newResearcher(llm agent.LLMClient, cfg *Config) *researcher {
	return &researcher{
		llm:    llm,
		prompt: cfg.QueryPrompt,
	}
}

func (r *researcher) buildQueries(ctx context.Context, question string, step PlanStep) ([]string, error) {
	if r.llm == nil {
		return r.fallbackQueries(step), nil
	}

	userPrompt := fmt.Sprintf("Original question:\n%s\n\nPlan step goal:\n%s\n\nKnown sub-questions:\n%s\nReturn JSON.", question, step.Goal, strings.Join(step.Questions, "\n"))
	msgs := []*message.Message{
		message.NewMessage(message.RoleSystem, r.prompt),
		message.NewMessage(message.RoleUser, userPrompt),
	}
	resp, err := r.llm.Generate(ctx, msgs, nil)
	if err != nil {
		return nil, fmt.Errorf("query agent failed: %w", err)
	}

	plan, err := decodeJSON[queryPlan](resp.Content)
	if err != nil {
		return nil, fmt.Errorf("query agent invalid output: %w", err)
	}

	if len(plan.Queries) == 0 {
		return r.fallbackQueries(step), nil
	}

	return plan.Queries, nil
}

func (r *researcher) fallbackQueries(step PlanStep) []string {
	if len(step.Questions) > 0 {
		return step.Questions
	}
	return []string{step.Goal}
}
