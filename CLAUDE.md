# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based AI framework called "ai-allin" that provides a modular architecture for building AI agents with message context management, tool capabilities, execution workflows, and LLM provider integration.

## Build and Development Commands

```bash
# Initialize Go module dependencies
go mod download

# Update dependencies
go mod tidy

# Build the project
go build ./...

# Run tests
go test ./...

# Run tests for a specific package
go test ./memory
go test ./session
go test ./tool
go test ./message

# Run tests with coverage
go test -cover ./...

# Run example code
go run examples/main.go
go run examples/basic/main.go
go run examples/tools/main.go
go run examples/context/main.go
go run examples/graph/main.go
go run examples/providers/main.go
```

## Architecture

The codebase is organized into several core packages that work together to provide an AI agent framework:

### Core Packages

- **message/** - Defines message structures with support for multiple roles (user, assistant, system, tool) and tool calls
- **context/** - Manages conversation context with automatic message history and size limiting (integrated into Agent)
- **tool/** - Implements a flexible tool system with parameter validation, JSON Schema export, and tool registry
- **prompt/** - Provides prompt template management with variable substitution and builders
- **graph/** - Implements execution flow graphs with support for conditional nodes, loops, and state management
- **agent/** - The core AI agent implementation with Options pattern configuration, Context integration, tool calling, and LLM client interface
- **session/** - Manages conversation sessions with support for multiple concurrent sessions
  - **session/store/** - Storage backends for sessions (currently Redis implementation)
- **memory/** - Defines memory storage interface for agent knowledge
  - **memory/store/** - Storage backends including in-memory and Redis implementations
- **runner/** - Provides task execution engines with support for parallel, sequential, and conditional execution
- **contrib/provider/** - LLM provider implementations
  - **contrib/provider/openai/** - OpenAI API integration using official `openai-go` SDK
  - **contrib/provider/claude/** - Anthropic Claude integration using official `anthropic-sdk-go` SDK

### Design Patterns

The codebase follows Go interface-based design with the following key patterns:

1. **Options Pattern**: Agent configuration through functional options without mutable Config structs
2. **Interface Segregation**: Core functionality defined through minimal interfaces (e.g., `LLMClient`, `MemoryStore`, `Session`)
3. **Storage Abstraction**: Pluggable storage backends for both memory and sessions
4. **Builder Pattern**: Fluent APIs for constructing complex objects like graphs and prompts
5. **Registry Pattern**: Tool and template registration with validation
6. **Strategy Pattern**: Different execution strategies (parallel, sequential, conditional) in runners

### Storage Backends

The framework supports multiple storage backends:

- **In-Memory**: Fast, suitable for development/testing (no external dependencies)
- **Redis**: Persistent, distributed, production-ready storage

To add a new storage backend:
1. Implement the `MemoryStore` interface in `memory/store/`
2. Implement session storage in `session/store/`
3. Add appropriate configuration structures

## Key Implementation Notes

- Agent uses Context module for message management with automatic history trimming
- All context operations are thread-safe using `sync.RWMutex`
- Memory and session operations accept `context.Context` for cancellation support
- Tool execution includes parameter validation before handler invocation
- Graph execution includes infinite loop detection (max 100 visits per node)
- Session managers support cleanup of inactive sessions
- Options pattern used for flexible Agent configuration:
  - `WithName()`, `WithSystemPrompt()`, `WithMaxIterations()`, `WithTemperature()`
  - `WithProvider()`, `WithTools()`, `WithMemory()`
- The project uses Go 1.23.1 as specified in [go.mod](go.mod)
- Module path is `github.com/sweetpotato0/ai-allin`

## LLM Integration

To integrate with an LLM provider, implement the `agent.LLMClient` interface:

```go
type LLMClient interface {
    Generate(ctx context.Context, messages []*message.Message, tools []map[string]interface{}) (*message.Message, error)
}
```

### Provided Implementations

#### OpenAI Provider

```go
import "github.com/sweetpotato0/ai-allin/contrib/provider/openai"

config := openai.DefaultConfig(apiKey)
config.Temperature = 0.7
config.MaxTokens = 2000
provider := openai.New(config)

agent := agent.New(
    agent.WithProvider(provider),
    agent.WithSystemPrompt("You are helpful"),
)
```

#### Claude Provider

```go
import "github.com/sweetpotato0/ai-allin/contrib/provider/claude"

config := claude.DefaultConfig(apiKey)
config.Temperature = 0.7
config.MaxTokens = 4096
provider := claude.New(config)

agent := agent.New(
    agent.WithProvider(provider),
    agent.WithSystemPrompt("You are helpful"),
)
```

Both providers support tool calling and are production-ready.

## Agent Usage with Options Pattern

```go
// Create agent with flexible options
ag := agent.New(
    agent.WithName("Assistant"),
    agent.WithSystemPrompt("You are a helpful assistant"),
    agent.WithProvider(llmProvider),
    agent.WithTemperature(0.7),
    agent.WithMaxIterations(10),
    agent.WithTools(true),
    agent.WithMemory(memoryStore),
)

// Run the agent
result, err := ag.Run(ctx, "Your question here")
```

## Context Management

The Context module is integrated into Agent for automatic message history management:

```go
// Context features:
// - AddMessage(msg) - Add a message to conversation
// - GetMessages() - Get all messages
// - GetLastMessage() - Get the most recent message
// - GetMessagesByRole() - Filter messages by role
// - Clear() - Clear all messages (preserves system message in Agent)
// - Size() - Get message count

// Agent's context automatically:
// - Trims old messages when exceeding maxSize (default 100)
// - Preserves system messages during trimming
// - Manages message history for the LLM
```

## Testing Strategy

- Unit tests for core logic in `*_test.go` files
- Mock LLM client for testing agent functionality
- Test coverage for tool validation, message creation, and registry operations
- Integration tests should use in-memory storage backends for speed
- Example files serve as integration tests demonstrating all features

## Examples

All examples are organized in `examples/` directory:

1. **examples/main.go** - Comprehensive framework examples
   - Basic agent usage
   - Agent with tools
   - Graph workflows
   - Session management
   - Parallel execution

2. **examples/basic/main.go** - Options pattern usage
3. **examples/tools/main.go** - Tool registration and execution
4. **examples/context/main.go** - Context module demonstration
5. **examples/graph/main.go** - Graph workflow examples
6. **examples/providers/main.go** - LLM provider usage (OpenAI, Claude)

## Completed Features

✅ Message structures with roles and tool calls
✅ Context management with automatic history limiting
✅ Tool registry with parameter validation and JSON Schema generation
✅ Prompt management with templates and builders
✅ Graph execution with conditional nodes and state management
✅ Agent core with Options pattern configuration
✅ Context module integration into Agent
✅ Session management with persistence
✅ In-memory and Redis storage backends
✅ OpenAI provider (using official openai-go SDK)
✅ Claude provider (using official anthropic-sdk-go SDK)
✅ Parallel, sequential, and conditional task runners
✅ Comprehensive examples demonstrating all features

## Future Enhancements

- Additional storage backends (PostgreSQL, MongoDB)
- Streaming LLM response support
- Vector search for memory
- Middleware support
- Additional LLM provider integrations

