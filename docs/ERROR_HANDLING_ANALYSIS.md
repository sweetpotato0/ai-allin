# ai-allin 项目错误处理分析报告

## 执行时间：2025-11-04

---

## 一、错误处理的不一致性分析

### 1.1 错误返回的不一致性

#### 问题1：方法签名不统一
**位置1：** `/agent/agent.go` 第149-150行
```go
func (a *Agent) RegisterTool(t *tool.Tool) error {
    return a.tools.Register(t)  // 返回 error
}
```

**位置2：** `/agent/agent.go` 第159-161行
```go
func (a *Agent) AddMiddleware(m middleware.Middleware) {
    a.middlewares.Add(m)  // 不返回 error - 一致性问题
}
```

**问题描述：** `RegisterTool()` 和 `RegisterPrompt()` 返回 error，但 `AddMiddleware()` 不返回 error。而 `Add()` 方法实际上只是 append 操作，不会失败。但这种不一致的设计令人困惑。

**修复建议：**
- 要么让所有注册/添加方法都返回 error
- 或者将返回 error 的方法改为返回 bool（用于已存在检查）
- 优先建议：保持 error 返回，可以接收未来的验证需求

**难度评估：** 低 - 仅需改变方法签名和调用点

---

#### 问题2：内存存储的错误处理不一致
**位置1：** `/memory/store/inmemory.go` 第84-90行
```go
func (s *InMemoryStore) Clear() error {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.memories = make([]*memory.Memory, 0)
    return nil  // 总是返回 nil
}
```

**位置2：** `/memory/store/inmemory.go` 第93-98行
```go
func (s *InMemoryStore) Count() int {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return len(s.memories)  // 不返回 error
}
```

**位置3：** `/memory/store/postgres.go` 第202-208行
```go
func (s *PostgresStore) Clear(ctx context.Context) error {
    _, err := s.db.ExecContext(ctx, "DELETE FROM memories")
    if err != nil {
        return fmt.Errorf("failed to clear memories: %w", err)
    }
    return nil
}
```

**位置4：** `/memory/store/postgres.go` 第211-218行
```go
func (s *PostgresStore) Count(ctx context.Context) (int, error) {
    var count int
    err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM memories").Scan(&count)
    if err != nil {
        return 0, fmt.Errorf("failed to count memories: %w", err)
    }
    return count, nil
}
```

**问题描述：**
- InMemoryStore: `Count()` 返回 `int`，不返回 error（不可能失败）
- PostgresStore: `Count()` 返回 `(int, error)`（可能失败）
- InMemoryStore: `Clear()` 总是返回 nil，不可能失败

这导致调用方难以一致地处理错误。

**修复建议：**
- 创建统一的接口定义，明确哪些方法返回 error
- InMemoryStore 的 `Count()` 应该改为不返回 error（不需要）
- 或者为了接口一致性，统一返回 `(int, error)`
- 推荐：在 `memory.go` 中定义接口时明确约定

**难度评估：** 中 - 需要修改多个实现，可能有接口兼容性问题

---

#### 问题3：Runner 中的错误返回不一致
**位置1：** `/runner/runner.go` 第91-111行
```go
func (pr *ParallelRunner) RunParallel(ctx context.Context, tasks []*Task) []*Result {
    // 不返回 error，但每个 Result 可能包含 error
    results := make([]*Result, len(tasks))
    // ... 
    return results
}
```

**位置2：** `/runner/runner.go` 第126-153行
```go
func (sr *SequentialRunner) RunSequential(ctx context.Context, tasks []*Task) (*Result, error) {
    // 返回 error - 与 RunParallel 不一致
    // ...
    return &Result{...}, nil
}
```

**问题描述：** `RunParallel()` 不返回 error，`RunSequential()` 返回 error。这种不一致使 API 难以使用。

**修复建议：**
```go
// 方案1：统一返回 error
func (pr *ParallelRunner) RunParallel(ctx context.Context, tasks []*Task) ([]*Result, error)

// 或方案2：都不返回 error，errors 在 Result 中
type Result struct {
    TaskID string
    Output string
    Error  error  // 已存在
}
```

