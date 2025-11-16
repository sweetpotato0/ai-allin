package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	agentContext "github.com/sweetpotato0/ai-allin/context"
	"github.com/sweetpotato0/ai-allin/memory"
	"github.com/sweetpotato0/ai-allin/message"
	"github.com/sweetpotato0/ai-allin/middleware"
	"github.com/sweetpotato0/ai-allin/pkg/logging"
	"github.com/sweetpotato0/ai-allin/pkg/telemetry"
	"github.com/sweetpotato0/ai-allin/prompt"
	runtimeprovider "github.com/sweetpotato0/ai-allin/runtime/provider"
	"github.com/sweetpotato0/ai-allin/tool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// LLMClient defines the interface for LLM providers
type LLMClient interface {
	// Generate generates a response from the LLM
	Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)

	// SetTemperature updates the temperature setting for generation
	SetTemperature(temp float64)

	// SetMaxTokens updates the maximum tokens limit for generation
	SetMaxTokens(max int64)

	// SetModel updates the model to use for generation
	SetModel(model string)
}

// Agent represents an AI agent
type Agent struct {
	name           string
	systemPrompt   string
	maxIterations  int
	temperature    float64
	enableMemory   bool
	enableTools    bool
	llm            LLMClient
	tools          *tool.Registry
	memory         memory.MemoryStore
	promptManager  *prompt.Manager
	ctx            *agentContext.Context
	middlewares    *middleware.MiddlewareChain
	toolSupervisor *runtimeprovider.ToolSupervisor
	logger         *slog.Logger
}

var agentTracer = otel.Tracer("github.com/sweetpotato0/ai-allin/agent")

// Option is a function that configures an Agent
type Option func(*Agent)

// WithName sets the agent name
func WithName(name string) Option {
	return func(a *Agent) {
		a.name = name
	}
}

// WithSystemPrompt sets the system prompt
func WithSystemPrompt(prompt string) Option {
	return func(a *Agent) {
		a.systemPrompt = prompt
	}
}

// WithMaxIterations sets the maximum iterations for tool calling
func WithMaxIterations(max int) Option {
	return func(a *Agent) {
		a.maxIterations = max
	}
}

// WithTemperature sets the temperature for LLM generation
func WithTemperature(temp float64) Option {
	return func(a *Agent) {
		a.temperature = temp
	}
}

// WithMemory enables memory and sets the memory store
func WithMemory(store memory.MemoryStore) Option {
	return func(a *Agent) {
		a.memory = store
		a.enableMemory = true
	}
}

// WithTools enables or disables tool usage
func WithTools(enable bool) Option {
	return func(a *Agent) {
		a.enableTools = enable
	}
}

// WithProvider sets the LLM provider
func WithProvider(provider LLMClient) Option {
	return func(a *Agent) {
		a.llm = provider
	}
}

// WithToolProvider registers a tool provider that will supply tools on demand.
func WithToolProvider(provider tool.Provider) Option {
	return func(a *Agent) {
		if provider == nil {
			return
		}
		if a.toolSupervisor == nil {
			a.toolSupervisor = runtimeprovider.NewToolSupervisor(a.tools, runtimeprovider.WithErrorHandler(a.reportToolError))
		}
		a.toolSupervisor.Register(provider)
	}
}

// WithMiddleware adds a middleware to the agent
func WithMiddleware(m middleware.Middleware) Option {
	return func(a *Agent) {
		a.middlewares.Add(m)
	}
}

// WithMiddlewares sets the middleware chain
func WithMiddlewares(middlewares ...middleware.Middleware) Option {
	return func(a *Agent) {
		a.middlewares = middleware.NewChain(middlewares...)
	}
}

// WithLogger overrides the logger used by the agent.
func WithLogger(logger *slog.Logger) Option {
	return func(a *Agent) {
		if logger != nil {
			a.logger = logger
		}
	}
}

