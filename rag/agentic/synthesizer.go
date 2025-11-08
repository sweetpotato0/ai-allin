package agentic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/message"
)

type synthesizer struct {
	llm    agent.LLMClient
	prompt string
}

func newSynthesizer(llm agent.LLMClient, cfg *Config) *synthesizer {
	return &synthesizer{
		llm:    llm,
		prompt: cfg.SynthesisPrompt,
	}
}

func (s *synthesizer) Compose(ctx context.Context, question string, plan *Plan, evidence []Evidence) (string, error) {
	if s.llm == nil {
		return "", fmt.Errorf("synthesizer LLM is not configured")
	}

	var planJSON string
	if plan != nil {
		if data, err := json.Marshal(plan); err == nil {
			planJSON = string(data)
		}
	}

	contextBlock := formatEvidence(evidence)
	userPrompt := fmt.Sprintf("Question:\n%s\n\nPlan:\n%s\n\nEvidence:\n%s", question, planJSON, contextBlock)

	msgs := []*message.Message{
		message.NewMessage(message.RoleSystem, s.prompt),
		message.NewMessage(message.RoleUser, userPrompt),
	}

	resp, err := s.llm.Generate(ctx, msgs, nil)
	if err != nil {
		return "", fmt.Errorf("synthesizer failed: %w", err)
	}

	return strings.TrimSpace(resp.Content), nil
}

func formatEvidence(evidence []Evidence) string {
	if len(evidence) == 0 {
		return "No external context was retrieved."
	}
	var b strings.Builder
	for _, ev := range evidence {
		var title string
		if ev.Document != nil {
			title = ev.Document.Title
			if title == "" {
				title = ev.Document.ID
			}
		}
		if title == "" {
			title = ev.Chunk.ID
		}
		fmt.Fprintf(&b, "[Doc:%s Step:%s Score:%.2f]\n%s\n\n%s\n---\n", ev.Chunk.DocumentID, ev.StepID, ev.Score, title, ev.Chunk.Content)
	}
	return b.String()
}
