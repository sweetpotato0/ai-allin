# AI-ALLIN - AI Agent 框架

一个全面、生产级别的Go框架，用于构建支持流式响应、工具集成和多后端存储的AI智能体。

**[English Documentation](./README.md)**

## 特性

- **多提供商 LLM 支持**: OpenAI、Anthropic Claude、Groq、Cohere、Google Gemini
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
- **会话管理**: 对话会话跟踪和管理
- **执行图**: 支持条件分支的工作流编排
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

    // 运行智能体
    response, err := ag.Run(context.Background(), "What is AI?")
    if err != nil {
        panic(err)
    }

    println(response)
}
```

### MCP 集成示例

```go
package main

import (
    "context"
    "log"

    frameworkmcp "github.com/sweetpotato0/ai-allin/mcp"
)

func main() {
    ctx := context.Background()

    // 通过命令运行的 stdio MCP 服务器
    stdioClient, err := frameworkmcp.NewStdioClient(ctx, "my-mcp-server")
    if err != nil {
        log.Fatalf("连接 stdio MCP 失败: %v", err)
    }
    defer stdioClient.Close()

    // 通过流式 HTTP（SSE）连接远程 MCP 服务器
    httpClient, err := frameworkmcp.NewStreamableClient(ctx, "https://example.com/mcp")
    if err != nil {
        log.Fatalf("连接 streamable MCP 失败: %v", err)
    }
    defer httpClient.Close()
}
```

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
- **session**: 会话管理
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