func (a *Agent) ensureToolProviders(ctx context.Context) error {
	if !a.enableTools || a.toolSupervisor == nil {
		return nil
	}
	if a.logger != nil {
		a.logger.Debug("refreshing tool providers")
	}
	if err := a.toolSupervisor.Refresh(ctx); err != nil {
		if a.logger != nil {
			a.logger.Error("tool provider refresh failed", "error", err)
		}
		return err
	}
	if a.logger != nil {
		a.logger.Debug("tool providers refreshed")
	}
	return nil
}

func (a *Agent) reportToolError(err error) {
	if err == nil || a.ctx == nil {
		return
	}
	if a.logger != nil {
		a.logger.Error("tool provider error", "error", err)
	}
	a.ctx.AddMessage(message.NewMessage(message.RoleSystem, fmt.Sprintf("Failed to refresh tools: %v", err)))
}

// New creates a new agent with the given options
func New(opts ...Option) *Agent {
	// Default values
	agent := &Agent{
		name:          "Agent",
		systemPrompt:  "You are a helpful AI assistant.",
		maxIterations: 10,
		temperature:   0.7,
		enableMemory:  false,
		enableTools:   true,
		tools:         tool.NewRegistry(),
		promptManager: prompt.NewManager(),
		ctx:           agentContext.New(),
		middlewares:   middleware.NewChain(),
	}
	agent.toolSupervisor = runtimeprovider.NewToolSupervisor(agent.tools, runtimeprovider.WithErrorHandler(agent.reportToolError))

	// Apply options
	for _, opt := range opts {
		opt(agent)
	}

	if agent.logger == nil {
		agent.logger = logging.WithComponent("agent")
	}
	agent.logger = agent.logger.With("agent", agent.name)

	// Add system prompt as first message if set
	if agent.systemPrompt != "" {
		agent.ctx.AddMessage(message.NewMessage(message.RoleSystem, agent.systemPrompt))
	}

	return agent
}

// SetMemory sets the memory store
func (a *Agent) SetMemory(mem memory.MemoryStore) {
	a.memory = mem
	a.enableMemory = true
}

// RegisterTool registers a tool with the agent
func (a *Agent) RegisterTool(t *tool.Tool) error {
	return a.tools.Register(t)
}

// RegisterPrompt registers a prompt template
func (a *Agent) RegisterPrompt(name, content string) error {
	return a.promptManager.RegisterString(name, content)
}

// AddMiddleware adds a middleware to the agent with validation
func (a *Agent) AddMiddleware(m middleware.Middleware) error {
	if m == nil {
		return fmt.Errorf("middleware cannot be nil")
	}
	a.middlewares.Add(m)
	return nil
}

// GetMiddlewareChain returns the middleware chain
func (a *Agent) GetMiddlewareChain() *middleware.MiddlewareChain {
	return a.middlewares
}

// AddMessage adds a message to the conversation
func (a *Agent) AddMessage(msg *message.Message) {
	a.ctx.AddMessage(msg)
}

// GetMessages returns all messages
func (a *Agent) GetMessages() []*message.Message {
	return a.ctx.GetMessages()
}

// ClearMessages clears all messages except system messages
func (a *Agent) ClearMessages() {
	a.ctx.Clear()
	// Re-add system prompt
	if a.systemPrompt != "" {
		a.ctx.AddMessage(message.NewMessage(message.RoleSystem, a.systemPrompt))
	}
}

// RestoreMessages replaces the current conversation history with the provided messages.
// System prompts should be included in the provided slice; when the slice is empty
// the agent falls back to the default system prompt.
func (a *Agent) RestoreMessages(messages []*message.Message) {
	a.ctx.Clear()
	if len(messages) == 0 {
		if a.systemPrompt != "" {
			a.ctx.AddMessage(message.NewMessage(message.RoleSystem, a.systemPrompt))
		}
		return
	}
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		a.ctx.AddMessage(message.Clone(msg))
	}
}

