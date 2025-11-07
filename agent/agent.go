package agent

import (
	"context"
	"fmt"
	"sync"

	agentContext "github.com/sweetpotato0/ai-allin/context"
	"github.com/sweetpotato0/ai-allin/memory"
	"github.com/sweetpotato0/ai-allin/message"
	"github.com/sweetpotato0/ai-allin/middleware"
	"github.com/sweetpotato0/ai-allin/prompt"
	"github.com/sweetpotato0/ai-allin/tool"
)

// LLMClient defines the interface for LLM providers
type LLMClient interface {
	// Generate generates a response from the LLM
	Generate(ctx context.Context, messages []*message.Message, tools []map[string]any) (*message.Message, error)

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
	providerMu     sync.Mutex
	toolProviders  []tool.Provider
	providerLoaded map[tool.Provider]bool
	providerWatch  map[tool.Provider]context.CancelFunc
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

// WithToolProvider registers a tool provider that will supply tools on demand.
func WithToolProvider(provider tool.Provider) Option {
	return func(a *Agent) {
		if provider == nil {
			return
		}
		a.providerMu.Lock()
		defer a.providerMu.Unlock()
		a.toolProviders = append(a.toolProviders, provider)
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

func (a *Agent) loadToolProviders(ctx context.Context) error {
	if !a.enableTools {
		return nil
	}

	for _, provider := range a.getToolProviders() {
		if provider == nil {
			continue
		}

		if a.isProviderLoaded(provider) {
			continue
		}

		if err := a.updateProviderTools(ctx, provider); err != nil {
			return err
		}

		a.markProviderLoaded(provider)
		a.startProviderWatcher(provider)
	}

	return nil
}

func (a *Agent) getToolProviders() []tool.Provider {
	a.providerMu.Lock()
	defer a.providerMu.Unlock()
	return append([]tool.Provider(nil), a.toolProviders...)
}

func (a *Agent) isProviderLoaded(provider tool.Provider) bool {
	a.providerMu.Lock()
	defer a.providerMu.Unlock()
	return a.providerLoaded[provider]
}

func (a *Agent) markProviderLoaded(provider tool.Provider) {
	a.providerMu.Lock()
	defer a.providerMu.Unlock()
	if a.providerLoaded == nil {
		a.providerLoaded = make(map[tool.Provider]bool)
	}
	a.providerLoaded[provider] = true
}

func (a *Agent) updateProviderTools(ctx context.Context, provider tool.Provider) error {
	tools, err := provider.Tools(ctx)
	if err != nil {
		return fmt.Errorf("load tools from provider: %w", err)
	}

	for _, t := range tools {
		if t == nil || t.Name == "" {
			continue
		}
		if err := a.tools.Upsert(t); err != nil {
			return err
		}
	}
	return nil
}

func (a *Agent) startProviderWatcher(provider tool.Provider) {
	ch := provider.ToolsChanged()
	if ch == nil {
		return
	}

	a.providerMu.Lock()
	if a.providerWatch == nil {
		a.providerWatch = make(map[tool.Provider]context.CancelFunc)
	}
	if _, exists := a.providerWatch[provider]; exists {
		a.providerMu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.providerWatch[provider] = cancel
	a.providerMu.Unlock()

	go a.watchProvider(ctx, provider, ch)
}

func (a *Agent) watchProvider(ctx context.Context, provider tool.Provider, ch <-chan struct{}) {
	defer a.removeProviderWatcher(provider)

	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-ch:
			if !ok {
				return
			}
			if err := a.updateProviderTools(ctx, provider); err != nil {
				a.ctx.AddMessage(message.NewMessage(message.RoleSystem, fmt.Sprintf("Failed to refresh tools: %v", err)))
			}
		}
	}
}

func (a *Agent) removeProviderWatcher(provider tool.Provider) {
	a.providerMu.Lock()
	defer a.providerMu.Unlock()
	if cancel, ok := a.providerWatch[provider]; ok {
		cancel()
		delete(a.providerWatch, provider)
	}
}

// New creates a new agent with the given options
func New(opts ...Option) *Agent {
	// Default values
	agent := &Agent{
		name:           "Agent",
		systemPrompt:   "You are a helpful AI assistant.",
		maxIterations:  10,
		temperature:    0.7,
		enableMemory:   false,
		enableTools:    true,
		tools:          tool.NewRegistry(),
		promptManager:  prompt.NewManager(),
		ctx:            agentContext.New(),
		middlewares:    middleware.NewChain(),
		providerLoaded: make(map[tool.Provider]bool),
		providerWatch:  make(map[tool.Provider]context.CancelFunc),
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

// Run executes the agent with the given input
func (a *Agent) Run(ctx context.Context, input string) (string, error) {
	if err := a.loadToolProviders(ctx); err != nil {
		return "", err
	}

	// Create middleware context
	mwCtx := middleware.NewContext(ctx)
	mwCtx.Input = input

	// Execute with middleware chain
	err := a.middlewares.Execute(mwCtx, func(mwCtx *middleware.Context) error {
		// Add user message
		userMsg := message.NewMessage(message.RoleUser, input)
		a.AddMessage(userMsg)
		mwCtx.Messages = a.GetMessages()

		// Search relevant memories if enabled
		if a.enableMemory && a.memory != nil {
			memories, err := a.memory.SearchMemory(mwCtx.Context(), input)
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
			var toolSchemas []map[string]any
			if a.enableTools {
				toolSchemas = a.tools.ToJSONSchemas()
			}

			// Call LLM
			response, err := a.llm.Generate(mwCtx.Context(), a.ctx.GetMessages(), toolSchemas)
			if err != nil {
				return fmt.Errorf("LLM generation failed: %w", err)
			}

			a.AddMessage(response)
			mwCtx.Response = response

			// Check if there are tool calls
			if len(response.ToolCalls) == 0 {
				// No tool calls, return the response
				if a.enableMemory && a.memory != nil {
					// Store conversation in memory
					conversationContent := fmt.Sprintf("User: %s\nAssistant: %s", input, response.Content)
					mem := &memory.Memory{
						ID:       memory.GenerateMemoryID(),
						Content:  conversationContent,
						Metadata: map[string]any{"input": input, "response": response.Content},
					}
					a.memory.AddMemory(mwCtx.Context(), mem)
				}
				return nil
			}

			// Execute tool calls
			for _, toolCall := range response.ToolCalls {
				result, err := a.tools.Execute(mwCtx.Context(), toolCall.Name, toolCall.Args)
				if err != nil {
					result = fmt.Sprintf("Error executing tool %s: %v", toolCall.Name, err)
				}

				// Add tool response
				toolMsg := message.NewToolResponseMessage(toolCall.ID, result)
				a.AddMessage(toolMsg)
			}

			// Continue loop to get final response
		}

		mwCtx.Error = fmt.Errorf("max iterations (%d) reached", a.maxIterations)
		return mwCtx.Error
	})

	if err != nil {
		return "", err
	}

	if mwCtx.Response != nil {
		return mwCtx.Response.Content, nil
	}

	return "", fmt.Errorf("no response generated")
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
	cloned := New(
		WithName(a.name),
		WithSystemPrompt(a.systemPrompt),
		WithMaxIterations(a.maxIterations),
		WithTemperature(a.temperature),
		WithProvider(a.llm),
		WithTools(a.enableTools),
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

	if len(a.toolProviders) > 0 {
		cloned.toolProviders = append(cloned.toolProviders, a.toolProviders...)
	}

	return cloned
}
