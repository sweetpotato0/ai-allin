# AI-ALLIN - AI Agent Framework

A comprehensive, production-ready Go framework for building AI agents with streaming support, tool integration, and multi-backend storage.

**[中文文档](./README_ZH.md)**

## Features

- **Multi-Provider LLM Support**: OpenAI, Anthropic Claude, Groq, Cohere, Google Gemini
- **Streaming Response Support**: Real-time streaming for all LLM providers
- **Agent Framework**: Configurable agents with middleware, prompts, and memory
- **Tool Integration**: Register and execute tools/functions
- **Model Context Protocol (MCP) Support**: Connect over stdio or streamable HTTP (SSE), discover MCP tools, and invoke them through agents
- **Multi-Backend Storage**:
  - In-Memory (development)
  - PostgreSQL with full-text search
  - Redis with caching
  - MongoDB document storage
  - PGVector for embeddings
- **Session Management**: Conversation session tracking and management, supporting single-agent and multi-agent shared sessions, with serializable session records for persistence and analytics
- **Runtime Executor Layer**: Agent specs and executors decouple runtime behavior from stored conversations, enabling custom execution strategies and observability
- **Tool Supervisor**: A built-in supervisor keeps tool providers synchronized and refreshes tool schemas automatically
- **Execution Graphs**: Workflow orchestration with conditional branching
- **Agentic RAG**: Multi-agent Retrieval-Augmented Generation pipeline with planners, researchers, writers, and critics wired through `graph.Graph`
- **RAG Building Blocks**: Dedicated packages for documents, chunking, embedders, retrievers, and rerankers to compose custom pipelines
- **Thread-Safe Operations**: RWMutex protected concurrent access
- **Configuration Validation**: Environment-based configuration with validation

## Quick Start

### Installation

```bash
go get github.com/sweetpotato0/ai-allin
```

### Basic Usage

```go
package main

import (
    "context"
    "github.com/sweetpotato0/ai-allin/agent"
    "github.com/sweetpotato0/ai-allin/contrib/provider/openai"
)

func main() {
    // Create LLM provider
    llm := openai.New(&openai.Config{
        APIKey:      "your-api-key",
        Model:       "gpt-4",
        MaxTokens:   2000,
        Temperature: 0.7,
    })

    // Create agent
    ag := agent.New(
        agent.WithName("MyAgent"),
        agent.WithSystemPrompt("You are a helpful assistant"),
        agent.WithProvider(llm),
    )

    // Run agent
    response, err := ag.Run(context.Background(), "What is AI?")
    if err != nil {
        panic(err)
    }

    println(response)
}
```

### Runtime Executor

The `runtime` package lets you decouple persisted conversation state from the live `agent.Agent`. A runtime executor consumes a session transcript (`session.Record`) and produces a `runtime.TurnResult` with timing metadata plus the final assistant message:

```go
exec := runtime.NewAgentExecutor(ag)
result, err := exec.Execute(ctx, &runtime.Request{
    SessionID: "session-1",
    Input:     "What's next?",
    History:   existingMessages,
})
if err != nil {
    log.Fatalf("executor failed: %v", err)
}
fmt.Println("assistant:", result.Output, "took", result.Duration)
```

This is the same executor that backs `session.SingleAgentSession` and `SharedSession`, so you can swap in alternative executors (streaming, tracing, multi-agent) without touching session code.

### Tool Supervisor

Registering a tool provider with `agent.WithToolProvider` now delegates to the runtime tool supervisor. The supervisor loads tools on demand, watches for provider updates, and refreshes the agent's registry automatically:

```go
ag := agent.New(
    agent.WithProvider(llm),
    agent.WithToolProvider(myProvider), // supervisor handles refresh & errors
)
```

You can inspect refresh failures by adding middleware or memory stores—the supervisor pushes errors back into the agent's conversation as system messages so they can be logged or surfaced to observability pipelines.

### MCP Integration