**难度评估：** 中 - 需要修改调用方的错误处理

---

### 1.2 错误包装的不一致性

#### 问题4：fmt.Errorf 包装方式不一致
**位置1：** `/agent/agent.go` 第225行
```go
return fmt.Errorf("LLM generation failed: %w", err)  // 使用 %w - 好做法
```

**位置2：** `/memory/store/inmemory.go` 第31行
```go
return fmt.Errorf("memory cannot be nil")  // 不包装任何内容
```

**位置3：** `/vector/store/pgvector.go` 第294行
```go
fmt.Sscanf(strings.TrimSpace(part), "%f", &v)  // 错误完全忽略！
vec = append(vec, v)
```

**位置4：** `/runner/runner.go` 第188行
```go
return results, fmt.Errorf("condition evaluation failed: %w", err)  // 正确使用 %w
```

**问题描述：** 
- 大多数地方使用 `fmt.Errorf(..., %w, err)` 正确包装
- 有些地方忽略底层错误信息
- 有些地方直接忽略返回的 error（见位置3）

**修复建议：**
```go
// pgvector.go 第294行应改为：
var v float32
if _, err := fmt.Sscanf(strings.TrimSpace(part), "%f", &v); err != nil {
    return nil, fmt.Errorf("failed to parse vector component: %w", err)
}
```

**难度评估：** 低 - 只需添加错误检查和包装

---

### 1.3 错误信息格式的不一致

#### 问题5：错误信息的风格不统一
```go
// 风格1：snake_case + "cannot be" 
fmt.Errorf("memory cannot be nil")

// 风格2：省略号 + 冒号
fmt.Errorf("failed to connect to PostgreSQL: %w", err)

// 风格3：不同的动词
fmt.Errorf("tool %s has no handler", t.Name)
fmt.Errorf("embedding ID cannot be empty")
fmt.Errorf("embedding dimension mismatch: expected %d, got %d", ...)

// 风格4：含糊的错误
fmt.Errorf("memory not found")  // 应该更具体：缺少 ID 或存储问题？
```

**问题描述：** 错误消息的格式和风格不一致，不利于调试。

**修复建议：** 建立统一的错误消息约定：
```go
// 推荐格式：
"<package>: <operation> failed: <reason>"

例如：
"store: add memory failed: memory cannot be nil"
"postgres: query failed: connection refused"
"vector: search failed: dimension mismatch: expected 1536, got 1024"
```

**难度评估：** 低 - 仅需更新错误消息，不改变代码逻辑

---

## 二、Panic 处理分析

### 2.1 Goroutine 中缺少 Panic 保护

#### 问题6：ParallelRunner 中的 Goroutine 没有 Panic 恢复
**位置：** `/runner/runner.go` 第95-107行
```go
func (pr *ParallelRunner) RunParallel(ctx context.Context, tasks []*Task) []*Result {
    results := make([]*Result, len(tasks))
    var wg sync.WaitGroup

    for i, task := range tasks {
        wg.Add(1)
        go func(index int, t *Task) {
            defer wg.Done()  // 没有 panic 恢复！

            output, err := pr.runner.Run(ctx, t.Agent, t.Input)
            results[index] = &Result{
                TaskID: t.ID,
                Output: output,
                Error:  err,
            }
        }(i, task)
    }

    wg.Wait()
    return results
}
```

**问题描述：** 
- Goroutine 中如果 `pr.runner.Run()` 发生 panic，会导致整个程序崩溃
- 没有 defer recovery 来捕获 panic
- `wg.Done()` 之前没有 panic 处理，如果 panic 可能导致 `wg.Done()` 不被调用

