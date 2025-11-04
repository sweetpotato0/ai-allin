# CLAUDE.md

本文件为 Claude Code (claude.ai/code) 在本仓库中工作提供指导。

## 项目概览

这是一个基于Go的AI框架，称为 "ai-allin"，提供了一个模块化的架构，用于构建具有消息上下文管理、工具能力、执行工作流和LLM提供商集成的AI代理。

## 构建和开发命令

```bash
# 初始化Go模块依赖
go mod download

# 更新依赖
go mod tidy

# 构建项目
go build ./...

# 运行测试
go test ./...

# 运行特定包的测试
go test ./middleware ./middleware/logger ./middleware/validator
go test ./middleware/errorhandler ./middleware/enricher ./middleware/limiter
go test ./vector/store
go test ./memory
go test ./session
go test ./tool
go test ./message

# 运行测试并显示覆盖率
go test -cover ./...

# 运行示例代码
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

## 架构设计

代码库被组织成多个核心包，它们一起工作以提供AI代理框架：

### 核心包

- **message/** - 定义支持多种角色（用户、助手、系统、工具）的消息结构和工具调用
- **context/** - 管理会话上下文，包含自动消息历史记录和大小限制（集成到Agent中）
- **tool/** - 实现灵活的工具系统，支持参数验证、JSON Schema导出和工具注册表
- **prompt/** - 提供带变量替换和构建器的提示模板管理
- **graph/** - 实现支持条件节点、循环和状态管理的执行流图
- **agent/** - 核心AI代理实现，使用Options模式配置、Context集成、工具调用和LLM客户端接口
- **session/** - 管理会话，支持多个并发会话
  - **session/store/** - 会话存储后端（当前支持Redis）
- **memory/** - 定义代理知识的内存存储接口
  - **memory/store/** - 存储后端包括内存、Redis、PostgreSQL和MongoDB实现
- **middleware/** - 可扩展的请求/响应处理管道
  - **middleware/logger/** - 请求/响应日志中间件
  - **middleware/validator/** - 输入验证和响应过滤
  - **middleware/errorhandler/** - 错误处理和恢复
  - **middleware/enricher/** - 上下文元数据丰富
  - **middleware/limiter/** - 速率限制
- **vector/** - 向量搜索和嵌入支持
  - **vector/store/** - 向量存储后端（内存和pgvector）
- **runner/** - 提供支持并行、顺序和条件执行的任务执行引擎
- **contrib/provider/** - LLM提供商实现
  - **contrib/provider/openai/** - OpenAI API集成，使用官方 `openai-go` SDK
  - **contrib/provider/claude/** - Anthropic Claude集成，使用官方 `anthropic-sdk-go` SDK
  - **contrib/provider/groq/** - Groq API集成（mixtral-8x7b-32768）
  - **contrib/provider/cohere/** - Cohere API集成，支持企业级LLM
  - **contrib/provider/gemini/** - Google Gemini集成

### 设计模式

代码库遵循Go接口设计，采用以下关键模式：

1. **Options模式**：通过函数式选项的Agent配置，无需可变Config结构体
2. **接口隔离**：通过最小化接口定义核心功能（例如 `LLMClient`、`MemoryStore`、`Session`）
3. **存储抽象**：可插拔的存储后端，用于内存和会话
4. **构建器模式**：用于构造复杂对象（如图和提示）的流畅API
5. **注册表模式**：带验证的工具和模板注册
6. **策略模式**：runner中不同的执行策略（并行、顺序、条件）

### 存储后端

框架支持多种存储后端，用于内存和会话：

### 内存存储

- **内存存储**：快速，适合开发/测试（无外部依赖）
- **Redis**：持久化、分布式、生产就绪的存储
- **PostgreSQL**：完整的SQL数据库支持，支持CRUD操作和JSON元数据
- **MongoDB**：基于文档的存储，支持正则表达式搜索

#### 添加新的存储后端

1. 在 `memory/store/` 中实现 `MemoryStore` 接口
2. 在 `session/store/` 中实现会话存储（如需要）
3. 添加适当的配置结构和默认配置

### 向量存储

- **内存存储**：线程安全的向量存储，支持余弦相似度和欧几里得距离计算
- **PostgreSQL pgvector**：使用PostgreSQL pgvector扩展的可扩展向量存储，支持HNSW或IVFFLAT索引

#### 使用向量搜索

```go
import "github.com/sweetpotato0/ai-allin/vector/store"

// 创建内存向量存储
vectorStore := store.NewInMemoryVectorStore()

// 添加嵌入
embedding := &vector.Embedding{
    ID:     "doc1",
    Text:   "您的文本内容",
    Vector: []float32{0.1, 0.2, 0.3, ...},
}
vectorStore.AddEmbedding(ctx, embedding)

