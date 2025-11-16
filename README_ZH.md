# AI-ALLIN - AI Agent 框架

一个全面、生产级别的Go框架，用于构建支持流式响应、工具集成和多后端存储的AI智能体。

**[English Documentation](./README.md)**

## 特性

- **多提供商 LLM 支持**: OpenAI、Anthropic Claude、Google Gemini
- **流式响应支持**: 所有 LLM 提供商的实时流式输出
- **Agent 框架**: 支持中间件、提示词和记忆的可配置智能体
- **工具集成**: 注册和执行工具/函数
- **MCP 模型上下文协议支持**：支持 stdio 与流式 HTTP（SSE）连接，自动同步 MCP 工具并通过智能体调用
- **多后端存储**:
  - 内存存储(开发环境)
  - PostgreSQL 全文搜索
  - Redis 缓存
  - MongoDB 文档存储
  - PGVector 向量嵌入
- **会话管理**: 对话会话跟踪和管理，支持单 agent 和多 agent 共享会话，提供可持久化的会话快照与运行统计
- **运行时执行层**：Agent Spec + Executor 解耦运行逻辑与会话数据，便于观测和扩展
- **工具监督器**：内置监督器自动加载并刷新工具提供商，保持工具 schema 最新
- **执行图**: 支持条件分支的工作流编排
- **Agentic RAG**: 基于 `graph.Graph` 串联规划、检索、写作、审阅智能体的多智能体 RAG 流程
- **RAG 组件化**: `rag/document`、`rag/chunking`、`rag/embedder`、`rag/retriever`、`rag/reranker` 等包让数据准备、检索、重排更易扩展
- **可观测性就绪**：`pkg/logging` + `pkg/telemetry` 提供结构化日志与 OpenTelemetry 追踪钩子，覆盖 Agent、RAG、Session、Runtime 等核心路径
- **线程安全**: RWMutex 保护的并发访问
- **配置验证**: 基于环境变量的配置与验证

## 快速开始

### 安装

```bash
go get github.com/sweetpotato0/ai-allin
```

### 基本用法

```go
package main

import (
    "context"
    "github.com/sweetpotato0/ai-allin/agent"
    "github.com/sweetpotato0/ai-allin/contrib/provider/openai"
)

func main() {
    // 创建 LLM 提供商
    llm := openai.New(&openai.Config{
        APIKey:      "your-api-key",
        Model:       "gpt-4",
        MaxTokens:   2000,
        Temperature: 0.7,
    })

    // 创建智能体
    ag := agent.New(
        agent.WithName("MyAgent"),
        agent.WithSystemPrompt("You are a helpful assistant"),
        agent.WithProvider(llm),
    )

    // 运行智能体；Run 现在返回 *message.Message，可读取更多元数据
    resp, err := ag.Run(context.Background(), "What is AI?")
    if err != nil {
        panic(err)
    }

    println(resp.Text())
}
```

### 运行时执行器

`runtime` 包让“运行配置”与“会话历史”解耦。你可以直接构造一个执行器，给它一份历史记录，它会返回最新的回复以及耗时等元数据：

```go
exec := runtime.NewAgentExecutor(ag)
result, err := exec.Execute(ctx, &runtime.Request{
    SessionID: "session-1",
    Input:     "接下来怎么办？",
    History:   historyMessages,
})
if err != nil {
    log.Fatalf("executor 执行失败: %v", err)
}
fmt.Println("assistant:", result.Output, "duration:", result.Duration)
```

`session.SingleAgentSession` 与 `SharedSession` 也正是基于该执行器构建，因此你可以很容易地替换为自定义的执行策略（流式、带追踪、并行等）。

### 工具监督器

当你使用 `agent.WithToolProvider` 注册工具提供商时，新的工具监督器会自动拉取、缓存并监听提供商的变更：

```go
ag := agent.New(
    agent.WithProvider(llm),
    agent.WithToolProvider(myProvider),
)
```

如果刷新失败，监督器会以系统消息的形式注入上下文，方便你通过日志或监控系统捕获。

