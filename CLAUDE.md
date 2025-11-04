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
go test ./middleware ./middleware/logger ./middleware/validator
go test ./middleware/errorhandler ./middleware/enricher ./middleware/limiter
go test ./vector/store
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
go run examples/middleware/main.go
go run examples/streaming/main.go
go run examples/allproviders/main.go
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
  - **memory/store/** - Storage backends including in-memory, Redis, PostgreSQL, and MongoDB implementations
- **middleware/** - Extensible request/response processing pipeline
  - **middleware/logger/** - Request/response logging middleware
  - **middleware/validator/** - Input validation and response filtering
  - **middleware/errorhandler/** - Error handling and recovery
  - **middleware/enricher/** - Context metadata enrichment
  - **middleware/limiter/** - Rate limiting
- **vector/** - Vector search and embedding support
  - **vector/store/** - Vector storage backends (in-memory and pgvector)
- **runner/** - Provides task execution engines with support for parallel, sequential, and conditional execution
- **contrib/provider/** - LLM provider implementations
  - **contrib/provider/openai/** - OpenAI API integration using official `openai-go` SDK
  - **contrib/provider/claude/** - Anthropic Claude integration using official `anthropic-sdk-go` SDK
  - **contrib/provider/groq/** - Groq API integration (mixtral-8x7b-32768)
  - **contrib/provider/cohere/** - Cohere API integration for enterprise LLM
  - **contrib/provider/gemini/** - Google Gemini integration

### Design Patterns

The codebase follows Go interface-based design with the following key patterns:

1. **Options Pattern**: Agent configuration through functional options without mutable Config structs
2. **Interface Segregation**: Core functionality defined through minimal interfaces (e.g., `LLMClient`, `MemoryStore`, `Session`)
3. **Storage Abstraction**: Pluggable storage backends for both memory and sessions
4. **Builder Pattern**: Fluent APIs for constructing complex objects like graphs and prompts
5. **Registry Pattern**: Tool and template registration with validation
6. **Strategy Pattern**: Different execution strategies (parallel, sequential, conditional) in runners

### Storage Backends

The framework supports multiple storage backends for both memory and sessions:

### Memory Storage

- **In-Memory**: Fast, suitable for development/testing (no external dependencies)
- **Redis**: Persistent, distributed, production-ready storage
- **PostgreSQL**: Full SQL database support with CRUD operations and JSON metadata
- **MongoDB**: Document-based storage with regex search support

#### Adding a New Storage Backend

1. Implement the `MemoryStore` interface in `memory/store/`
2. Implement session storage in `session/store/` (if needed)
3. Add appropriate configuration structures and default configs

### Vector Storage

- **In-Memory**: Thread-safe vector storage with cosine similarity and Euclidean distance calculations
- **PostgreSQL pgvector**: Scalable vector storage using PostgreSQL's pgvector extension with HNSW or IVFFLAT indexing

#### Using Vector Search

```go
import "github.com/sweetpotato0/ai-allin/vector/store"

// Create in-memory vector store
vectorStore := store.NewInMemoryVectorStore()

// Add embedding
embedding := &vector.Embedding{
    ID:     "doc1",
    Text:   "Your text here",
    Vector: []float32{0.1, 0.2, 0.3, ...},
}
vectorStore.AddEmbedding(ctx, embedding)

// Search for similar vectors
queryVector := []float32{0.15, 0.25, 0.35, ...}
results, err := vectorStore.Search(ctx, queryVector, 10) // Top 10 results
```

## Middleware System Details

The middleware system is organized into specialized packages for better modularity:

### Package Organization

- **middleware/middleware.go** - Core interfaces and chain orchestration
- **middleware/logger/** - Request/response logging
- **middleware/validator/** - Input validation and response filtering
- **middleware/errorhandler/** - Error handling and recovery
- **middleware/enricher/** - Context metadata enrichment
- **middleware/limiter/** - Rate limiting

### Advanced Middleware Usage

```go
ag := agent.New(
    agent.WithProvider(llm),
    // Add multiple middlewares in order
    agent.WithMiddleware(logger.NewRequestLogger(func(msg string) {
        log.Println(msg)
    })),
    agent.WithMiddleware(validator.NewInputValidator(func(input string) error {
        if len(input) > 1000 {
            return fmt.Errorf("input too long")
        }
        return nil
    })),
    agent.WithMiddleware(limiter.NewRateLimiter(100)), // Max 100 requests
    agent.WithMiddleware(errorhandler.NewErrorHandler(func(err error) error {
        log.Printf("Error: %v\n", err)
        return nil // Continue processing
    })),
)
```

### Creating Custom Middleware

```go
type CustomMiddleware struct {
    processor func(*middleware.Context) error
}

func (m *CustomMiddleware) Name() string {
    return "custom-middleware"
}

func (m *CustomMiddleware) Execute(ctx *middleware.Context, next middleware.Handler) error {
    // Pre-processing
    if err := m.processor(ctx); err != nil {
        return err
    }

    // Call next middleware
    if err := next(ctx); err != nil {
        return err
    }

    // Post-processing
    return nil
}
```

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
    SetTemperature(temp float64)
    SetMaxTokens(max int64)
    SetModel(model string)
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

#### Groq Provider

```go
import "github.com/sweetpotato0/ai-allin/contrib/provider/groq"

config := groq.DefaultConfig(apiKey)
config.Model = "mixtral-8x7b-32768"  // Fast inference
provider := groq.New(config)

agent := agent.New(
    agent.WithProvider(provider),
    agent.WithSystemPrompt("You are helpful"),
)
```

#### Cohere Provider

```go
import "github.com/sweetpotato0/ai-allin/contrib/provider/cohere"

config := cohere.DefaultConfig(apiKey)
config.Model = "command"
provider := cohere.New(config)

agent := agent.New(
    agent.WithProvider(provider),
    agent.WithSystemPrompt("You are helpful"),
)
```

#### Gemini Provider

```go
import "github.com/sweetpotato0/ai-allin/contrib/provider/gemini"

config := gemini.DefaultConfig(apiKey)
config.Model = "gemini-pro"
provider := gemini.New(config)

agent := agent.New(
    agent.WithProvider(provider),
    agent.WithSystemPrompt("You are helpful"),
)
```

All providers support tool calling, configuration methods, and are production-ready. You can dynamically switch providers by updating the provider configuration:

```go
provider.SetTemperature(0.9)
provider.SetMaxTokens(1024)
provider.SetModel("different-model")
```

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
✅ PostgreSQL storage backend with full CRUD operations
✅ MongoDB storage backend with document-based storage
✅ OpenAI provider (using official openai-go SDK)
✅ Claude provider (using official anthropic-sdk-go SDK)
✅ Groq provider for fast inference (mixtral-8x7b-32768)
✅ Cohere provider for enterprise LLM integration
✅ Gemini provider for Google's generative AI
✅ Parallel, sequential, and conditional task runners
✅ Comprehensive examples demonstrating all features
✅ Streaming LLM response support
✅ Middleware support for extensible request/response processing
✅ Vector search functionality with cosine similarity and Euclidean distance
✅ In-memory vector storage with TopK similarity search
✅ PostgreSQL pgvector storage for scalable vector operations
✅ LLMClient interface with SetTemperature, SetMaxTokens, SetModel methods

## Middleware System

The framework includes a flexible middleware system for request/response processing:

### Middleware Interface

```go
type Middleware interface {
    Name() string
    Execute(ctx *Context, next Handler) error
}

type Handler func(*Context) error
```

### Built-in Middleware

1. **RequestLogger** - Logs incoming requests
2. **ResponseLogger** - Logs outgoing responses
3. **InputValidator** - Validates and cleans input
4. **ResponseFilter** - Filters or transforms responses
5. **ContextEnricher** - Adds metadata to context
6. **ErrorHandler** - Handles errors in the pipeline
7. **RateLimiter** - Rate limiting support

### Usage Example

```go
ag := agent.New(
    agent.WithProvider(llm),
    agent.WithMiddleware(middleware.NewRequestLogger(func(msg string) {
        fmt.Println(msg)
    })),
    agent.WithMiddleware(middleware.NewInputValidator(func(input string) error {
        if len(input) > 1000 {
            return errors.New("input too long")
        }
        return nil
    })),
)
```

### Middleware Chain Execution

Middlewares are executed in order, with each middleware able to:
- Inspect or modify the request context
- Perform pre-processing before LLM call
- Perform post-processing after LLM response
- Stop execution and return an error
- Pass control to the next middleware

## Future Enhancements

- Embedding service integration (OpenAI embeddings, Cohere embeddings, etc.)
- Additional storage backends (Elasticsearch, Milvus, Weaviate)
- Advanced middleware patterns (caching, retry logic, circuit breaker)
- Tool calling improvements and extensions
- Performance optimization and benchmarking
- Distributed agent support for multi-agent systems
- Web UI for agent management and monitoring
- Plugin system for custom extensions
- GraphQL API for agent interaction