// Run executes the agent with the given input
func (a *Agent) Run(ctx context.Context, input string) (*message.Message, error) {
	ctx, span := agentTracer.Start(ctx, "Agent.Run",
		oteltrace.WithAttributes(
			attribute.String("agent.name", a.name),
			attribute.Int("agent.max_iterations", a.maxIterations),
			attribute.String("agent.input_preview", trimLogText(input, 96)),
		))
	var spanErr error
	defer func() { telemetry.End(span, spanErr) }()

	if a.logger != nil {
		a.logger.Info("agent run started", "input", trimLogText(input, 160))
	}
	if err := a.ensureToolProviders(ctx); err != nil {
		if a.logger != nil {
			a.logger.Error("tool provider refresh failed", "error", err)
		}
		spanErr = err
		return nil, err
	}

	mwCtx := middleware.NewContext(ctx)
	mwCtx.Input = input

	err := a.middlewares.Execute(mwCtx, func(mwCtx *middleware.Context) error {
		userMsg := message.NewMessage(message.RoleUser, input)
		a.AddMessage(userMsg)
		mwCtx.Messages = a.GetMessages()

		if a.enableMemory && a.memory != nil {
			memories, err := a.memory.SearchMemory(mwCtx.Context(), input)
			if err == nil && len(memories) > 0 {
				if a.logger != nil {
					a.logger.Debug("memory hits found", "count", len(memories))
				}
				span.AddEvent("memory_hits", oteltrace.WithAttributes(attribute.Int("count", len(memories))))
				memoryContext := "Relevant memories:\n"
				for _, mem := range memories {
					memoryContext += fmt.Sprintf("- %v\n", mem)
				}
				contextMsg := message.NewMessage(message.RoleSystem, memoryContext)
				a.ctx.AddMessage(contextMsg)
			} else if err != nil {
				if a.logger != nil {
					a.logger.Warn("memory search failed", "error", err)
				}
				span.AddEvent("memory_search_failed", oteltrace.WithAttributes(attribute.String("error", err.Error())))
			}
		}

		for i := 0; i < a.maxIterations; i++ {
			if a.logger != nil {
				a.logger.Debug("llm turn started", "iteration", i+1)
			}
			span.AddEvent("agent_iteration", oteltrace.WithAttributes(attribute.Int("iteration", i+1)))

			var toolSchemas []map[string]any
			if a.enableTools {
				toolSchemas = a.tools.ToJSONSchemas()
				if a.logger != nil {
					a.logger.Debug("tools available", "count", len(toolSchemas))
				}
			}

			req := &GenerateRequest{
				Messages: a.ctx.GetMessages(),
				Tools:    toolSchemas,
			}
			resp, err := a.llm.Generate(mwCtx.Context(), req)
			if err != nil {
				if a.logger != nil {
					a.logger.Error("llm generation failed", "iteration", i+1, "error", err)
				}
				return fmt.Errorf("LLM generation failed: %w", err)
			}

			a.AddMessage(resp.Message)
			mwCtx.Response = resp.Message

			if len(resp.Message.ToolCalls) == 0 {
				if a.enableMemory && a.memory != nil {
					conversationContent := fmt.Sprintf("User: %s\nAssistant: %s", input, resp.Message.Text())
					mem := &memory.Memory{
						ID:       memory.GenerateMemoryID(),
						Content:  conversationContent,
						Metadata: map[string]any{"input": input, "response": resp.Message.Text()},
					}
					a.memory.AddMemory(mwCtx.Context(), mem)
				}
				if a.logger != nil {
					a.logger.Info("agent run completed without tool calls", "iteration", i+1)
				}
				return nil
			}

			for _, toolCall := range resp.Message.ToolCalls {
				if a.logger != nil {
					a.logger.Info("executing tool call", "tool", toolCall.Name)
				}
				span.AddEvent("tool_call", oteltrace.WithAttributes(attribute.String("tool.name", toolCall.Name)))
				result, err := a.tools.Execute(mwCtx.Context(), toolCall.Name, toolCall.Args)
				if err != nil {
					if a.logger != nil {
						a.logger.Error("tool execution failed", "tool", toolCall.Name, "error", err)
					}
					span.AddEvent("tool_error",
						oteltrace.WithAttributes(
							attribute.String("tool.name", toolCall.Name),
							attribute.String("error", err.Error()),
						))
					result = fmt.Sprintf("Error executing tool %s: %v", toolCall.Name, err)
				}

				toolMsg := message.NewToolResponseMessage(toolCall.ID, result)
				a.AddMessage(toolMsg)
			}
		}

		mwCtx.Error = fmt.Errorf("max iterations (%d) reached", a.maxIterations)
		return mwCtx.Error
	})

	if err != nil {
		if a.logger != nil {
			a.logger.Error("agent run failed", "error", err)
		}
		spanErr = err
		return nil, err
	}

	if mwCtx.Response != nil {
		if a.logger != nil {
			a.logger.Info("agent run completed", "output", trimLogText(mwCtx.Response.Text(), 160))
		}
		return mwCtx.Response, nil
	}

	if a.logger != nil {
		a.logger.Error("agent run ended without response")
	}
	spanErr = fmt.Errorf("no response generated")
	return nil, spanErr
}