// 搜索相似向量
queryVector := []float32{0.15, 0.25, 0.35, ...}
results, err := vectorStore.Search(ctx, queryVector, 10) // 获取前10个结果
```

## 中间件系统详情

中间件系统被组织成专门的包，以获得更好的模块化：

### 包组织

- **middleware/middleware.go** - 核心接口和链编排
- **middleware/logger/** - 请求/响应日志记录
- **middleware/validator/** - 输入验证和响应过滤
- **middleware/errorhandler/** - 错误处理和恢复
- **middleware/enricher/** - 上下文元数据丰富
- **middleware/limiter/** - 速率限制

### 高级中间件使用

```go
ag := agent.New(
    agent.WithProvider(llm),
    // 按顺序添加多个中间件
    agent.WithMiddleware(logger.NewRequestLogger(func(msg string) {
        log.Println(msg)
    })),
    agent.WithMiddleware(validator.NewInputValidator(func(input string) error {
        if len(input) > 1000 {
            return fmt.Errorf("输入过长")
        }
        return nil
    })),
    agent.WithMiddleware(limiter.NewRateLimiter(100)), // 最多100个请求
    agent.WithMiddleware(errorhandler.NewErrorHandler(func(err error) error {
        log.Printf("错误：%v\n", err)
        return nil // 继续处理
    })),
)
```

### 创建自定义中间件

```go
type CustomMiddleware struct {
    processor func(*middleware.Context) error
}

func (m *CustomMiddleware) Name() string {
    return "custom-middleware"
}

func (m *CustomMiddleware) Execute(ctx *middleware.Context, next middleware.Handler) error {
    // 预处理
    if err := m.processor(ctx); err != nil {
        return err
    }

    // 调用下一个中间件
    if err := next(ctx); err != nil {
        return err
    }

    // 后处理
    return nil
}
```

## 关键实现说明

- Agent使用Context模块进行消息管理，具有自动历史记录修剪功能
- 所有上下文操作都使用 `sync.RWMutex` 实现线程安全
- 内存和会话操作接受 `context.Context` 以支持取消
- 工具执行在处理程序调用之前包含参数验证
- 图执行包含无限循环检测（每个节点最多100次访问）
- 会话管理器支持清理非活动会话
- Options模式用于灵活的Agent配置：
  - `WithName()`、`WithSystemPrompt()`、`WithMaxIterations()`、`WithTemperature()`
  - `WithProvider()`、`WithTools()`、`WithMemory()`
- 项目使用Go 1.23.1（如 [go.mod](go.mod) 中指定）
- 模块路径为 `github.com/sweetpotato0/ai-allin`

## LLM集成

要与LLM提供商集成，实现 `agent.LLMClient` 接口：

```go
type LLMClient interface {
    Generate(ctx context.Context, messages []*message.Message, tools []map[string]interface{}) (*message.Message, error)
    SetTemperature(temp float64)
    SetMaxTokens(max int64)
    SetModel(model string)
}
```

### 提供的实现

#### OpenAI提供商

```go
import "github.com/sweetpotato0/ai-allin/contrib/provider/openai"

config := openai.DefaultConfig(apiKey)
config.Temperature = 0.7
config.MaxTokens = 2000
provider := openai.New(config)

agent := agent.New(
    agent.WithProvider(provider),
    agent.WithSystemPrompt("你是一个有帮助的助手"),
)
```

#### Claude提供商

```go
import "github.com/sweetpotato0/ai-allin/contrib/provider/claude"

config := claude.DefaultConfig(apiKey)
config.Temperature = 0.7
config.MaxTokens = 4096
provider := claude.New(config)

agent := agent.New(
    agent.WithProvider(provider),
    agent.WithSystemPrompt("你是一个有帮助的助手"),
)
```

#### Groq提供商

```go
import "github.com/sweetpotato0/ai-allin/contrib/provider/groq"

config := groq.DefaultConfig(apiKey)
config.Model = "mixtral-8x7b-32768"  // 快速推理
provider := groq.New(config)

agent := agent.New(
    agent.WithProvider(provider),
    agent.WithSystemPrompt("你是一个有帮助的助手"),
)
```

#### Cohere提供商

```go
import "github.com/sweetpotato0/ai-allin/contrib/provider/cohere"

config := cohere.DefaultConfig(apiKey)
config.Model = "command"
provider := cohere.New(config)

agent := agent.New(
    agent.WithProvider(provider),
    agent.WithSystemPrompt("你是一个有帮助的助手"),
)
```

#### Gemini提供商

```go
import "github.com/sweetpotato0/ai-allin/contrib/provider/gemini"

config := gemini.DefaultConfig(apiKey)
config.Model = "gemini-pro"
provider := gemini.New(config)

agent := agent.New(
    agent.WithProvider(provider),
    agent.WithSystemPrompt("你是一个有帮助的助手"),
)
```

所有提供商都支持工具调用、配置方法，并且生产就绪。您可以通过更新提供商配置动态切换提供商：

```go
provider.SetTemperature(0.9)
provider.SetMaxTokens(1024)
provider.SetModel("different-model")
```

## 使用Options模式的Agent用法

```go
// 使用灵活的选项创建agent
ag := agent.New(
    agent.WithName("Assistant"),
    agent.WithSystemPrompt("你是一个有帮助的助手"),
    agent.WithProvider(llmProvider),
    agent.WithTemperature(0.7),
    agent.WithMaxIterations(10),
    agent.WithTools(true),
    agent.WithMemory(memoryStore),
)

