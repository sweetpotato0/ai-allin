# ai-allin 项目错误处理改进实现示例

## 示例代码

### 问题6：Goroutine Panic 恢复 - 修复前后对比

#### 修复前 (runner/runner.go)
```go
func (pr *ParallelRunner) RunParallel(ctx context.Context, tasks []*Task) []*Result {
    results := make([]*Result, len(tasks))
    var wg sync.WaitGroup

    for i, task := range tasks {
        wg.Add(1)
        go func(index int, t *Task) {
            defer wg.Done()  // 潜在风险：panic 时不会执行

            output, err := pr.runner.Run(ctx, t.Agent, t.Input)
            results[index] = &Result{
                TaskID: t.ID,
                Output: output,
                Error:  err,
            }
        }(i, task)  // ← 可能 panic
    }

    wg.Wait()
    return results
}
```

#### 修复后
```go
func (pr *ParallelRunner) RunParallel(ctx context.Context, tasks []*Task) []*Result {
    results := make([]*Result, len(tasks))
    var wg sync.WaitGroup

    for i, task := range tasks {
        wg.Add(1)
        go func(index int, t *Task) {
            defer wg.Done()
            // 添加 panic 恢复
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
    }

    wg.Wait()
    return results
}
```

---

### 问题7：Middleware 链 Panic 恢复 - 修复示例

#### 修复前 (middleware/middleware.go)
```go
func (c *MiddlewareChain) executeMiddleware(ctx *Context, index int, finalHandler Handler) error {
    if index >= len(c.middlewares) {
        return finalHandler(ctx)
    }

    nextHandler := func(ctx *Context) error {
        return c.executeMiddleware(ctx, index+1, finalHandler)
    }

    // ← 中间件可能 panic，无保护
    return c.middlewares[index].Execute(ctx, nextHandler)
}
```

#### 修复后
```go
func (c *MiddlewareChain) executeMiddleware(ctx *Context, index int, finalHandler Handler) error {
    defer func() {
        if r := recover(); r != nil {
            // 将 panic 转换为 error，存储在 context 中
            if ctx.Error == nil {
                ctx.Error = fmt.Errorf("middleware %d panicked: %v", index, r)
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
    
    // 检查是否发生了 panic
    if ctx.Error != nil {
        return ctx.Error
    }
    
    return nil
}
```

**更优雅的方案：创建 RecoveryMiddleware**
```go
// middleware/recovery/recovery.go
package recovery

import (
    "fmt"
    "github.com/sweetpotato0/ai-allin/middleware"
)

// RecoveryMiddleware 捕获 panic 并转换为 error
type RecoveryMiddleware struct {
    logFunc func(string)  // 可选的日志函数
}

func NewRecoveryMiddleware(logFunc func(string)) *RecoveryMiddleware {
    return &RecoveryMiddleware{logFunc: logFunc}
}

func (m *RecoveryMiddleware) Name() string {
    return "Recovery"
}

func (m *RecoveryMiddleware) Execute(ctx *middleware.Context, next middleware.Handler) error {
    defer func() {
        if r := recover(); r != nil {
            if m.logFunc != nil {
                m.logFunc(fmt.Sprintf("Panic recovered: %v", r))
            }
            ctx.Error = fmt.Errorf("panic: %v", r)
        }
    }()
    
    return next(ctx)
}
```

**使用方式：**
```go
agent := agent.New(
    agent.WithProvider(provider),
    agent.WithMiddlewares(
        recovery.NewRecoveryMiddleware(logger.Printf),  // 最外层
        logger.NewRequestLogger(logger.Printf),
        validator.NewInputValidator(validateInput),
        // ... 其他中间件
    ),
)
```

---

### 问题4：错误包装修复 - pgvector.go 示例

#### 修复前
```go
func (s *PGVectorStore) stringToVector(str string) []float32 {
    str = strings.TrimPrefix(str, "[")
    str = strings.TrimSuffix(str, "]")
    parts := strings.Split(str, ",")

    vec := make([]float32, 0, len(parts))
    for _, part := range parts {
        var v float32
        fmt.Sscanf(strings.TrimSpace(part), "%f", &v)  // ← 错误被忽略！
        vec = append(vec, v)
    }
    return vec
}
```

#### 修复后
```go
func (s *PGVectorStore) stringToVector(str string) ([]float32, error) {
    str = strings.TrimPrefix(str, "[")
    str = strings.TrimSuffix(str, "]")
    parts := strings.Split(str, ",")

    vec := make([]float32, 0, len(parts))
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

// 相应更新调用处
func (s *PGVectorStore) Search(ctx context.Context, queryVector []float32, topK int) ([]*vector.Embedding, error) {
    // ...
    for rows.Next() {
        var id, text string
        var vectorStr string

        err := rows.Scan(&id, &text, &vectorStr)
        if err != nil {
            return nil, fmt.Errorf("failed to scan embedding: %w", err)
        }

        vec, err := s.stringToVector(vectorStr)  // ← 现在检查错误
        if err != nil {
            return nil, fmt.Errorf("failed to parse embedding vector: %w", err)
        }
        
        embeddings = append(embeddings, &vector.Embedding{
            ID:     id,
            Text:   text,
            Vector: vec,
        })
    }
    // ...
}
```