**修复建议：**
```go
go func(index int, t *Task) {
    defer wg.Done()
    defer func() {
        if r := recover(); r != nil {
            results[index] = &Result{
                TaskID: t.ID,
                Error:  fmt.Errorf("task panicked: %v", r),
            }
        }
    }()
    
    output, err := pr.runner.Run(ctx, t.Agent, t.Input)
    results[index] = &Result{
        TaskID: t.ID,
        Output: output,
        Error:  err,
    }
}(i, task)
```

**难度评估：** 低 - 只需添加 defer recovery 块

---

### 2.2 Middleware 链中没有 Panic 保护

#### 问题7：中间件执行没有 Panic 恢复
**位置：** `/middleware/middleware.go` 第87-105行
```go
func (c *MiddlewareChain) Execute(ctx *Context, finalHandler Handler) error {
    return c.executeMiddleware(ctx, 0, finalHandler)
}

func (c *MiddlewareChain) executeMiddleware(ctx *Context, index int, finalHandler Handler) error {
    if index >= len(c.middlewares) {
        return finalHandler(ctx)  // 没有 panic 保护
    }

    nextHandler := func(ctx *Context) error {
        return c.executeMiddleware(ctx, index+1, finalHandler)  // 递归，没有 panic 保护
    }

    return c.middlewares[index].Execute(ctx, nextHandler)  // 中间件可能 panic
}
```

**问题描述：** 
- 任何中间件发生 panic 都会导致整个 Agent 崩溃
- 没有上层的 panic 恢复机制
- 特别是在递归调用时，panic 会直接传播

**修复建议：**
```go
func (c *MiddlewareChain) executeMiddleware(ctx *Context, index int, finalHandler Handler) error {
    defer func() {
        if r := recover(); r != nil {
            // 记录 panic 并设置错误
            ctx.Error = fmt.Errorf("middleware panic: %v", r)
        }
    }()
    
    if index >= len(c.middlewares) {
        return finalHandler(ctx)
    }

    nextHandler := func(ctx *Context) error {
        return c.executeMiddleware(ctx, index+1, finalHandler)
    }

    return c.middlewares[index].Execute(ctx, nextHandler)
}
```

**难度评估：** 中 - 需要处理 panic 时如何返回错误，避免改变返回类型

---

### 2.3 Agent.Run 中的 Panic 问题

#### 问题8：Agent.Run 的中间件执行缺少 Panic 保护
**位置：** `/agent/agent.go` 第188-275行
```go
func (a *Agent) Run(ctx context.Context, input string) (string, error) {
    mwCtx := middleware.NewContext(ctx)
    mwCtx.Input = input

    err := a.middlewares.Execute(mwCtx, func(mwCtx *middleware.Context) error {
        // 大量业务逻辑，如果任何地方 panic...
        // ...
        response, err := a.llm.Generate(mwCtx.Context(), a.ctx.GetMessages(), toolSchemas)
        if err != nil {
            return fmt.Errorf("LLM generation failed: %w", err)
        }
        // ...
    })

    if err != nil {
        return "", err
    }
    // ...
}
```

**问题描述：** 
- Lambda 中的任何 panic 都会导致 Agent 崩溃
- 特别是 `a.llm.Generate()` 的调用，LLM 提供者的代码质量不确定

**修复建议：**
```go
func (a *Agent) Run(ctx context.Context, input string) (string, error) {
    mwCtx := middleware.NewContext(ctx)
    mwCtx.Input = input

    err := a.middlewares.Execute(mwCtx, func(mwCtx *middleware.Context) error {
        defer func() {
            if r := recover(); r != nil {
                mwCtx.Error = fmt.Errorf("agent execution panicked: %v", r)
            }
        }()

        // ... 业务逻辑
    })

    if err != nil {
        return "", err
    }
    // ...
}
```

**难度评估：** 中 - 需要小心处理现有的错误流程

---

## 三、标准化错误分析

### 3.1 缺少自定义错误类型

#### 问题9：只用标准 error，缺少结构化错误信息
**位置1：** `/middleware/errors.go`
```go
var (
    ErrRateLimitExceeded = errors.New("rate limit exceeded")
    ErrInvalidInput = errors.New("invalid input")
    ErrMiddlewareChainFailed = errors.New("middleware chain failed")
    ErrInvalidContext = errors.New("invalid middleware context")
)
```

