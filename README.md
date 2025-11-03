# AI-Allin

A comprehensive Go-based AI Agent framework that provides modular architecture for building intelligent agents with context management, tool integration, LLM provider support, and execution workflows.

## Features

- **Modular Architecture**: Clean separation of concerns with well-defined interfaces
- **Message Management**: Support for different message roles (user, assistant, system, tool)
- **Context Management**: Automatic conversation history management with automatic trimming and system message preservation
- **Tool System**: Flexible tool registration and execution with parameter validation
- **Prompt Templates**: Template management with variable substitution
- **Graph Workflows**: Build complex execution flows with conditional nodes
- **Session Management**: Multi-session support with both in-memory and Redis backends
- **Memory Store**: Persistent memory storage with pluggable backends
- **Concurrent Execution**: Parallel, sequential, and conditional task execution
- **LLM Providers**: Built-in support for OpenAI and Anthropic Claude APIs
- **Options Pattern**: Flexible agent configuration without mutable Config structs
- **Production-Ready**: Thread-safe implementations with proper error handling

## Architecture

```
ai-allin/
├── agent/             # AI agent core with Options pattern
├── context/           # Conversation context management (integrated into Agent)
├── graph/             # Execution flow graph
├── memory/            # Memory storage
│   └── store/         # Storage implementations (in-memory, Redis)
├── message/           # Message structures
├── prompt/            # Prompt template management
├── runner/            # Task execution engines
├── session/           # Session management
│   └── store/         # Session storage implementations
├── tool/              # Tool system
├── contrib/provider/  # LLM provider implementations
│   ├── openai/        # OpenAI provider (openai-go SDK)
│   └── claude/        # Claude provider (anthropic-sdk-go SDK)
└── examples/          # Usage examples
```

## Installation

```bash
go get github.com/sweetpotato0/ai-allin
```

## Quick Start

### Basic Agent with Options Pattern

```go
package main

import (
    "context"
    "github.com/sweetpotato0/ai-allin/agent"
    "github.com/sweetpotato0/ai-allin/message"
)

// Implement LLMClient interface
type MyLLMClient struct{}

func (c *MyLLMClient) Generate(ctx context.Context, messages []*message.Message, tools []map[string]interface{}) (*message.Message, error) {
    // Your LLM implementation here
    return message.NewMessage(message.RoleAssistant, "Response from LLM"), nil
}

func main() {
    ctx := context.Background()
    llm := &MyLLMClient{}

    // Create agent using Options pattern
    ag := agent.New(
        agent.WithName("Assistant"),
        agent.WithSystemPrompt("You are a helpful assistant"),
        agent.WithProvider(llm),
        agent.WithTemperature(0.7),
    )

    // Run agent
    result, err := ag.Run(ctx, "Hello!")
    if err != nil {
        panic(err)
    }
    println(result)
}
```

### Using OpenAI Provider

```go
import (
    "github.com/sweetpotato0/ai-allin/agent"
    "github.com/sweetpotato0/ai-allin/contrib/provider/openai"
)

// Create OpenAI provider
config := openai.DefaultConfig(os.Getenv("OPENAI_API_KEY"))
config.Temperature = 0.7
config.MaxTokens = 2000
provider := openai.New(config)

// Create agent with provider
ag := agent.New(
    agent.WithProvider(provider),
    agent.WithSystemPrompt("You are a helpful assistant"),
)

result, err := ag.Run(ctx, "Hello!")
```

### Using Claude Provider

```go
import (
    "github.com/sweetpotato0/ai-allin/agent"
    "github.com/sweetpotato0/ai-allin/contrib/provider/claude"
)

// Create Claude provider
config := claude.DefaultConfig(os.Getenv("ANTHROPIC_API_KEY"))
config.Temperature = 0.7
config.MaxTokens = 4096
provider := claude.New(config)

// Create agent with provider
ag := agent.New(
    agent.WithProvider(provider),
    agent.WithSystemPrompt("You are a helpful assistant"),
)

result, err := ag.Run(ctx, "Hello!")
```

### Agent Options

The Agent supports flexible configuration through options:

```go
ag := agent.New(
    agent.WithName("MyAgent"),
    agent.WithSystemPrompt("You are helpful"),
    agent.WithProvider(llmClient),
    agent.WithTemperature(0.5),
    agent.WithMaxIterations(10),
    agent.WithTools(true),
    agent.WithMemory(memoryStore),
)
```

### Register Tools

```go
calculatorTool := &tool.Tool{
    Name:        "calculator",
    Description: "Performs arithmetic operations",
    Parameters: []tool.Parameter{
        {Name: "operation", Type: "string", Required: true},
        {Name: "a", Type: "number", Required: true},
        {Name: "b", Type: "number", Required: true},
    },
    Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
        // Tool implementation
        return "42", nil
    },
}

ag.RegisterTool(calculatorTool)
```