---

### 问题9：自定义错误类型示例

#### 创建 errors.go
```go
// middleware/errors.go
package middleware

import (
    "errors"
    "fmt"
)

// ErrorCode 表示错误类型的代码
type ErrorCode string

const (
    ErrCodeRateLimitExceeded ErrorCode = "RATE_LIMIT_EXCEEDED"
    ErrCodeValidationFailed  ErrorCode = "VALIDATION_FAILED"
    ErrCodeMiddlewareFailed  ErrorCode = "MIDDLEWARE_FAILED"
    ErrCodePanicRecovered    ErrorCode = "PANIC_RECOVERED"
    ErrCodeInvalidContext    ErrorCode = "INVALID_CONTEXT"
)

// AppError 是结构化的应用错误
type AppError struct {
    Code      ErrorCode
    Message   string
    Cause     error
    Context   map[string]interface{}
}

// Error 实现 error 接口
func (e *AppError) Error() string {
    msg := fmt.Sprintf("[%s] %s", e.Code, e.Message)
    if e.Cause != nil {
        msg += fmt.Sprintf(" (cause: %v)", e.Cause)
    }
    return msg
}

// Unwrap 支持 errors.Is() 和 errors.As()
func (e *AppError) Unwrap() error {
    return e.Cause
}

// WithContext 添加上下文信息
func (e *AppError) WithContext(key string, value interface{}) *AppError {
    if e.Context == nil {
        e.Context = make(map[string]interface{})
    }
    e.Context[key] = value
    return e
}

// NewAppError 创建一个新的 AppError
func NewAppError(code ErrorCode, message string, cause error) *AppError {
    return &AppError{
        Code:    code,
        Message: message,
        Cause:   cause,
        Context: make(map[string]interface{}),
    }
}

// IsErrorCode 检查是否为特定错误码
func IsErrorCode(err error, code ErrorCode) bool {
    var appErr *AppError
    return errors.As(err, &appErr) && appErr.Code == code
}

// 便捷函数
func IsRateLimitExceeded(err error) bool {
    return IsErrorCode(err, ErrCodeRateLimitExceeded)
}

func IsValidationFailed(err error) bool {
    return IsErrorCode(err, ErrCodeValidationFailed)
}
```

#### 使用示例
```go
// middleware/limiter/limiter.go
package limiter

import (
    "github.com/sweetpotato0/ai-allin/middleware"
)

func (m *RateLimiter) Execute(ctx *middleware.Context, next middleware.Handler) error {
    m.mu.Lock()
    if m.counter >= m.maxRequests {
        m.mu.Unlock()
        // 使用结构化错误
        return middleware.NewAppError(
            middleware.ErrCodeRateLimitExceeded,
            fmt.Sprintf("rate limit exceeded: %d/%d", m.counter, m.maxRequests),
            nil,
        ).WithContext("max_requests", m.maxRequests).
           WithContext("current_count", m.counter)
    }
    m.counter++
    m.mu.Unlock()
    return next(ctx)
}

// 在调用方检查
result, err := agent.Run(ctx, input)
if err != nil {
    if middleware.IsRateLimitExceeded(err) {
        // 处理限流，可能返回等待提示
        log.Printf("Rate limited: %v", err)
        return "", fmt.Errorf("请稍后再试")
    }
    if middleware.IsValidationFailed(err) {
        // 处理验证失败
        var appErr *middleware.AppError
        errors.As(err, &appErr)
        if details, ok := appErr.Context["validation_details"]; ok {
            return "", fmt.Errorf("输入不合法: %v", details)
        }
    }
    return "", err
}
```

---

### 问题11：添加错误上下文示例

#### 修复前
```go
func (s *PostgresStore) SearchMemory(ctx context.Context, query string) ([]*memory.Memory, error) {
    var rows *sql.Rows
    var err error

    if query == "" {
        rows, err = s.db.QueryContext(ctx,
            `SELECT id, content, metadata, created_at, updated_at
             FROM memories
             ORDER BY created_at DESC`)
    } else {
        searchQuery := fmt.Sprintf("%%%s%%", query)
        rows, err = s.db.QueryContext(ctx,
            `SELECT id, content, metadata, created_at, updated_at
             FROM memories
             WHERE content ILIKE $1
             ORDER BY created_at DESC`,
            searchQuery)
    }

    if err != nil {
        return nil, fmt.Errorf("failed to search memories: %w", err)  // ← 信息不足
    }
    // ...
}
```

