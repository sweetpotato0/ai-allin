package runtime

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/message"
	"github.com/sweetpotato0/ai-allin/pkg/logging"
	"github.com/sweetpotato0/ai-allin/pkg/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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
	logger    *slog.Logger
}

var executorTracer = otel.Tracer("github.com/sweetpotato0/ai-allin/runtime/executor")

// NewAgentExecutor constructs a new runtime executor backed by a prototype agent.
func NewAgentExecutor(prototype *agent.Agent) *AgentExecutor {
	if prototype == nil {
		panic("runtime: agent prototype cannot be nil")
	}
	return &AgentExecutor{
		prototype: prototype,
		logger:    logging.WithComponent("executor").With("executor", "agent"),
	}
}

// Execute runs the underlying agent using the provided request and conversation history.
func (e *AgentExecutor) Execute(ctx context.Context, req *Request) (*TurnResult, error) {
	ctx, span := executorTracer.Start(ctx, "AgentExecutor.Execute")
	var spanErr error
	defer func() { telemetry.End(span, spanErr) }()
	if req == nil {
		spanErr = fmt.Errorf("runtime: request cannot be nil")
		return nil, spanErr
	}
	if req.Input == "" {
		spanErr = fmt.Errorf("runtime: input cannot be empty")
		return nil, spanErr
	}
	span.SetAttributes(attribute.String("session.id", req.SessionID))

	runner := e.prototype.Clone()
	if len(req.History) > 0 {
		runner.RestoreMessages(req.History)
	}

	if e.logger != nil {
		e.logger.Info("executor running turn", "session_id", req.SessionID, "history", len(req.History))
	}
	start := time.Now()
	output, err := runner.Run(ctx, req.Input)
	if err != nil {
		if e.logger != nil {
			e.logger.Error("executor run failed", "session_id", req.SessionID, "error", err)
		}
		spanErr = err
		return nil, err
	}
	duration := time.Since(start)
	if e.logger != nil {
		e.logger.Info("executor run completed", "session_id", req.SessionID, "duration_ms", duration.Milliseconds())
	}
	span.SetAttributes(attribute.Int64("duration_ms", duration.Milliseconds()))

	messages := message.CloneMessages(runner.GetMessages())
	var last *message.Message
	if len(messages) > 0 {
		last = message.Clone(messages[len(messages)-1])
	}

	return &TurnResult{
		SessionID:   req.SessionID,
		Output:      output.Text(),
		Messages:    messages,
		LastMessage: last,
		Duration:    duration,
	}, nil
}