// Stream executes the agent with streaming responses
func (a *Agent) Stream(ctx context.Context, input string, callback func(*message.Message) error) error {
	// This is a placeholder for streaming implementation
	// In a real implementation, this would stream tokens as they're generated
	ctx, span := agentTracer.Start(ctx, "Agent.Stream",
		oteltrace.WithAttributes(
			attribute.String("agent.name", a.name),
			attribute.String("input_preview", trimLogText(input, 96)),
		))
	var spanErr error
	defer func() { telemetry.End(span, spanErr) }()
	if a.logger != nil {
		a.logger.Info("agent stream started", "input", trimLogText(input, 160))
	}
	result, err := a.Run(ctx, input)
	if err != nil {
		if a.logger != nil {
			a.logger.Error("agent stream failed", "error", err)
		}
		spanErr = err
		return err
	}
	if a.logger != nil {
		a.logger.Info("agent stream callback", "output", trimLogText(result.Text(), 160))
	}
	if err := callback(result); err != nil {
		spanErr = err
		return err
	}
	return nil
}

// Clone creates a copy of the agent with the same configuration
func (a *Agent) Clone() *Agent {
	cloned := New(
		WithName(a.name),
		WithSystemPrompt(a.systemPrompt),
		WithMaxIterations(a.maxIterations),
		WithTemperature(a.temperature),
		WithProvider(a.llm),
		WithTools(a.enableTools),
		WithLogger(a.logger),
	)

	// Clone memory store if set
	if a.memory != nil {
		cloned.memory = a.memory
		cloned.enableMemory = a.enableMemory
	}

	// Clone all registered tools
	for _, tool := range a.tools.List() {
		if tool != nil {
			_ = cloned.tools.Register(tool)
		}
	}

	// Clone all registered prompts
	if a.promptManager != nil {
		cloned.promptManager = a.promptManager // Share prompt manager
	}

	// Clone middleware chain
	if a.middlewares != nil {
		cloned.middlewares = middleware.NewChain(a.middlewares.List()...)
	}

	if a.toolSupervisor != nil && cloned.toolSupervisor != nil {
		for _, provider := range a.toolSupervisor.Providers() {
			cloned.toolSupervisor.Register(provider)
		}
	}

	return cloned
}

func trimLogText(text string, limit int) string {
	text = strings.TrimSpace(text)
	if limit <= 0 || len([]rune(text)) <= limit {
		return text
	}
	return string([]rune(text)[:limit]) + "..."
}
