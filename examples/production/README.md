# 生产级电商智能客服平台

## 📋 项目概述

这是一个利用 `ai-allin` 框架构建的**生产级电商智能客服平台**，完整演示了如何在实际应用中集成和使用框架的所有核心组件。

### 🎯 核心功能

- **多Agent协调**: 4种专业Agent并行运作
  - 💬 **客服Agent**: 处理客户咨询和问题
  - 📊 **运营Agent**: 分析客户行为和趋势
  - ✅ **QA Agent**: 审查服务质量
  - 📚 **知识Agent**: 管理知识库

- **完整的Session和Context管理**
  - 每个客户一个独立Session
  - 每个Agent一个独立Context
  - 支持多轮对话和历史追踪

- **生产级特性**
  - 高并发处理（10000+ sessions）
  - 实时监控和告警
  - 完整的错误恢复
  - 数据持久化和备份

---

## 🏗️ 架构设计

```
┌─────────────────────────────────────────────────────┐
│        ECommerceServicePlatform                      │
│  (平台主控制层)                                      │
├─────────────────────────────────────────────────────┤
│                                                      │
│  ┌──────────────────────────────────────────────┐  │
│  │  多Agent协调层                              │  │
│  │  ┌────────────┬────────────┬────────────┐   │  │
│  │  │ CS Agent   │ Op Agent   │ QA Agent   │   │  │
│  │  │ (Context)  │ (Context)  │ (Context)  │   │  │
│  │  └──────┬─────┴──────┬─────┴──────┬─────┘   │  │
│  │         │            │            │         │  │
│  │  ┌──────v──────v─────v──────v────────────┐ │  │
│  │  │  Session Manager                      │ │  │
│  │  │  (管理多个客户会话)                   │ │  │
│  │  └──────┬──────────────────────────────┘ │  │
│  │         │                                 │  │
│  └─────────┼─────────────────────────────────┘  │
│            │                                     │
├────────────┼──────────────────────────────────────┤
│ 中间件层   │                                      │
│ Logger ────┼─── Limiter ─── Enricher ─── ErrorHandler
├────────────┼──────────────────────────────────────┤
│ 数据层     │                                      │
│            ├─ PostgreSQL (订单/客户/工单)        │
│            ├─ Redis (缓存/Session)               │
│            ├─ Vector Store (知识库)              │
│            └─ Memory Store (对话记忆)            │
└────────────────────────────────────────────────────┘
```

---

## 🚀 快速开始

### 前置要求

- Docker & Docker Compose
- Go 1.20+
- OpenAI API Key（或其他LLM提供商）

### 1️⃣ 环境准备

```bash
# 克隆项目
cd examples/production

# 配置环境变量
cat > .env << EOF
OPENAI_API_KEY=your-api-key
DB_PASSWORD=your-db-password
REDIS_PASSWORD=your-redis-password
GRAFANA_PASSWORD=admin
EOF

# 初始化数据库
docker-compose up postgres redis elasticsearch -d
```

### 2️⃣ 启动服务

```bash
# 启动完整的Docker Compose栈
docker-compose up -d

# 查看日志
docker-compose logs -f app

# 检查服务健康状态
docker-compose ps
```

### 3️⃣ 运行演示

```bash
# 构建项目
go build -o platform main.go ecommerce_service.go

# 运行完整演示
./platform

# 输出示例：
# ╔════════════════════════════════════════════════════════════════╗
# ║   生产级电商智能客服平台 - 完整功能演示                      ║
# ║   Production-Grade E-Commerce AI Customer Service Platform   ║
# ╚════════════════════════════════════════════════════════════════╝
```

---

## 📊 演示场景详解

### 场景1: 单客户咨询处理

**流程**:
1. 验证客户身份
2. 创建独立Session
3. 配置中间件（日志、限流、错误处理）
4. 调用客服Agent处理
5. 生成工单记录
6. 保存对话历史

```go
// 代码示例
response, _ := platform.HandleCustomerInquiry("CUST001", "订单查询...")
// ✓ Session创建: cs_CUST001_1234567890
// ✓ Context消息数: 3
// ✓ 工单创建: TKT_1234567890
```

### 场景2: 多轮对话（核心特性）

**关键点**:
- **Session**: 代表一个客户的完整对话会话
- **Context**: 记录本Session内所有消息历史
- **多轮交互**: 每次Run()都会向Context添加新消息