```go
package main

import (
    "context"
    "log"

    "github.com/sweetpotato0/ai-allin/agent"
    frameworkmcp "github.com/sweetpotato0/ai-allin/tool/mcp"
)

func main() {
    ctx := context.Background()

    provider, err := frameworkmcp.NewProvider(ctx, frameworkmcp.Config{
        Transport: frameworkmcp.TransportStreamable,
        Endpoint:  "https://example.com/mcp",
    })
    if err != nil {
        log.Fatalf("connect MCP: %v", err)
    }
    defer provider.Close()

    ag := agent.New(
        agent.WithName("mcp-agent"),
        agent.WithSystemPrompt("You are a helpful assistant."),
        agent.WithToolProvider(provider),
    )

    if _, err := ag.Run(ctx, "List available MCP tools."); err != nil {
        log.Fatalf("agent run failed: %v", err)
    }
}
```

### Session Management

```go
package main

import (
    "context"
    "fmt"

    "github.com/sweetpotato0/ai-allin/agent"
    "github.com/sweetpotato0/ai-allin/contrib/provider/openai"
    "github.com/sweetpotato0/ai-allin/session"
    "github.com/sweetpotato0/ai-allin/session/store"
)

func main() {
    ctx := context.Background()

    // Create LLM provider
    llm := openai.New(&openai.Config{
        APIKey: "your-api-key",
        Model:  "gpt-4",
    })

    // Create session manager with Option pattern for store injection
    mgr := session.NewManager(session.WithStore(store.NewInMemoryStore()))

    // Create single-agent session
    ag := agent.New(agent.WithProvider(llm))
    sess, err := mgr.Create(ctx, "session-1", ag)
    if err != nil {
        panic(err)
    }

    // Run session
    response, err := sess.Run(ctx, "Hello")
    if err != nil {
        panic(err)
    }
    fmt.Println(response)

    // Create shared session (multi-agent collaboration)
    sharedSess, err := mgr.CreateShared(ctx, "shared-session")
    if err != nil {
        panic(err)
    }

    // Run with different agents in shared session
    agent1 := agent.New(agent.WithProvider(llm), agent.WithName("researcher"))
    agent2 := agent.New(agent.WithProvider(llm), agent.WithName("solver"))

    resp1, _ := sharedSess.RunWithAgent(ctx, agent1, "Collect information")
    resp2, _ := sharedSess.RunWithAgent(ctx, agent2, "Provide solution based on information")

    fmt.Println(resp1, resp2)
}
```

Every session exposes a serializable `session.Record` (via `session.Session.Snapshot()`), which includes the full message transcript, the last assistant message, and timing metadata for the most recent turn. Combine this with `manager.Save(ctx, session)` after each interaction to persist state into any `session/store` implementation (in-memory, Redis, Postgres, etc.).

### Agentic RAG

The `rag/agentic` package ships a ready-to-use, multi-agent Retrieval-Augmented Generation workflow. It sits on top of dedicated building blocks so that the entire lifecycle stays explicit:

1. **Data preparation** – model sources as `rag/document.Document` and chunk them with a `rag/chunking.Chunker`.
2. **Index construction** – feed chunks to an `rag/embedder.Embedder` (e.g., `embedder.NewVectorAdapter`) and persist vectors via the `rag/retriever` package.
3. **Query & retrieval** – `retriever.Search` embeds questions, queries the `vector.VectorStore`, and optionally reranks using `rag/reranker`.
4. **Generation integration** – the Agentic pipeline plans, routes, and generates the final answer using the curated evidence.

A planner agent decomposes the task, a researcher agent issues searches, a writer agent uses the retrieved evidence, and an optional critic agent reviews the draft answer.

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/sweetpotato0/ai-allin/contrib/provider/openai"
    "github.com/sweetpotato0/ai-allin/rag/agentic"
    vectorstore "github.com/sweetpotato0/ai-allin/vector/store"
)

