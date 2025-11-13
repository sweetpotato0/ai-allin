package agentic

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/message"
)

type critic struct {
	llm    agent.LLMClient
	prompt string
}

func newCritic(llm agent.LLMClient, cfg *Config) *critic {
	if llm == nil {
		return nil
	}
	return &critic{
		llm:    llm,
		prompt: cfg.CriticPrompt,
	}
}

func (c *critic) Review(ctx context.Context, question, draft string, plan *Plan, evidence []Evidence) (*CriticFeedback, error) {
	if c == nil || c.llm == nil {
		return nil, nil
	}

	var planJSON string
	if plan != nil {
		if data, err := json.Marshal(plan); err == nil {
			planJSON = string(data)
		}
	}

	userPrompt := fmt.Sprintf("Question:\n%s\n\nPlan:\n%s\n\nEvidence:\n%s\n\nDraft answer:\n%s\n\nReturn JSON only.", question, planJSON, formatEvidence(evidence), draft)
	msgs := []*message.Message{
		message.NewMessage(message.RoleSystem, c.prompt),
		message.NewMessage(message.RoleUser, userPrompt),
	}

	genResp, err := c.llm.Generate(ctx, &agent.GenerateRequest{
		Messages: msgs,
	})
	if err != nil {
		return nil, fmt.Errorf("critic failed: %w", err)
	}
	if genResp == nil || genResp.Message == nil {
		return nil, fmt.Errorf("critic returned empty response")
	}

	feedback, err := decodeJSON[CriticFeedback](genResp.Message.Text())
	if err != nil {
		return &CriticFeedback{
			Verdict:     "approve",
			Notes:       fmt.Sprintf("critic output parse error: %v", err),
			FinalAnswer: draft,
		}, nil
	}

	if feedback.Verdict == "" {
		feedback.Verdict = "approve"
	}
	if feedback.FinalAnswer == "" {
		feedback.FinalAnswer = draft
	}

	return feedback, nil
}
