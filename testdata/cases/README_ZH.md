
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