func main() {
    ctx := context.Background()
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        log.Fatal("missing OPENAI_API_KEY")
    }

    llm := openai.New(openai.DefaultConfig(apiKey))
    embedder := newKeywordEmbedder() // see examples/rag/agentic for a placeholder implementation
    store := vectorstore.NewInMemoryVectorStore()

    pipeline, err := agentic.NewPipeline(
        agentic.Clients{Default: llm},
        embedder,
        store,
        agentic.WithTopK(3),
    )
    if err != nil {
        log.Fatal(err)
    }

    _ = pipeline.IndexDocuments(ctx,
        agentic.Document{ID: "shipping", Title: "Shipping Policy", Content: "..."},
        agentic.Document{ID: "returns", Title: "Return Policy", Content: "..."},
    )

    resp, err := pipeline.Run(ctx, "Summarize shipping timelines and returns.")
    if err != nil {
        log.Fatal(err)
    }

    log.Println("Plan steps:", len(resp.Plan.Steps))
    log.Println("Answer:", resp.FinalAnswer)
}
```

See `docs/rag/overview.md` for a deeper dive and `examples/rag/agentic` for a runnable demo with OpenAI plus a toy embedder. Already have a production retrieval stack? Wrap it and hand it to the pipeline via `agentic.WithRetriever(...)`.

If you need to rehydrate sessions from a persistent store in a new process, register an `AgentResolver` with `session.WithAgentResolver` so the manager knows how to rebuild the underlying agent prototype for any single-agent session.

## Architecture

### Core Packages

- **agent**: Agent implementation with options pattern
- **context**: Conversation context management
- **graph**: Workflow graph execution
- **memory**: Memory storage interface and implementations
- **message**: Message and role definitions
- **middleware**: Middleware chain for request processing
- **prompt**: Prompt template management
- **runner**: Parallel task execution
- **session**: Session management, supporting single-agent and multi-agent shared sessions
  - **session/store**: Session storage backends (InMemory, Redis, etc.)
- **tool**: Tool registration and execution
- **vector**: Vector embedding storage and search

### Storage Implementations

- **InMemory**: Fast development storage
- **PostgreSQL**: Production-grade with full-text search indexes
- **Redis**: High-performance caching layer
- **MongoDB**: Document-based storage
- **PGVector**: Vector similarity search

## Configuration

### Environment Variables

```bash
# PostgreSQL Configuration
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5432
export POSTGRES_USER=postgres
export POSTGRES_PASSWORD=your_password
export POSTGRES_DB=ai_allin
export POSTGRES_SSLMODE=disable

# Redis Configuration
export REDIS_ADDR=localhost:6379
export REDIS_PASSWORD=""
export REDIS_DB=0
export REDIS_PREFIX=ai-allin:memory:

# MongoDB Configuration
export MONGODB_URI=mongodb://localhost:27017
export MONGODB_DB=ai_allin
export MONGODB_COLLECTION=memories
```

## Performance Optimizations

### Recent Improvements

| Operation | Before | After | Improvement |
|-----------|--------|-------|-------------|
| ID Generation | 1000 ns/op | 113 ns/op | 9x faster |
| Full-Text Search | O(n) scan | O(log n) index | 10-1000x faster |
| Concurrent Connections | Unlimited | 25 pooled | More stable |
| Query Timeouts | None | 30 seconds | Resource safe |

### Thread Safety

All concurrent operations are protected with sync.RWMutex:
- Context message management
- Tool registry operations
- Prompt template management

## Testing

Run all tests:

```bash
go test ./...
```

Run specific package tests:

```bash
go test ./agent -v
go test ./config -v
go test ./memory -v
```

## Production Deployment

### Prerequisites

1. PostgreSQL 12+ (optional, for production storage)
2. Go 1.18+
3. Set required environment variables

### Configuration

1. Set up environment variables for your database
2. Run database migrations
3. Configure connection pooling based on your load
4. Enable query timeouts (default: 30 seconds)

### Monitoring

Monitor these metrics:
- Active database connections
- Query execution times
- Memory usage (with pagination limits)
- Error rates by operation type

## Contributing

Contributions are welcome! Please ensure:
- Code passes `go build ./...`
- Tests pass `go test ./...`
- Code follows Go conventions
- Changes are well-documented

## License

MIT

## Support

For issues, questions, or contributions, please refer to the project repository.