```
用户轮次1: "查询订单"
  ↓ Session.Run()
  ↓ Agent处理 + Context记录消息
  ↓ Context消息数: 2 (user + agent)

用户轮次2: "什么时候到？"
  ↓ Session.Run()
  ↓ Agent处理（可以看到之前的消息历史）
  ↓ Context消息数: 4

用户轮次3: "可以退货吗？"
  ↓ Session.Run()
  ↓ Agent处理（有完整的对话上下文）
  ↓ Context消息数: 6
```

### 场景3: 多Agent协调（完整工作流）

**执行顺序**:
1. **客服Agent** (CS Agent)
   - 处理客户问题
   - 提供初步方案
   - Context记录对话

2. **运营Agent** (Operation Agent)
   - 分析客户价值
   - 评估VIP等级
   - 预测流失风险

3. **QA Agent** (QA Agent)
   - 审查客服回复
   - 检查政策合规
   - 提供改进建议

4. **知识Agent** (Knowledge Agent)
   - 识别知识缺口
   - 建议文档更新
   - 改进知识库

```
流程图:
Customer Inquiry
    ↓
[CS Agent]  ← Context记录对话
    ↓
[Operation Agent]  ← 分析客户价值
    ↓
[QA Agent]  ← 审查质量
    ↓
[Knowledge Agent]  ← 更新知识库
    ↓
完整解决方案 + 改进建议
```

### 场景4: 并行处理

同时处理多个客户的咨询，每个Session独立运行。

```
客户1 ──[Session_1]──[Agent]──[Context]
                                   ↓
客户2 ──[Session_2]──[Agent]──[Context]
                                   ↓
客户3 ──[Session_3]──[Agent]──[Context]
                                   ↓
客户4 ──[Session_4]──[Agent]──[Context]
                                   ↓
客户5 ──[Session_5]──[Agent]──[Context]

✓ 5个独立Session并行运行
✓ SessionManager维护所有Session
✓ 不相互干扰
```

---

## 🔧 框架组件使用

### 1. Session (会话管理)

```go
// 创建Session
sess, err := sessionManager.Create(ctx, sessionID, agent)
defer sess.Close()

// Session内执行对话
response, err := sess.Run(ctx, userInput)

// 获取Session中的消息
messages := sess.GetMessages()

// 查询Session状态
state := sess.GetState()  // "active" / "inactive" / "closed"

// 管理所有Sessions
sessionManager.List()      // 列出所有Session IDs
sessionManager.Count()     // 获取活跃Session数
sessionManager.Delete(id)  // 删除Session
```

### 2. Context (对话历史)

```go
// Agent内部自动管理Context
// 每次Run()时会自动记录消息

// Context特性:
// - 自动维护消息历史
// - 支持Role-based筛选
// - 自动裁剪超大对话
// - 线程安全

// 获取所有消息
messages := agent.GetMessages()

// 按Role筛选
userMsgs := agent.GetMessagesByRole(message.RoleUser)

// 获取最后一条消息
lastMsg := agent.GetLastMessage()

// 清空历史
agent.Clear()
```

### 3. Agent (智能代理)

```go
// 创建Agent
ag := agent.New(
    agent.WithName("customer_service"),
    agent.WithSystemPrompt(systemPrompt),
    agent.WithMaxIterations(8),
    agent.WithTemperature(0.6),
    agent.WithMemory(memoryStore),
    agent.WithTools(true),
)

// 添加中间件
ag.AddMiddleware(logger.NewRequestLogger("service"))
ag.AddMiddleware(limiter.NewRateLimiter(100, time.Second))

// 在Session中运行
sess, _ := sessionManager.Create(ctx, sessionID, ag)
response, _ := sess.Run(ctx, input)
```

### 4. Middleware (中间件)

```go
// 请求日志
ag.AddMiddleware(logger.NewRequestLogger("service"))

// 速率限制
ag.AddMiddleware(limiter.NewRateLimiter(
    100,           // 请求数
    time.Second,   // 时间窗口
))

// 上下文增强
ag.AddMiddleware(enricher.NewContextEnricher(map[string]string{
    "session_id": sessionID,
    "customer_id": customerID,
}))

// 错误处理
ag.AddMiddleware(errorhandler.NewErrorHandler())
```

### 5. Memory (智能记忆)