#### 修复后
```go
func (s *PostgresStore) SearchMemory(ctx context.Context, query string) ([]*memory.Memory, error) {
    var rows *sql.Rows
    var err error

    sqlQuery := `SELECT id, content, metadata, created_at, updated_at FROM memories`
    args := []interface{}{}
    
    if query == "" {
        sqlQuery += " ORDER BY created_at DESC"
    } else {
        sqlQuery += " WHERE content ILIKE $1 ORDER BY created_at DESC"
        args = append(args, fmt.Sprintf("%%%s%%", query))
    }

    rows, err = s.db.QueryContext(ctx, sqlQuery, args...)
    if err != nil {
        return nil, middleware.NewAppError(
            middleware.ErrCodeMiddlewareFailed,
            "failed to query memories from database",
            err,
        ).WithContext("query", sqlQuery).
           WithContext("search_term", query).
           WithContext("database", "postgres")
    }
    defer rows.Close()

    memories := make([]*memory.Memory, 0)
    rowNum := 0
    for rows.Next() {
        rowNum++
        mem := &memory.Memory{}
        var metadataJSON string

        err := rows.Scan(&mem.ID, &mem.Content, &metadataJSON, &mem.CreatedAt, &mem.UpdatedAt)
        if err != nil {
            return nil, middleware.NewAppError(
                middleware.ErrCodeMiddlewareFailed,
                "failed to scan memory row from database",
                err,
            ).WithContext("row_number", rowNum).
               WithContext("memory_id", mem.ID).
               WithContext("operation", "scan")
        }

        // 解析 metadata
        mem.Metadata = make(map[string]interface{})
        if metadataJSON != "" && metadataJSON != "{}" {
            err := json.Unmarshal([]byte(metadataJSON), &mem.Metadata)
            if err != nil {
                return nil, middleware.NewAppError(
                    middleware.ErrCodeMiddlewareFailed,
                    "failed to unmarshal metadata JSON",
                    err,
                ).WithContext("row_number", rowNum).
                   WithContext("memory_id", mem.ID).
                   WithContext("metadata_raw", metadataJSON)
            }
        }

        memories = append(memories, mem)
    }

    if err = rows.Err(); err != nil {
        return nil, middleware.NewAppError(
            middleware.ErrCodeMiddlewareFailed,
            "error iterating memory rows",
            err,
        ).WithContext("total_rows_processed", rowNum).
           WithContext("operation", "iterate")
    }

    return memories, nil
}
```

---

### 问题1：统一方法签名示例

#### 修复前
```go
// agent/agent.go
func (a *Agent) RegisterTool(t *tool.Tool) error {
    return a.tools.Register(t)  // 返回 error
}

func (a *Agent) RegisterPrompt(name, content string) error {
    return a.promptManager.RegisterString(name, content)  // 返回 error
}

func (a *Agent) AddMiddleware(m middleware.Middleware) {
    a.middlewares.Add(m)  // 不返回 error - 不一致！
}
```

#### 修复后
```go
// agent/agent.go
func (a *Agent) RegisterTool(t *tool.Tool) error {
    if t == nil {
        return fmt.Errorf("tool cannot be nil")
    }
    return a.tools.Register(t)
}

func (a *Agent) RegisterPrompt(name, content string) error {
    if name == "" {
        return fmt.Errorf("prompt name cannot be empty")
    }
    return a.promptManager.RegisterString(name, content)
}

// 选项1：让 AddMiddleware 也返回 error（保持一致）
func (a *Agent) AddMiddleware(m middleware.Middleware) error {
    if m == nil {
        return fmt.Errorf("middleware cannot be nil")
    }
    return a.middlewares.Add(m)  // 需要修改 MiddlewareChain.Add
}

// 或选项2：保持当前不返回 error，但添加验证
func (a *Agent) AddMiddleware(m middleware.Middleware) {
    if m != nil {
        a.middlewares.Add(m)
    }
}
```

#### 对应的 MiddlewareChain 修改
```go
// middleware/middleware.go
type MiddlewareChain struct {
    middlewares []Middleware
}

// 添加验证
func (c *MiddlewareChain) Add(m Middleware) error {
    if m == nil {
        return fmt.Errorf("middleware cannot be nil")
    }
    if c.middlewares == nil {
        c.middlewares = make([]Middleware, 0)
    }
    c.middlewares = append(c.middlewares, m)
    return nil
}
```

---

## 错误处理最佳实践总结

### 1. 使用链式错误包装
```go
// 好的做法
if err := someOperation(); err != nil {
    return fmt.Errorf("operation failed: %w", err)
}

// 不好的做法
if err := someOperation(); err != nil {
    return err  // 丢失上下文
}
```

### 2. 为 Goroutine 添加恢复
```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Goroutine panicked: %v", r)
        }
    }()
    // 业务逻辑
}()
```

### 3. 使用结构化错误
```go
// 避免
return errors.New("something went wrong")

// 推荐
return &AppError{
    Code:    ErrCodeOperation,
    Message: "operation X failed",
    Cause:   originalErr,
    Context: map[string]interface{}{
        "operation_id": opID,
        "retry_count":  retries,
    },
}
```

### 4. 提供足够的上下文
```go
return fmt.Errorf("failed to process item %d in batch %s: %w", 
    itemIndex, batchID, err)
```

### 5. 使用 errors.Is() 检查特定错误
```go
if errors.Is(err, ErrRateLimitExceeded) {
    // 处理限流
}
```