**位置2：** `/middleware/limiter/limiter.go`
```go
var (
    ErrRateLimitExceeded = errors.New("rate limit exceeded")  // 重复定义！
)
```

**问题描述：**
- 相同的错误定义在多个地方（middleware/errors.go 和 limiter.go）
- 没有结构化的错误信息（如错误代码、原始错误、上下文）
- 难以编程化地处理不同类型的错误

**修复建议：**
```go
// errors.go
package middleware

import "errors"

// ErrorCode 表示错误的类型
type ErrorCode string

const (
    ErrCodeRateLimitExceeded ErrorCode = "rate_limit_exceeded"
    ErrCodeInvalidInput      ErrorCode = "invalid_input"
    ErrCodeMiddlewareFailed  ErrorCode = "middleware_failed"
    ErrCodeValidationFailed  ErrorCode = "validation_failed"
)

// AppError 是自定义错误类型
type AppError struct {
    Code    ErrorCode
    Message string
    Cause   error  // 底层错误
    Context map[string]interface{}
}

func (e *AppError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
    return e.Cause
}

// 辅助函数
func NewAppError(code ErrorCode, message string, cause error) *AppError {
    return &AppError{
        Code:    code,
        Message: message,
        Cause:   cause,
        Context: make(map[string]interface{}),
    }
}

// 检查函数
func IsRateLimitExceeded(err error) bool {
    var appErr *AppError
    return errors.As(err, &appErr) && appErr.Code == ErrCodeRateLimitExceeded
}
```

**难度评估：** 高 - 需要修改所有错误创建和检查的代码

---

### 3.2 错误的可比较性问题

#### 问题10：无法使用 errors.Is() 检查特定错误
**位置1：** `/agent/agent.go` 第249-252行
```go
result, err := a.tools.Execute(mwCtx.Context(), toolCall.Name, toolCall.Args)
if err != nil {
    // 无法检查是什么类型的错误（工具未找到？参数无效？）
    result = fmt.Sprintf("Error executing tool %s: %v", toolCall.Name, err)
}
```

**位置2：** `/runner/runner.go` 第177-189行
```go
for _, ctask := range tasks {
    shouldRun := true
    if ctask.Condition != nil {
        var err error
        shouldRun, err = ctask.Condition(ctx, lastResult)
        if err != nil {
            return results, fmt.Errorf("condition evaluation failed: %w", err)
            // 无法区分不同的失败原因
        }
    }
    // ...
}
```

**问题描述：**
- 代码只能使用 `if err != nil` 判断
- 无法使用 `errors.Is(err, targetErr)` 检查特定错误类型
- 这使得错误处理不够灵活和精确

**修复建议：**
```go
// 使用自定义错误类型
var (
    ErrToolNotFound = &AppError{Code: ErrCodeToolNotFound}
    ErrInvalidArgs  = &AppError{Code: ErrCodeInvalidArgs}
    ErrToolExecution = &AppError{Code: ErrCodeToolExecution}
)

// 在调用方
result, err := a.tools.Execute(ctx, toolCall.Name, toolCall.Args)
if err != nil {
    if errors.Is(err, ErrToolNotFound) {
        // 工具未找到，可能是 LLM 的幻觉
        result = fmt.Sprintf("Tool %s not available", toolCall.Name)
    } else if errors.Is(err, ErrInvalidArgs) {
        // 参数无效，返回给 LLM 继续修正
        result = fmt.Sprintf("Invalid arguments for tool %s: %v", toolCall.Name, err)
    } else {
        // 其他执行错误
        result = fmt.Sprintf("Error executing tool %s: %v", toolCall.Name, err)
    }
}
```

**难度评估：** 中 - 需要定义错误类型并更新检查点

---

### 3.3 错误的上下文信息不足

