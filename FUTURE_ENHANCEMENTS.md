# Future Enhancements - Implementation Plan

This document outlines the planned future enhancements for the AI-Allin framework.

## 1. Additional Storage Backends

### PostgreSQL Backend
- **Status**: Planned
- **Location**: `memory/store/postgres/` and `session/store/postgres/`
- **Purpose**: Persistent, scalable storage for production environments
- **Features**:
  - Connection pooling with configurable pool size
  - JSON columns for flexible data storage
  - TTL support with background cleanup
  - Transaction support for atomicity
  - Full-text search capabilities
  - Automatic schema migration

**Implementation Steps**:
1. Create `memory/store/postgres/` package
   - Implement `MemoryStore` interface
   - Use `github.com/lib/pq` driver
   - Support connection pooling with pgx
   - Implement TTL cleanup job

2. Create `session/store/postgres/` package
   - Implement session storage
   - Support multiple concurrent sessions
   - Handle session expiration

3. Add tests and documentation
4. Create example usage

**Dependencies**:
- `github.com/lib/pq` or `github.com/jackc/pgx/v5`
- Database migration tool (e.g., golang-migrate)

### MongoDB Backend
- **Status**: Planned
- **Location**: `memory/store/mongodb/` and `session/store/mongodb/`
- **Purpose**: NoSQL storage for flexible schema and horizontal scaling
- **Features**:
  - TTL index support
  - Aggregation pipeline for complex queries
  - Bulk operations for performance
  - Replica set support for production
  - Change streams for real-time updates

**Implementation Steps**:
1. Create `memory/store/mongodb/` package
   - Implement `MemoryStore` interface
   - Use `go.mongodb.org/mongo-driver`
   - Set up TTL indexes
   - Implement connection pooling

2. Create `session/store/mongodb/` package
   - Implement session storage
   - Use MongoDB's expiration capability

3. Add comprehensive tests
4. Create example usage

**Dependencies**:
- `go.mongodb.org/mongo-driver`
- MongoDB server (local or cloud)

## 2. Streaming LLM Response Support

- **Status**: Planned
- **Location**: `agent/stream.go`, `contrib/provider/*/stream.go`
- **Purpose**: Real-time token streaming for improved UX
- **Features**:
  - Stream callback support
  - Proper error handling during streaming
  - Connection timeout handling
  - Token buffering options

**Implementation Steps**:
1. Add `StreamCallback` type to agent package:
   ```go
   type StreamCallback func(token string) error
   ```

2. Extend `LLMClient` interface:
   ```go
   type StreamLLMClient interface {
       GenerateStream(ctx context.Context, messages []*message.Message,
                     tools []map[string]interface{},
                     callback StreamCallback) (*message.Message, error)
   }
   ```

3. Implement streaming in OpenAI provider
   - Use OpenAI's streaming endpoint
   - Parse SSE format
   - Handle tool calls during streaming

4. Implement streaming in Claude provider
   - Use Claude's event stream format
   - Parse response format
   - Handle tool calls during streaming

5. Add `Agent.RunStream()` method
6. Create streaming examples

## 3. Vector Search for Memory

- **Status**: Planned
- **Location**: `memory/vector/`
- **Purpose**: Semantic search capabilities for memory retrieval
- **Features**:
  - Vector embedding generation
  - Similarity search (cosine, euclidean)
  - ANN (Approximate Nearest Neighbor) support
  - Hybrid search (semantic + keyword)

**Implementation Steps**:
1. Create `memory/vector/embedder.go`
   - Define `Embedder` interface
   - Support multiple embedding models:
     - OpenAI's text-embedding-3-small
     - OpenAI's text-embedding-3-large
     - Hugging Face embeddings (via local server or API)
     - Sentence transformers

2. Create `memory/vector/search.go`
   - Implement similarity search algorithms
   - Support different distance metrics
   - Handle dimension mismatches

3. Extend storage backends:
   - PostgreSQL: Use `pgvector` extension
   - MongoDB: Use MILVUS or native vector search
   - In-Memory: Simple array-based search

4. Update `MemoryStore` interface:
   ```go
   type VectorMemoryStore interface {
       MemoryStore
       SearchByVector(ctx context.Context, vector []float32, topK int) ([]Memory, error)
       AddMemoryWithVector(ctx context.Context, memory *Memory, vector []float32) error
   }
   ```

5. Create examples:
   - Vector-based memory search
   - Hybrid search examples

## 4. Middleware Support