### MCP 集成示例

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
        log.Fatalf("连接 MCP 失败: %v", err)
    }
    defer provider.Close()

    ag := agent.New(
        agent.WithName("mcp-agent"),
        agent.WithSystemPrompt("你是一名能够调用 MCP 工具的智能助手。"),
        agent.WithToolProvider(provider),
    )

    if _, err := ag.Run(ctx, "列出所有 MCP 工具。"); err != nil {
        log.Fatalf("Agent 运行失败: %v", err)
    }
}
```

#### 本地 MCP 演示服务

`examples/mcp` 目录包含两个可运行的 MCP 服务，覆盖 HTTP（SSE）与 stdio 传输，方便端到端验证：

```bash
# HTTP 流式（两个终端）
go run ./examples/mcp/http --host 127.0.0.1 --port 8080 --path /mcp
go run ./examples/mcp -transport stream -endpoint http://127.0.0.1:8080/mcp -prompt "列出可用工具"

# stdio（先构建，再让代理拉起二进制）
go build -o ./bin/mcp-stdio ./examples/mcp/stdio
go run ./examples/mcp -transport stdio -command ./bin/mcp-stdio -prompt "获取 Tokyo 天气"
```

### Session 管理示例

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

    // 创建 LLM 提供商
    llm := openai.New(&openai.Config{
        APIKey: "your-api-key",
        Model:  "gpt-4",
    })

    // 创建 session manager，使用 Option 模式注入 store 实现
    mgr := session.NewManager(session.WithStore(store.NewInMemoryStore()))

    // 创建单 agent session
    ag := agent.New(agent.WithProvider(llm))
    sess, err := mgr.Create(ctx, "session-1", ag)
    if err != nil {
        panic(err)
    }

    // 运行 session
    response, err := sess.Run(ctx, "Hello")
    if err != nil {
        panic(err)
    }
    fmt.Println(response)

    // 创建共享 session（多 agent 协作）
sharedSess, err := mgr.CreateShared(ctx, "shared-session")
if err != nil {
        panic(err)
    }

    // 使用不同的 agent 在共享 session 中运行
    agent1 := agent.New(agent.WithProvider(llm), agent.WithName("researcher"))
    agent2 := agent.New(agent.WithProvider(llm), agent.WithName("solver"))

    resp1, _ := sharedSess.RunWithAgent(ctx, agent1, "收集信息")
    resp2, _ := sharedSess.RunWithAgent(ctx, agent2, "基于信息提供解决方案")

    fmt.Println(resp1, resp2)
}
```

### 可观测性示例

框架内置 `pkg/telemetry`，可一行代码启用 OpenTelemetry 追踪。完整示例见 `examples/telemetry`：

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
go run ./examples/telemetry
```

该示例展示了如何初始化遥测、构建一个使用自定义 LLM 的 Agent，并输出带追踪 ID 的结构化日志。当未配置 OTLP 端点时，追踪信息会自动打印到 stdout，方便本地调试。

每个会话都可以通过 `session.Session.Snapshot()` 生成 `session.Record`，其中包含完整消息历史、最近一次回复以及执行耗时。调用 `mgr.Save(ctx, sess)` 即可把最新快照写入任意 `session/store` 实现（内存、Redis、Postgres 等），用于持久化或分析。

如果需要在新的进程中恢复单 Agent 会话，可以通过 `session.WithAgentResolver` 注册一个 Agent 解析器，让 `Manager` 知道如何为对应的 `session.Record` 重建 Agent。

### Agentic RAG（多智能体）

`rag/agentic` 提供一套即开即用的多智能体 RAG 流程，并围绕经典的四个阶段拆分为独立组件：

1. **数据准备**：使用 `rag/document.Document` 表示原始资料，配合 `rag/chunking.Chunker` 切分成片段。
2. **索引构建**：将片段交给 `rag/embedder.Embedder` 生成向量，并使用 `rag/retriever.IndexDocuments` 写入 `vector.VectorStore`。
3. **查询与检索**：`retriever.Search` 负责生成查询向量、向量检索，以及可选的 `rag/reranker` 重排。
4. **生成集成**：Agentic 管线在拿到证据后调用规划 / 检索 / 写作 / 评审智能体，生成可审计的答案。

规划智能体拆解任务，检索智能体把步骤翻译成向量查询，写作智能体依据证据撰写草稿，可选的评审智能体在收尾阶段校对和改写。

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
        log.Fatal("缺少 OPENAI_API_KEY")
    }

    llm := openai.New(openai.DefaultConfig(apiKey))
    embedder := newKeywordEmbedder() // 示例嵌入器，生产环境请替换；参见 examples/rag/agentic
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

    resp, err := pipeline.Run(ctx, "总结物流时效与退货政策。")
    if err != nil {
        log.Fatal(err)
    }

    log.Println("规划步骤数:", len(resp.Plan.Steps))
    log.Println("最终回答:", resp.FinalAnswer)
}
```