// 运行agent
result, err := ag.Run(ctx, "您的问题")
```

## 上下文管理

Context模块已集成到Agent中，用于自动消息历史管理：

```go
// Context特性：
// - AddMessage(msg) - 添加消息到会话
// - GetMessages() - 获取所有消息
// - GetLastMessage() - 获取最新消息
// - GetMessagesByRole() - 按角色过滤消息
// - Clear() - 清除所有消息（在Agent中保留系统消息）
// - Size() - 获取消息计数

// Agent的上下文自动：
// - 当超过maxSize时修剪旧消息（默认100条）
// - 在修剪期间保留系统消息
// - 为LLM管理消息历史
```

## 测试策略

- 在 `*_test.go` 文件中进行核心逻辑的单元测试
- 使用Mock LLM客户端测试agent功能
- 测试工具验证、消息创建和注册表操作的覆盖
- 集成测试应使用内存存储后端以加快速度
- 示例文件充当集成测试，展示所有功能

## 示例

所有示例都在 `examples/` 目录中组织：

1. **examples/main.go** - 综合框架示例
   - 基本agent使用
   - 带工具的agent
   - 图工作流
   - 会话管理
   - 并行执行

2. **examples/basic/main.go** - Options模式用法
3. **examples/tools/main.go** - 工具注册和执行
4. **examples/context/main.go** - Context模块演示
5. **examples/graph/main.go** - 图工作流示例
6. **examples/providers/main.go** - LLM提供商用法（OpenAI、Claude）
7. **examples/middleware/main.go** - 中间件系统演示
8. **examples/streaming/main.go** - 流式LLM响应演示
9. **examples/allproviders/main.go** - 所有LLM提供商演示

## 已完成的功能

✅ 带角色和工具调用的消息结构
✅ 自动历史记录限制的上下文管理
✅ 带参数验证和JSON Schema生成的工具注册表
✅ 带模板和构建器的提示管理
✅ 支持条件节点和状态管理的图执行
✅ Options模式配置的Agent核心
✅ 集成到Agent中的Context模块
✅ 支持持久化的会话管理
✅ 内存和Redis存储后端
✅ 带完整CRUD操作的PostgreSQL存储后端
✅ 基于文档的MongoDB存储后端
✅ OpenAI提供商（使用官方openai-go SDK）
✅ Claude提供商（使用官方anthropic-sdk-go SDK）
✅ 用于快速推理的Groq提供商（mixtral-8x7b-32768）
✅ 用于企业级LLM集成的Cohere提供商
✅ Google生成AI的Gemini提供商
✅ 并行、顺序和条件任务运行器
✅ 展示所有功能的综合示例
✅ 流式LLM响应支持
✅ 用于可扩展请求/响应处理的中间件支持
✅ 带余弦相似度和欧几里得距离的向量搜索功能
✅ 支持TopK相似度搜索的内存向量存储
✅ 用于可扩展向量操作的PostgreSQL pgvector存储
✅ 带SetTemperature、SetMaxTokens、SetModel方法的LLMClient接口

## 中间件系统

框架包括用于请求/响应处理的灵活中间件系统：

### 中间件接口

```go
type Middleware interface {
    Name() string
    Execute(ctx *Context, next Handler) error
}

type Handler func(*Context) error
```

### 内置中间件

1. **RequestLogger** - 记录传入请求
2. **ResponseLogger** - 记录传出响应
3. **InputValidator** - 验证和清理输入
4. **ResponseFilter** - 过滤或转换响应
5. **ContextEnricher** - 将元数据添加到上下文
6. **ErrorHandler** - 处理管道中的错误
7. **RateLimiter** - 速率限制支持

### 使用示例

```go
ag := agent.New(
    agent.WithProvider(llm),
    agent.WithMiddleware(middleware.NewRequestLogger(func(msg string) {
        fmt.Println(msg)
    })),
    agent.WithMiddleware(middleware.NewInputValidator(func(input string) error {
        if len(input) > 1000 {
            return errors.New("输入过长")
        }
        return nil
    })),
)
```

### 中间件链执行

中间件按顺序执行，每个中间件都能够：
- 检查或修改请求上下文
- 在LLM调用前执行预处理
- 在LLM响应后执行后处理
- 停止执行并返回错误
- 将控制权传递给下一个中间件

## 未来增强

- 嵌入服务集成（OpenAI嵌入、Cohere嵌入等）
- 额外的存储后端（Elasticsearch、Milvus、Weaviate）
- 高级中间件模式（缓存、重试逻辑、断路器）
- 工具调用改进和扩展
- 性能优化和基准测试
- 分布式agent支持，用于多agent系统
- 用于agent管理和监控的Web UI
- 用于自定义扩展的插件系统
- 用于agent交互的GraphQL API
