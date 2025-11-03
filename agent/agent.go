package agent

import (
	"context"
	"fmt"

	agentContext "github.com/sweetpotato0/ai-allin/context"
	"github.com/sweetpotato0/ai-allin/memory"
	"github.com/sweetpotato0/ai-allin/message"
	"github.com/sweetpotato0/ai-allin/prompt"
	"github.com/sweetpotato0/ai-allin/tool"
)

// LLMClient defines the interface for LLM providers
type LLMClient interface {
	// Generate generates a response from the LLM
	Generate(ctx context.Context, messages []*message.Message, tools []map[string]interface{}) (*message.Message, error)
}

// Agent represents an AI agent
type Agent struct {
	name          string
	systemPrompt  string
	maxIterations int
	temperature   float64
	enableMemory  bool
	enableTools   bool
	llm           LLMClient
	tools         *tool.Registry
	memory        memory.MemoryStore
	promptManager *prompt.Manager
	ctx           *agentContext.Context
}

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
	}

	// Apply options
	for _, opt := range opts {
		opt(agent)
	}

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

// Run executes the agent with the given input
func (a *Agent) Run(ctx context.Context, input string) (string, error) {
	// Add user message
	userMsg := message.NewMessage(message.RoleUser, input)
	a.AddMessage(userMsg)

	// Search relevant memories if enabled
	if a.enableMemory && a.memory != nil {
		memories, err := a.memory.SearchMemory(ctx, input)
		if err == nil && len(memories) > 0 {
			// Add memories as context (simplified)
			memoryContext := "Relevant memories:\n"
			for _, mem := range memories {
				memoryContext += fmt.Sprintf("- %v\n", mem)
			}
			contextMsg := message.NewMessage(message.RoleSystem, memoryContext)
			a.ctx.AddMessage(contextMsg)
		}
	}

	// Execution loop with tool calls
	for i := 0; i < a.maxIterations; i++ {
		// Get tool schemas if enabled
		var toolSchemas []map[string]interface{}
		if a.enableTools {
			toolSchemas = a.tools.ToJSONSchemas()
		}

		// Call LLM
		response, err := a.llm.Generate(ctx, a.ctx.GetMessages(), toolSchemas)
		if err != nil {
			return "", fmt.Errorf("LLM generation failed: %w", err)
		}

		a.AddMessage(response)

		// Check if there are tool calls
		if len(response.ToolCalls) == 0 {
			// No tool calls, return the response
			if a.enableMemory && a.memory != nil {
				// Store conversation in memory
				mem := &memory.Memory{}
				a.memory.AddMemory(ctx, mem)
			}
			return response.Content, nil
		}

		// Execute tool calls
		for _, toolCall := range response.ToolCalls {
			result, err := a.tools.Execute(ctx, toolCall.Name, toolCall.Args)
			if err != nil {
				result = fmt.Sprintf("Error executing tool %s: %v", toolCall.Name, err)
			}

			// Add tool response
			toolMsg := message.NewToolResponseMessage(toolCall.ID, result)
			a.AddMessage(toolMsg)
		}

		// Continue loop to get final response
	}

	return "", fmt.Errorf("max iterations (%d) reached", a.maxIterations)
}

// Stream executes the agent with streaming responses
func (a *Agent) Stream(ctx context.Context, input string, callback func(string) error) error {
	// This is a placeholder for streaming implementation
	// In a real implementation, this would stream tokens as they're generated
	result, err := a.Run(ctx, input)
	if err != nil {
		return err
	}
	return callback(result)
}

// Clone creates a copy of the agent with the same configuration
func (a *Agent) Clone() *Agent {
	return New(
		WithName(a.name),
		WithSystemPrompt(a.systemPrompt),
		WithMaxIterations(a.maxIterations),
		WithTemperature(a.temperature),
		WithProvider(a.llm),
		WithTools(a.enableTools),
	)
}