更多细节请查看 `docs/rag/overview.md`，并参考 `examples/rag/agentic` 运行端到端示例。若已有独立的检索系统，也可以通过 `agentic.WithRetriever(...)` 将其直接注入流水线，跳过默认的切片与索引步骤。

#### 基于 Postgres + pgvector 的示例

`examples/rag/postgres` 展示了如何把 Agentic RAG 流程落在 pgvector 上，并使用 OpenAI `text-embedding-3-small` 生成向量。示例会自动读取 `docs/` 目录下的所有 Markdown 文件，以及仓库根目录的 `README*.md`、`AGENTS.md`、`CLAUDE.md`，并将它们写入向量库，便于直接就文档内容提问。

```bash
# 1. 启动 pgvector（也可以改成自己的集群）
docker run --rm -e POSTGRES_PASSWORD=postgres -p 5432:5432 ankane/pgvector

# 2. 准备运行所需的环境变量
export OPENAI_API_KEY=sk-...
export PGVECTOR_PASSWORD=postgres
export PGVECTOR_USER=postgres
export PGVECTOR_DATABASE=postgres

# 3. 运行示例（首次启动会自动建索引，之后复用；加 -reindex 可重建）
go run ./examples/rag/postgres -question "AI-Allin 如何支持 MCP 协议？"
```

如需自定义连接信息，设置 `PGVECTOR_HOST`、`PGVECTOR_PORT`、`PGVECTOR_SSLMODE`、`PGVECTOR_TABLE` 等环境变量即可。

## 架构

### 核心包

- **agent**: 智能体实现，采用选项模式
- **context**: 对话上下文管理
- **graph**: 工作流图执行
- **memory**: 内存存储接口与实现
- **message**: 消息和角色定义
- **middleware**: 请求处理中间件链
- **prompt**: 提示词模板管理
- **runner**: 并行任务执行
- **session**: 会话管理，支持单 agent 和多 agent 共享会话
  - **session/store**: 会话存储后端（InMemory、Redis 等）
- **tool**: 工具注册和执行
- **vector**: 向量嵌入存储和搜索

### 存储实现

- **InMemory**: 快速开发存储
- **PostgreSQL**: 生产级别，支持全文搜索索引
- **Redis**: 高性能缓存层
- **MongoDB**: 文档型存储
- **PGVector**: 向量相似度搜索

## 配置

### 环境变量

```bash
# PostgreSQL 配置
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5432
export POSTGRES_USER=postgres
export POSTGRES_PASSWORD=your_password
export POSTGRES_DB=ai_allin
export POSTGRES_SSLMODE=disable

# Redis 配置
export REDIS_ADDR=localhost:6379
export REDIS_PASSWORD=""
export REDIS_DB=0
export REDIS_PREFIX=ai-allin:memory:

# MongoDB 配置
export MONGODB_URI=mongodb://localhost:27017
export MONGODB_DB=ai_allin
export MONGODB_COLLECTION=memories
```

## 性能优化

### 最近改进

| 操作 | 优化前 | 优化后 | 性能提升 |
|------|-------|-------|---------|
| ID生成 | 1000 ns/op | 113 ns/op | 9倍 |
| 全文搜索 | O(n) 扫描 | O(log n) 索引 | 10-1000倍 |
| 并发连接 | 无限制 | 25个连接池 | 更稳定 |
| 查询超时 | 无 | 30秒 | 资源安全 |

### 线程安全

所有并发操作都通过 sync.RWMutex 保护:
- 上下文消息管理
- 工具注册表操作
- 提示词模板管理

## 测试

运行所有测试:

```bash
go test ./...
```

运行特定包的测试:

```bash
go test ./agent -v
go test ./config -v
go test ./memory -v
```

## 生产部署

### 环境要求

1. PostgreSQL 12+ (可选，用于生产存储)
2. Go 1.18+
3. 设置必需的环境变量

### 配置步骤

1. 为数据库设置环境变量
2. 运行数据库迁移
3. 根据负载配置连接池
4. 启用查询超时(默认: 30秒)

### 监控指标

监控以下指标:
- 活跃数据库连接数
- 查询执行时间
- 内存使用量(受分页限制保护)
- 各类操作的错误率

## 贡献

欢迎贡献! 请确保:
- 代码通过 `go build ./...`
- 测试通过 `go test ./...`
- 代码遵循 Go 规范
- 变更有良好的文档

## 许可证

MIT

## 支持

如有问题、疑问或建议，请参考项目仓库。