#### 问题11：错误信息缺少足够的上下文
**位置1：** `/memory/store/postgres.go` 第179行
```go
err := rows.Scan(&mem.ID, &mem.Content, &metadataJSON, &mem.CreatedAt, &mem.UpdatedAt)
if err != nil {
    return nil, fmt.Errorf("failed to scan memory: %w", err)
    // 缺少：哪一行？什么操作？
}
```

**位置2：** `/vector/store/pgvector.go` 第184-187行
```go
err := rows.Scan(&id, &text, &vectorStr)
if err != nil {
    return nil, fmt.Errorf("failed to scan embedding: %w", err)
    // 缺少：第几条记录？什么查询？
}
```

**位置3：** `/agent/agent.go` 第262-263行
```go
mwCtx.Error = fmt.Errorf("max iterations (%d) reached", a.maxIterations)
return mwCtx.Error
// 缺少：为什么还没有完成？最后一次 LLM 调用返回了什么？
```

**问题描述：**
- 错误消息没有提供足够的调试信息
- 无法快速定位问题发生的位置
- 特别是在分布式系统中难以追踪

**修复建议：**
```go
// 方案1：增加上下文信息
func (s *PostgresStore) SearchMemory(ctx context.Context, query string) ([]*memory.Memory, error) {
    rows, err := s.db.QueryContext(ctx, sqlQuery, params...)
    if err != nil {
        return nil, fmt.Errorf("postgres search: query failed (query=%s): %w", 
            redactedQuery, err)
    }
    
    memories := make([]*memory.Memory, 0)
    rowNum := 0
    for rows.Next() {
        rowNum++
        mem := &memory.Memory{}
        err := rows.Scan(...)
        if err != nil {
            return nil, fmt.Errorf("postgres search: scan row %d failed (query=%s): %w", 
                rowNum, redactedQuery, err)
        }
        memories = append(memories, mem)
    }
    // ...
}

// 方案2：使用错误包装和注解
type ScanError struct {
    Row    int
    Query  string
    Cause  error
}

func (e *ScanError) Error() string {
    return fmt.Sprintf("failed to scan row %d: %v (query: %s)", e.Row, e.Cause, e.Query)
}
```

**难度评估：** 低-中 - 需要修改很多地方但无需改变结构

---

## 四、总结表格

| 问题ID | 问题 | 严重程度 | 修复难度 | 优先级 |
|--------|------|--------|--------|--------|
| 1 | RegisterTool/AddMiddleware 签名不一致 | 中 | 低 | 高 |
| 2 | 存储接口错误返回不一致 | 中 | 中 | 高 |
| 3 | Runner 错误处理不一致 | 中 | 中 | 中 |
| 4 | 错误包装方式不一致 | 低 | 低 | 低 |
| 5 | 错误消息格式不统一 | 低 | 低 | 低 |
| 6 | Goroutine 缺少 Panic 恢复 | 高 | 低 | 最高 |
| 7 | Middleware 链缺少 Panic 恢复 | 高 | 中 | 最高 |
| 8 | Agent.Run 缺少 Panic 恢复 | 高 | 中 | 最高 |
| 9 | 缺少自定义错误类型 | 中 | 高 | 高 |
| 10 | 错误不可比较 | 中 | 中 | 高 |
| 11 | 错误上下文不足 | 低 | 低 | 中 |

---

## 五、改进路线图

### 第一阶段（最高优先级 - 防止崩溃）
1. 为所有 Goroutine 添加 Panic 恢复 (问题 6, 7, 8)
2. 统一 Store 接口错误处理 (问题 2)

### 第二阶段（高优先级 - 改进可维护性）
1. 定义自定义错误类型 (问题 9)
2. 修复错误包装 (问题 4)
3. 标准化方法签名 (问题 1, 3)

### 第三阶段（中优先级 - 增强调试）
1. 添加错误上下文 (问题 11)
2. 统一错误消息格式 (问题 5)
3. 实现 errors.Is() 支持 (问题 10)

