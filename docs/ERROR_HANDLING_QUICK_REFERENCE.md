# ai-allin 错误处理快速参考指南

## 问题速览

| ID | 问题 | 文件 | 行号 | 严重性 | 修复时间 |
|----|------|------|------|--------|---------|
| 1 | 方法签名不一致 | agent/agent.go | 159-161 | 中 | 15分钟 |
| 2 | Store接口不一致 | memory/store/*.go | 多处 | 中 | 30分钟 |
| 3 | Runner错误不一致 | runner/runner.go | 91-153 | 中 | 20分钟 |
| 4 | 错误包装不完整 | vector/store/pgvector.go | 294 | 低 | 10分钟 |
| 5 | 错误消息格式乱 | 全项目 | 多处 | 低 | 20分钟 |
| 6 | Goroutine无panic保护 | runner/runner.go | 95-107 | **高** | **5分钟** |
| 7 | Middleware无panic保护 | middleware/middleware.go | 87-105 | **高** | **10分钟** |
| 8 | Agent.Run无panic保护 | agent/agent.go | 188-275 | **高** | **10分钟** |
| 9 | 缺自定义错误类型 | middleware/errors.go | 全部 | 中 | 45分钟 |
| 10 | 错误不可比较 | 全项目 | 多处 | 中 | 30分钟 |
| 11 | 错误上下文不足 | memory/store/*.go | 多处 | 低 | 25分钟 |

## 立即修复（最高优先级 - 防止崩溃）

### 修复1：ParallelRunner Goroutine Panic 恢复
**文件：** `/runner/runner.go` 第97-98行

```go
// 修改前
go func(index int, t *Task) {
    defer wg.Done()
    output, err := pr.runner.Run(ctx, t.Agent, t.Input)

// 修改后
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
```

### 修复2：Middleware 链 Panic 恢复
**文件：** `/middleware/middleware.go` 第92-105行

```go
// 修改前的 executeMiddleware 函数
func (c *MiddlewareChain) executeMiddleware(ctx *Context, index int, finalHandler Handler) error {
    if index >= len(c.middlewares) {
        return finalHandler(ctx)
    }
    // ...
    return c.middlewares[index].Execute(ctx, nextHandler)  // 可能panic
}

// 修改后
func (c *MiddlewareChain) executeMiddleware(ctx *Context, index int, finalHandler Handler) error {
    defer func() {
        if r := recover(); r != nil {
            if ctx.Error == nil {
                ctx.Error = fmt.Errorf("middleware panic at index %d: %v", index, r)
            }
        }
    }()
    
    if index >= len(c.middlewares) {
        return finalHandler(ctx)
    }
    
    nextHandler := func(ctx *Context) error {
        return c.executeMiddleware(ctx, index+1, finalHandler)
    }
    
    if err := c.middlewares[index].Execute(ctx, nextHandler); err != nil {
        return err
    }
    
    if ctx.Error != nil {
        return ctx.Error
    }
    
    return nil
}
```

### 修复3：Agent.Run 中的业务逻辑 Panic 恢复
**文件：** `/agent/agent.go` 第194-264行

```go
// 在 a.middlewares.Execute(mwCtx, func(mwCtx *middleware.Context) error { 
// 后面立即添加

err := a.middlewares.Execute(mwCtx, func(mwCtx *middleware.Context) error {
    defer func() {
        if r := recover(); r != nil {
            mwCtx.Error = fmt.Errorf("agent execution panicked: %v", r)
        }
    }()
    
    // 原有的业务逻辑
    userMsg := message.NewMessage(message.RoleUser, input)
    // ...
})
```

---

## 高优先级修复（改进稳定性）

### 修复4：修复 pgvector 中的错误忽略
**文件：** `/vector/store/pgvector.go` 第285-298行

```go
// 修改前
func (s *PGVectorStore) stringToVector(str string) []float32 {
    // ...
    for _, part := range parts {
        var v float32
        fmt.Sscanf(strings.TrimSpace(part), "%f", &v)  // 错误被忽略
        vec = append(vec, v)
    }
    return vec
}

// 修改后
func (s *PGVectorStore) stringToVector(str string) ([]float32, error) {
    // ...
    for i, part := range parts {
        var v float32
        n, err := fmt.Sscanf(strings.TrimSpace(part), "%f", &v)
        if err != nil || n != 1 {
            return nil, fmt.Errorf("failed to parse vector component %d (%s): %w", 
                i, part, err)
        }
        vec = append(vec, v)
    }
    return vec, nil
}

// 更新调用处：Search() 和 GetEmbedding()
vec, err := s.stringToVector(vectorStr)
if err != nil {
    return nil, fmt.Errorf("failed to parse embedding vector: %w", err)
}
```

### 修复5：统一 Agent 方法签名
**文件：** `/agent/agent.go` 第159-161行

```go
// 修改前
func (a *Agent) AddMiddleware(m middleware.Middleware) {
    a.middlewares.Add(m)
}

// 修改后
func (a *Agent) AddMiddleware(m middleware.Middleware) error {
    if m == nil {
        return fmt.Errorf("middleware cannot be nil")
    }
    return a.middlewares.Add(m)
}

// 同时修改 MiddlewareChain.Add() 使其返回 error
func (c *MiddlewareChain) Add(m Middleware) error {
    if m == nil {
        return fmt.Errorf("middleware cannot be nil")
    }
    c.middlewares = append(c.middlewares, m)
    return nil
}
```

---

## 中优先级修复（代码质量）

### 修复6：统一 Store 接口
**文件：** `/memory/memory.go`

```go
// 修改 MemoryStore 接口定义
type MemoryStore interface {
    AddMemory(context.Context, *Memory) error
    SearchMemory(context.Context, string) ([]*Memory, error)
    // 明确声明这些方法的签名（可选，看是否有不同的实现）
    Clear(context.Context) error
    Count(context.Context) (int, error)
}

// 然后确保所有实现都符合这个接口
// InMemoryStore.Count() 应改为：
func (s *InMemoryStore) Count(ctx context.Context) (int, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return len(s.memories), nil
}
```

### 修复7：创建自定义错误类型（可选，提高代码质量）
**新文件：** 保留 `middleware/errors.go` 并扩展

```go
// 添加到 middleware/errors.go
package middleware

import (
    "errors"
    "fmt"
)

// ErrorCode 用于错误分类
type ErrorCode string

const (
    ErrCodeRateLimitExceeded ErrorCode = "RATE_LIMIT_EXCEEDED"
    ErrCodeValidationFailed  ErrorCode = "VALIDATION_FAILED"
    ErrCodePanicRecovered    ErrorCode = "PANIC_RECOVERED"
)

// AppError 结构化错误
type AppError struct {
    Code    ErrorCode
    Message string
    Cause   error
    Context map[string]interface{}
}

func (e *AppError) Error() string {
    msg := fmt.Sprintf("[%s] %s", e.Code, e.Message)
    if e.Cause != nil {
        msg += fmt.Sprintf(" (%v)", e.Cause)
    }
    return msg
}

func (e *AppError) Unwrap() error {
    return e.Cause
}

// 判断错误类型
func IsRateLimitExceeded(err error) bool {
    var appErr *AppError
    if errors.As(err, &appErr) {
        return appErr.Code == ErrCodeRateLimitExceeded
    }
    return false
}
```

---

## 修复顺序建议

### 第1天（30分钟内完成）
1. 修复 ParallelRunner Goroutine Panic 保护 ✓
2. 修复 Middleware 链 Panic 保护 ✓
3. 修复 Agent.Run Panic 保护 ✓

### 第2天（1小时）
1. 修复 pgvector stringToVector 错误处理 ✓
2. 统一 Agent 方法签名 ✓
3. 统一 Store 接口 ✓

### 第3天（按需）
1. 创建自定义错误类型
2. 更新错误消息格式
3. 添加错误上下文信息

---

## 测试清单

修复后应该执行以下测试：

```bash
# 1. 单元测试
go test ./runner/... -v
go test ./middleware/... -v
go test ./agent/... -v
go test ./memory/store/... -v
go test ./vector/store/... -v

# 2. 集成测试
go test ./... -v

# 3. 运行示例确保功能正常
go run ./examples/basic/main.go
go run ./examples/streaming/main.go
```

### 关键测试场景

**Panic 恢复测试：**
```go
// 创建一个会 panic 的 Task
panicTask := &Task{
    ID: "panic_task",
    Agent: agentWithBadMiddleware(),  // 会 panic
    Input: "test",
}

results := runner.RunParallel(ctx, []*Task{panicTask})
// 应该得到 Result.Error != nil，而不是程序崩溃
```

**错误处理测试：**
```go
// 测试向量解析错误
vectorStore := pgvector.New(config)
badVector := "[1.0, 2.0, invalid, 4.0]"
_, err := vectorStore.stringToVector(badVector)
// 应该返回有意义的错误信息
assert(err != nil)
assert(strings.Contains(err.Error(), "failed to parse"))
```

---

## 常见问题

**Q: 修复 Panic 恢复会改变 API 吗？**
A: 不会。`defer recover()` 只是添加到函数内部，不改变公开的函数签名。

**Q: 需要修改调用方吗？**
A: 对于 Panic 恢复修复，不需要。对于签名更改（如添加 error 返回），需要更新调用方。

**Q: 这些修复的性能影响？**
A: 
- Panic 恢复：极小（只有发生 panic 时才有开销）
- 错误检查：无性能影响
- 自定义错误类型：无性能影响

**Q: 可以分阶段修复吗？**
A: 可以。建议先修复 Panic（最高优先级），再修复签名（无 API 中断），最后修复错误类型（可选）。

---

## 参考文档

- 详细分析报告：`docs/ERROR_HANDLING_ANALYSIS.md`
- 实现示例代码：`docs/IMPLEMENTATION_EXAMPLES.md`
- Go 错误处理最佳实践：https://golang.org/doc/effective_go#errors