- **Status**: Planned
- **Location**: `middleware/`
- **Purpose**: Extensible request/response processing pipeline
- **Features**:
  - Pre-processing hooks (before LLM call)
  - Post-processing hooks (after LLM call)
  - Logging and monitoring
  - Rate limiting
  - Caching
  - Request/response transformation
  - Error handling

**Implementation Steps**:
1. Create middleware package structure:
   ```
   middleware/
   ├── middleware.go     # Core middleware interface
   ├── logger.go         # Logging middleware
   ├── ratelimit.go      # Rate limiting
   ├── cache.go          # Response caching
   ├── transform.go      # Request/response transformation
   └── examples.go       # Example implementations
   ```

2. Define middleware interfaces:
   ```go
   type Middleware interface {
       Name() string
       Process(ctx context.Context, req *Request, next Handler) (*Response, error)
   }

   type Handler func(ctx context.Context, req *Request) (*Response, error)
   ```

3. Implement middleware types:
   - **LoggingMiddleware**: Log all requests and responses
   - **RateLimitMiddleware**: Limit requests per time window
   - **CacheMiddleware**: Cache LLM responses by prompt
   - **TimeoutMiddleware**: Enforce request timeouts
   - **MetricsMiddleware**: Collect performance metrics

4. Integrate with Agent:
   ```go
   type Agent struct {
       // ... existing fields
       middlewares []Middleware
   }
   ```

5. Create middleware examples

## 5. Additional LLM Provider Integrations

### Planned Providers:
1. **Google's Gemini** (`contrib/provider/gemini/`)
   - Use Google's Generative AI SDK
   - Support tool calling
   - Support streaming

2. **LLaMA/Ollama** (`contrib/provider/ollama/`)
   - Local model support
   - API compatibility with OpenAI format
   - Support for custom models

3. **Groq** (`contrib/provider/groq/`)
   - High-speed inference
   - OpenAI API compatible
   - Support streaming

4. **Cohere** (`contrib/provider/cohere/`)
   - Multi-language support
   - Semantic search capabilities
   - Command-specific models

5. **Azure OpenAI** (`contrib/provider/azureopenai/`)
   - Azure-hosted OpenAI models
   - RBAC support
   - Enterprise features

**Implementation Pattern**:
Each provider should follow the same pattern:
```go
// config.go
type Config struct {
    APIKey      string
    Model       string
    MaxTokens   int64
    Temperature float64
    // Provider-specific fields
}

func DefaultConfig(apiKey string) *Config { ... }

// provider.go
type Provider struct {
    config *Config
    client <ClientType>
}

func New(config *Config) *Provider { ... }

func (p *Provider) Generate(ctx context.Context, messages []*message.Message, tools []map[string]interface{}) (*message.Message, error) { ... }

// Optional streaming support
func (p *Provider) GenerateStream(ctx context.Context, messages []*message.Message, tools []map[string]interface{}, callback StreamCallback) (*message.Message, error) { ... }
```

## Implementation Priority

**Phase 1 (High Priority)**:
1. Streaming LLM Response Support - Improves UX significantly
2. Middleware Support - Enables extensibility and cross-cutting concerns
3. Vector Search for Memory - Enables semantic capabilities

**Phase 2 (Medium Priority)**:
1. PostgreSQL Backend - Enterprise production readiness
2. Additional LLM Providers - Expands platform support
3. MongoDB Backend - NoSQL alternative

**Phase 3 (Lower Priority)**:
1. Additional LLM Providers - Niche requirements
2. Advanced features - Depends on Phase 1 & 2

## Testing Strategy

For each feature:
1. Unit tests with mocks
2. Integration tests with real services (where applicable)
3. Examples demonstrating usage
4. Documentation and API docs
5. Performance benchmarks (for storage backends and streaming)

## Documentation Updates

For each feature:
1. Add to CLAUDE.md
2. Update README.md with examples
3. Create feature-specific guides
4. Add code examples
5. Update API documentation

## Performance Considerations

- Streaming: Reduce latency for user-facing features
- Vector Search: Use approximate algorithms for large datasets
- Middleware: Minimize overhead with efficient chain implementation
- Storage Backends: Connection pooling, query optimization
- Caching: Implement cache invalidation strategies

## Security Considerations

- Middleware: Validate and sanitize inputs
- Streaming: Ensure proper resource cleanup
- Storage: Support encryption at rest and in transit
- Providers: Secure API key handling
- Vector Search: Protect against prompt injection

---

**Last Updated**: 2024-11-04
**Status**: Planning Phase
**Next Review**: After Phase 1 implementation
