# Session 重构方案

## 当前问题分析

### 1. 代码重复
- `Session` 和 `Conversation` 有很多重复的字段和方法
- `Manager` 和 `ConversationManager` 功能相似但分离
- 都使用相同的 `State` 类型，但定义重复

### 2. 设计不一致
- `Session` 是接口，`Conversation` 是具体类型，未实现接口
- `Manager` 管理 `Session`，`ConversationManager` 管理 `Conversation`
- `Orchestrator` 使用 `ConversationManager`，但也可以使用 `Session`

### 3. 概念混乱
- `Session` 和 `Conversation` 概念重叠，但使用场景不同
- 用户需要理解两个不同的概念和 API

## 重构目标

1. **统一接口**：让 `Conversation` 实现 `Session` 接口
2. **统一管理**：统一 `Manager` 和 `ConversationManager`
3. **简化设计**：减少重复代码，提高可维护性
4. **向后兼容**：保留现有 API，确保现有代码可以正常工作

## 重构方案

### 方案 1：统一 Session 接口（推荐）

#### 步骤 1：提取公共基础结构

创建 `types.go`：
```go
// State 和 Session 接口定义
// SessionBase 公共基础结构
```

#### 步骤 2：重构 Session 实现

将 `session.go` 中的 `session` 重命名为 `SingleAgentSession`：
- 单一 agent 绑定
- 直接使用 agent 的消息历史

#### 步骤 3：让 Conversation 实现 Session 接口

修改 `conversation.go`：
- 重命名为 `MultiAgentSession`
- 实现 `Session` 接口
- 支持可选默认 agent（用于 `Session.Run()`）

#### 步骤 4：统一 Manager

合并 `Manager` 和 `ConversationManager`：
- 统一管理所有 `Session` 实现
- 提供 `CreateSingleAgent` 和 `CreateMultiAgent` 方法
- 保留 `Create` 方法作为向后兼容

#### 步骤 5：简化 Orchestrator

修改 `orchestrator.go`：
- 使用统一的 `Manager` 而不是 `ConversationManager`
- 使用 `Session` 接口而不是具体类型
- 简化代码逻辑

### 方案 2：保持分离但统一接口（保守）

如果不想大幅重构，可以：
1. 让 `Conversation` 实现 `Session` 接口
2. 保留 `Manager` 和 `ConversationManager` 分离
3. 让 `Orchestrator` 同时支持两种 Manager

## 文件结构

重构后的文件结构：
```
session/
├── types.go              # State, Session 接口, SessionBase
├── single_agent.go       # SingleAgentSession 实现
├── multi_agent.go        # MultiAgentSession 实现（原 Conversation）
├── manager.go            # 统一的 Manager
├── orchestrator.go       # Orchestrator（使用统一 Manager）
└── store/                # 存储后端
    └── redis.go
```

## 重构步骤

### 阶段 1：准备（不破坏现有代码）
1. 创建 `types.go` 提取公共定义
2. 创建新的实现文件（`single_agent.go`, `multi_agent.go`）
3. 保持旧文件不变

### 阶段 2：迁移（逐步替换）
1. 更新 `Manager` 使用新的实现
2. 更新 `Orchestrator` 使用统一 `Manager`
3. 更新测试和示例代码

### 阶段 3：清理（删除旧代码）
1. 删除旧的 `session.go` 和 `conversation.go`
2. 重命名新文件为标准名称
3. 更新文档

## API 兼容性

### 向后兼容的 API

```go
// 单一 agent session（向后兼容）
sess, _ := session.NewManager().Create("id", agent)
sess.Run(ctx, "input")

// 多 agent session（新 API）
multiSess, _ := session.NewManager().CreateMultiAgent("id")
multiSess.(*session.MultiAgentSession).RunWithAgent(ctx, agent1, "input")
multiSess.(*session.MultiAgentSession).RunWithAgent(ctx, agent2, "input")

// Orchestrator（向后兼容）
orchestrator := session.NewOrchestrator()
orchestrator.RegisterAgent("agent1", agent1)
orchestrator.Run(ctx, "session-id", "agent1", "input")
```

## 优势

1. **统一接口**：所有 session 都实现 `Session` 接口
2. **代码复用**：提取公共代码到 `SessionBase`
3. **简化管理**：统一的 `Manager` 管理所有 session
4. **向后兼容**：保留现有 API
5. **易于扩展**：未来可以添加更多 `Session` 实现

## 风险

1. **破坏性变更**：如果现有代码直接使用 `Conversation` 类型
2. **测试覆盖**：需要更新所有测试
3. **文档更新**：需要更新所有文档和示例

## 建议

建议采用**方案 1**（统一 Session 接口），因为：
1. 设计更清晰
2. 代码更易维护
3. 向后兼容性更好
4. 未来扩展更容易

