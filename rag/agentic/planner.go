package agentic

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/message"
)

type planner struct {
	llm       agent.LLMClient
	prompt    string
	maxSteps  int
	agentName string
}

func newPlanner(llm agent.LLMClient, cfg *Config) *planner {
	return &planner{
		llm:       llm,
		prompt:    cfg.PlannerPrompt,
		maxSteps:  cfg.MaxPlanSteps,
		agentName: cfg.Name + "-planner",
	}
}

func (p *planner) Plan(ctx context.Context, question string) (*Plan, error) {
	if p.llm == nil {
		return nil, fmt.Errorf("planner LLM is not configured")
	}

	systemPrompt := strings.ReplaceAll(p.prompt, "{{max_steps}}", strconv.Itoa(p.maxSteps))
	messages := []*message.Message{
		message.NewMessage(message.RoleSystem, systemPrompt),
		message.NewMessage(message.RoleUser, fmt.Sprintf("User question: %s\nReturn JSON only.", question)),
	}

	genResp, err := p.llm.Generate(ctx, &agent.GenerateRequest{
		Messages: messages,
	})
	if err != nil {
		return nil, fmt.Errorf("planner generation failed: %w", err)
	}
	if genResp == nil || genResp.Message == nil {
		return nil, fmt.Errorf("planner returned empty response")
	}

	plan, err := decodeJSON[Plan](genResp.Message.Text())
	if err != nil {
		return nil, fmt.Errorf("planner output invalid: %w", err)
	}

	if len(plan.Steps) == 0 {
		return nil, fmt.Errorf("planner produced no steps")
	}

	if len(plan.Steps) > p.maxSteps {
		plan.Steps = plan.Steps[:p.maxSteps]
	}

	for idx := range plan.Steps {
		if plan.Steps[idx].ID == "" {
			plan.Steps[idx].ID = fmt.Sprintf("step-%d", idx+1)
		}
	}

	return plan, nil
}