```go
// 添加记忆
err := memoryStore.AddMemory(ctx, message)

// 搜索相似记忆
results, _ := memoryStore.SearchMemory(ctx, query)

// 获取具体记忆
mem, _ := memoryStore.GetMemoryByID(ctx, memoryID)

// 更新记忆
err := memoryStore.UpdateMemory(ctx, id, newContent, metadata)

// 删除记忆
err := memoryStore.DeleteMemory(ctx, id)
```

### 6. Tool (工具系统)

```go
// 注册工具
toolReg := tool.NewRegistry()

toolReg.RegisterTool("query_order", tool.NewTool(
    "query_order",
    "查询订单信息",
    func(ctx context.Context, args map[string]any) (any, error) {
        orderID := args["order_id"].(string)
        // 业务逻辑...
        return result, nil
    },
))

// Agent使用工具（自动调用）
// 在SystemPrompt中描述可用工具
// Agent会自动判断何时调用
```

### 7. Prompt (提示词管理)

```go
// 创建Prompt Manager
pm := prompt.NewManager()

// 注册提示词模板
pm.Register("customer_service", systemPrompt)
pm.Register("operation", operationPrompt)

// 检索提示词
prompt := pm.Get("customer_service")

// 列出所有提示词
all := pm.List()
```

### 8. Vector (向量存储)

```go
// 保存向量
err := vectorStore.SaveEmbedding(ctx, id, content, embedding, metadata)

// 相似性搜索
results, _ := vectorStore.SearchSimilar(ctx, query, 5)

// 删除向量
err := vectorStore.DeleteEmbedding(ctx, id)
```

---

## 📈 监控和指标

### 内置指标

```
总请求数:        12500
成功请求:        12350 (98.8%)
失败请求:        150 (1.2%)
平均响应时间:    1250ms
当前活跃Sessions: 156
总客户数:        1500
总订单数:        8945
总销售额:        ¥2,500,000
```

### 可观测性

- **Prometheus**: 指标收集
- **Grafana**: 实时监控面板
- **Elasticsearch**: 日志存储
- **Kibana**: 日志查询

### 访问地址

| 服务 | 地址 |
|------|------|
| 应用 | http://localhost:8080 |
| Grafana | http://localhost:3000 |
| Prometheus | http://localhost:9090 |
| Kibana | http://localhost:5601 |
| Elasticsearch | http://localhost:9200 |

---

## 🔐 生产部署检查清单

- [ ] 配置API密钥和数据库密码
- [ ] 启用TLS/SSL加密
- [ ] 配置速率限制
- [ ] 设置监控告警
- [ ] 配置自动备份
- [ ] 部署日志聚合
- [ ] 配置负载均衡
- [ ] 性能测试和优化
- [ ] 安全审计
- [ ] 灾难恢复计划

---

## 📚 关键概念总结

### Session vs Context

| 维度 | Session | Context |
|------|---------|---------|
| **作用** | 用户会话管理 | 对话历史记录 |
| **范围** | 用户级别 | Agent级别 |
| **生命周期** | 用户从登录到登出 | Agent运行期间 |
| **数量** | 10000+ | 与Session数量相同 |
| **关系** | 1 Session : 1+ Agent | 1 Agent : 1 Context |

### 多轮对话工作原理

```
Session内的多轮对话流程：

Round 1: User Input → Session.Run()
         ↓
         Agent处理 + Context记录 (msg1, msg2)
         ↓
         Response1

Round 2: User Input → Session.Run()
         ↓
         Agent处理 (可访问前1轮的消息)
         ↓
         Context记录 (msg1, msg2, msg3, msg4)
         ↓
         Response2 (更准确，有完整上下文)

Round 3: User Input → Session.Run()
         ↓
         Agent处理 (可访问前2轮的所有消息)
         ↓
         Context记录 (msg1-msg6)
         ↓
         Response3 (最精准，有最完整的上下文)
```

---

## 🎓 学习路径

1. **基础**: 运行单个演示场景
2. **进阶**: 修改Agent的SystemPrompt
3. **高级**: 添加新的Agent或工具
4. **专家**: 实现自定义Middleware和Memory存储

---

## 📞 支持和反馈

- 提交Issue: GitHub Issues
- 讨论: GitHub Discussions
- 文档: [ai-allin Documentation](https://github.com/sweetpotato0/ai-allin)

---

## 📄 许可证

MIT License

---

**祝你使用愉快！🚀**
