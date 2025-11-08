package runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/message"
)

// Request captures the inputs required to execute a turn.
type Request struct {
	SessionID string
	Input     string
	History   []*message.Message
	Metadata  map[string]any
}

// TurnResult captures the outcome of a single executor run.
type TurnResult struct {
	SessionID   string
	Output      string
	Messages    []*message.Message
	LastMessage *message.Message
	Duration    time.Duration
}

// Executor defines the contract for runtime executors.
type Executor interface {
	Execute(ctx context.Context, req *Request) (*TurnResult, error)
}

// AgentExecutor wraps an agent.Agent and exposes it through the Executor interface.
type AgentExecutor struct {
	prototype *agent.Agent
}

// NewAgentExecutor constructs a new runtime executor backed by a prototype agent.
func NewAgentExecutor(prototype *agent.Agent) *AgentExecutor {
	if prototype == nil {
		panic("runtime: agent prototype cannot be nil")
	}
	return &AgentExecutor{prototype: prototype}
}

// Execute runs the underlying agent using the provided request and conversation history.
func (e *AgentExecutor) Execute(ctx context.Context, req *Request) (*TurnResult, error) {
	if req == nil {
		return nil, fmt.Errorf("runtime: request cannot be nil")
	}
	if req.Input == "" {
		return nil, fmt.Errorf("runtime: input cannot be empty")
	}

	runner := e.prototype.Clone()
	if len(req.History) > 0 {
		runner.RestoreMessages(req.History)
	}

	start := time.Now()
	output, err := runner.Run(ctx, req.Input)
	if err != nil {
		return nil, err
	}
	duration := time.Since(start)

	messages := message.CloneMessages(runner.GetMessages())
	var last *message.Message
	if len(messages) > 0 {
		last = message.Clone(messages[len(messages)-1])
	}

	return &TurnResult{
		SessionID:   req.SessionID,
		Output:      output,
		Messages:    messages,
		LastMessage: last,
		Duration:    duration,
	}, nil
}