### Context Management

The Context module is integrated into Agent for automatic message history management:

```go
// Agent automatically manages context with:
// - AddMessage(msg) - Add message to conversation
// - GetMessages() - Get all messages
// - GetLastMessage() - Get most recent message
// - Clear() - Clear conversation (preserves system message)

// The context automatically:
// - Trims old messages when exceeding maxSize (default 100)
// - Preserves system messages during trimming
// - Provides message history to LLM
```

### Graph Workflows

```go
g := graph.NewBuilder().
    AddNode("start", graph.NodeTypeStart, func(ctx context.Context, state graph.State) (graph.State, error) {
        state["step"] = 1
        return state, nil
    }).
    AddNode("process", graph.NodeTypeCustom, func(ctx context.Context, state graph.State) (graph.State, error) {
        state["result"] = "processed"
        return state, nil
    }).
    AddNode("end", graph.NodeTypeEnd, nil).
    AddEdge("start", "process").
    AddEdge("process", "end").
    SetStart("start").
    Build()

finalState, err := g.Execute(ctx, make(graph.State))
```

### Session Management

```go
// In-memory sessions
mgr := session.NewManager()
sess, err := mgr.Create("session-1", ag)
result, err := sess.Run(ctx, "Hello!")

// Redis-backed sessions
redisConfig := &store.RedisConfig{
    Addr:   "localhost:6379",
    Prefix: "ai-allin:session:",
    TTL:    24 * time.Hour,
}
redisMgr := store.NewRedisManager(redisConfig)
```

### Memory Storage

```go
// In-memory storage
memStore := store.NewInMemoryStore()
ag.SetMemory(memStore)

// Redis storage
redisMemStore := store.NewRedisStore(&store.RedisConfig{
    Addr:   "localhost:6379",
    Prefix: "ai-allin:memory:",
})
ag.SetMemory(redisMemStore)
```

### Parallel Execution

```go
tasks := []*runner.Task{
    {ID: "task-1", Agent: agent1, Input: "Input 1"},
    {ID: "task-2", Agent: agent2, Input: "Input 2"},
}

pr := runner.NewParallelRunner(5)
results := pr.RunParallel(ctx, tasks)
```

## Storage Backends

### In-Memory
- Fast, suitable for development and testing
- No external dependencies
- Data lost on restart

### Redis
- Persistent storage
- Distributed session support
- Scalable for production use

```go
// Redis configuration
config := &store.RedisConfig{
    Addr:     "localhost:6379",
    Password: "",              // Optional
    DB:       0,               // Redis database
    Prefix:   "ai-allin:",     // Key prefix
    TTL:      24 * time.Hour,  // Expiration time
}
```

## Testing

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./tool
go test ./message

# Run with coverage
go test -cover ./...
```

## Examples

Check the [examples](examples/) directory for complete working examples:

- **examples/main.go** - Comprehensive framework demonstration
- **examples/basic/main.go** - Options pattern usage
- **examples/tools/main.go** - Tool registration and execution
- **examples/context/main.go** - Context management
- **examples/graph/main.go** - Graph workflows
- **examples/providers/main.go** - LLM provider usage (OpenAI, Claude)

## Development

### Building

```bash
go build ./...
```

### Running Examples

```bash
# All-in-one example
go run examples/main.go

# Individual examples
go run examples/basic/main.go
go run examples/tools/main.go
go run examples/context/main.go
go run examples/graph/main.go

# Provider examples (requires API keys)
OPENAI_API_KEY=sk-... go run examples/providers/main.go
ANTHROPIC_API_KEY=sk-... go run examples/providers/main.go
```

## Completed Features

✅ Message structures with roles and tool calls
✅ Context management with automatic history limiting and trimming
✅ Tool registry with parameter validation and JSON Schema generation
✅ Prompt management with templates and builders
✅ Graph execution with conditional nodes and state management
✅ Agent core with Options pattern configuration
✅ Context module integration into Agent
✅ Session management with persistence
✅ In-memory and Redis storage backends
✅ OpenAI provider (official openai-go SDK)
✅ Claude provider (official anthropic-sdk-go SDK)
✅ Parallel, sequential, and conditional task runners
✅ Comprehensive examples

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## License

MIT License

## Roadmap

- [ ] Additional storage backends (PostgreSQL, MongoDB)
- [ ] Streaming support for LLM responses
- [ ] Vector search for memory
- [ ] Middleware support
- [ ] Web dashboard for session management
- [ ] Performance benchmarks
- [ ] Additional LLM provider integrations

## Support

For questions and support, please open an issue on GitHub.
