# Graph、Runner 中的 Session 历史需求分析

## 快速总结

| 组件 | 需要Session历史? | 何时需要 | 如何处理 |
|------|-------------|--------|--------|
| **Agent** | ✅ **必需** | 多轮对话、上下文理解 | Agent内部自动维护Context |
| **Session** | ✅ **必需** | Session封装Agent，代理其历史 | session.GetMessages()获取 |
| **Graph节点** | ⚠️ **有条件** | 节点内进行多轮对话时 | 在节点中创建/维护Session |
| **Graph State** | ❌ **不需要** | State仅用于临时节点间传递 | 不应存储永久消息 |
| **Runner** | ❌ **不需要** | Runner只执行任务，不维护历史 | 由被执行的Agent/Graph负责 |

---

## 1. Agent 中的 Session 历史（✅ 必需）

### 原因

Agent执行LLM调用和工具调用，需要完整的对话历史来：
- 理解上下文（用户说"第二个"时需要知道前面提到的内容）
- 维持连贯性（记住已做的操作）
- 支持多轮对话（每一轮都需要前面的消息）

### 代码示例

```go
ag := agent.New(
    agent.WithSystemPrompt("你是客服..."),
    agent.WithMemory(memoryStore),
)

// Round 1
response1, _ := ag.Run(ctx, "我要查询订单")
// Agent Context: [用户消息1, 助手回复1]

// Round 2 - 需要Round 1的历史！
response2, _ := ag.Run(ctx, "订单什么时候到？")
// Agent Context: [用户消息1, 助手回复1, 用户消息2, 助手回复2]
//                 ↑                   ↑
//                 Round 1必须保留！

// 获取完整历史
messages := ag.GetMessages()  // 包含所有轮次
```

---

## 2. Session 中的 Session 历史（✅ 必需）

### 关键点

**Session 是 Agent 的包装器**，它：
- 不创建自己的消息历史
- 代理内部 Agent 的历史
- 提供会话隔离和状态管理

### 代码示例

```go
// Session包装Agent
sess, _ := sessionManager.Create("session_123", agent)

// Session.Run() 最终调用 agent.Run()
sess.Run(ctx, "用户消息1")  // Agent记录消息

// 获取历史时，Session返回Agent的历史
messages := sess.GetMessages()  // = agent.GetMessages()
```

### 多个 Agent 共享同一个 Session

当多个 Agent 需要围绕同一用户线程协作时，可使用 `session.Conversation` / `session.Orchestrator` 封装历史搬运逻辑：

```go
orchestrator := session.NewOrchestrator()

// researcher 与 solver 交替执行，历史由 orchestrator 维护
orchestrator.Run(ctx, "case-42", researcherAgent, "收集关键信息")
orchestrator.Run(ctx, "case-42", solverAgent, "根据历史生成解决方案")
```

Orchestrator 在内部会：
1. 将当前会话历史灌入 Agent；
2. 调用 `agent.Run`；
3. 把新的消息历史写回会话。

开发者不再需要显式调用 `GetMessages` / `AddMessage`。

### 消息流向

```
User Input → sess.Run(ctx, input)
                    ↓
            agent.Run(ctx, input)
                    ↓
            LLM API Call
                    ↓
            Agent.Context += [User Msg, Assistant Response]
                    ↓
            Return response
                    ↓
sess.GetMessages() ← Agent.Context [保留所有历史]
```

---

## 3. Graph 中的 Session 历史（⚠️ 有条件需要）

### 关键区分

**Graph.State ≠ 消息历史**

- **Graph.State**: 临时的、节点间传递的上下文数据
- **消息历史**: 永久的、由Session/Agent维护

### 何时需要历史

#### 情景1: Graph节点内进行多轮对话

✅ **需要创建 Session，Session保留历史**

```go
builder.AddNode("multi_turn", graph.NodeTypeLLM, 
func(ctx context.Context, state graph.State) (graph.State, error) {
    
    // 【错误❌】为每轮创建新Agent（丢失历史）
    for each round {
        ag := createAgent()  // ❌ 新Agent，丢失历史
        ag.Run(ctx, userMsg)
    }
    
    // 【正确✅】创建一个Session，在其中进行多轮对话
    sess := state["session"].(session.Session)
    
    r1, _ := sess.Run(ctx, "第1轮")  // Session记录
    r2, _ := sess.Run(ctx, "第2轮")  // 保留第1轮历史
    r3, _ := sess.Run(ctx, "第3轮")  // 保留第1,2轮历史
    
    // 获取完整历史（包含所有轮次）
    messages := sess.GetMessages()
    state["history"] = messages
    
    return state, nil
})
```

#### 情景2: Graph节点间不需要消息历史

❌ **不需要在State中存储消息**

```go
// ❌ 错误：在State中存储消息（State只是临时容器）
state["messages"] = []string{...}  // 不可靠

// ✅ 正确：消息由Session维护，State只传递Session引用
state["session"] = sess  // 引用，不拷贝
```

### Graph执行流程

```
Graph.Execute(ctx, initialState)
    ↓
Node1: init_session
    └─ 创建Agent → 创建Session
    └─ state["session"] = sess  ← 保存Session引用
    ↓
Node2: multi_turn_dialog
    └─ 获取Session: sess = state["session"]
    └─ Round1: sess.Run() → Agent.Context += [R1]
    └─ Round2: sess.Run() → Agent.Context += [R2]  ← 保留R1
    └─ Round3: sess.Run() → Agent.Context += [R3]  ← 保留R1,R2
    └─ messages = sess.GetMessages()  ← 获取历史
    ↓
Node3: route_by_type
    └─ 根据state中的数据做决策（不涉及消息历史）
    ↓
Node4: specialized_handling
    └─ 继续使用same session进行更多对话
    ↓
Node5: cleanup
    └─ sess.Close()  ← 关闭Session

FinalState = {
    "final_response": "...",
    "conversation_history": [消息1, 消息2, ...],  ← 来自Session
    ...
}
```

---

## 4. Runner 中的 Session 历史（❌ 不需要）

### 原因

Runner 只负责**任务执行**，不维护消息历史

- **ParallelRunner**: 并行执行多个Agent/Graph
- **SequentialRunner**: 按顺序执行多个Agent/Graph
- **ConditionalRunner**: 条件执行多个Agent/Graph

### 代码示例

```go
// Runner执行Agent（历史由Agent维护）
runner := runner.New(10)
output, err := runner.Run(ctx, agent, input)
//                                           ↑
//                                    Agent维护历史

// Runner执行Graph（历史由Graph内的Session维护）
finalState, err := runner.RunGraph(ctx, graph, initialState)
//                                                    ↑
//                                        Session维护历史

// Runner不需要管理历史
// Runner只关心：执行开始→执行结束→返回结果
```

### 并行执行 + Session 历史

```go
// 为5个客户并行执行工作流
tasks := []*runner.Task{
    {ID: "task1", Agent: agent1, Input: "CUST001"},
    {ID: "task2", Agent: agent2, Input: "CUST002"},
    {ID: "task3", Agent: agent3, Input: "CUST003"},
    {ID: "task4", Agent: agent4, Input: "CUST004"},
    {ID: "task5", Agent: agent5, Input: "CUST005"},
}

// ParallelRunner并行执行
parallelRunner := runner.NewParallelRunner(10)
results := parallelRunner.RunParallel(ctx, tasks)

// 每个任务有自己独立的Agent和Session
//
// Task1: Agent1.Run() → Agent1.Context = [msg1, msg2, ...] ✓
// Task2: Agent2.Run() → Agent2.Context = [msg1, msg2, ...] ✓
// Task3: Agent3.Run() → Agent3.Context = [msg1, msg2, ...] ✓
// Task4: Agent4.Run() → Agent4.Context = [msg1, msg2, ...] ✓
// Task5: Agent5.Run() → Agent5.Context = [msg1, msg2, ...] ✓
//
// Runner只负责调度，不涉及历史管理
```

---

## 5. 实际应用场景对比

### 场景A: 简单单轮查询

```
User → "我的订单在哪？"
            ↓
        ┌─────────────────┐
        │ Agent (单轮)    │
        │ 不需要历史      │
        └─────────────────┘
            ↓
Answer → "您的订单在运输中..."
```

**是否需要Session历史**: ❌ 不需要

**代码**:
```go
ag := agentFactory.CreateAgent()
response, _ := ag.Run(ctx, "我的订单在哪？")
// 完全是原子操作，不需要保留历史
```

---

### 场景B: 多轮对话

```
User → "我要退货"
    ↓
Agent → "您要求全额还是部分退款？"  ← 需要记住上一轮
    ↓
User → "全额"
    ↓
Agent → "已处理，3-5天到账"  ← 需要记住前两轮
```

**是否需要Session历史**: ✅ 需要

**代码**:
```go
// 为这个客户创建Session（保留历史）
sess, _ := sessionManager.Create("session_001", agent)

r1, _ := sess.Run(ctx, "我要退货")
// Context: [用户消息1, 助手回复1]

r2, _ := sess.Run(ctx, "全额")
// Context: [用户消息1, 助手回复1, 用户消息2, 助手回复2]
//          └──────────────────────┘
//          需要这一轮的历史！

r3, _ := sess.Run(ctx, "谢谢")
// Context: [...所有之前的消息..., 用户消息3, 助手回复3]
```

---

### 场景C: Graph工作流（含多轮对话）

```
Request
   ↓
┌─────────────────────────┐
│ Graph Workflow          │
├─────────────────────────┤
│ Node1: 初始化Session    │
│   └─ Create Session     │
│   └─ state["session"]   │
├─────────────────────────┤
│ Node2: 问题分类（R1）    │
│   └─ sess.Run()         │
│   └─ Context: [R1]      │
├─────────────────────────┤
│ Node3: 信息收集（R2）    │
│   └─ sess.Run()         │
│   └─ Context: [R1, R2]  ← 需要R1
├─────────────────────────┤
│ Node4: 路由判断        │
│   └─ 分析分类结果      │
├─────────────────────────┤
│ Node5a: 处理（R3,R4）   │
│   └─ sess.Run() × 2     │
│   └─ Context: [R1..R4]  ← 需要R1,R2,R3
├─────────────────────────┤
│ Node6: 清理Session      │
│   └─ sess.Close()       │
│   └─ 保存完整历史       │
└─────────────────────────┘
   ↓
Result + History
```

**是否需要Session历史**: ✅ 需要（在Graph内部）

**代码**:
```go
// Node1: 初始化
builder.AddNode("init", graph.NodeTypeStart, func(ctx, state) (graph.State, error) {
    ag := createAgent()
    sess, _ := sessionManager.Create("sess_001", ag)
    state["session"] = sess
    return state, nil
})

// Node2: Round1
builder.AddNode("classify", graph.NodeTypeLLM, func(ctx, state) (graph.State, error) {
    sess := state["session"].(session.Session)
    r1, _ := sess.Run(ctx, "分类问题")
    return state, nil
})

// Node3: Round2 - 需要历史
builder.AddNode("collect_info", graph.NodeTypeLLM, func(ctx, state) (graph.State, error) {
    sess := state["session"].(session.Session)
    r2, _ := sess.Run(ctx, "收集信息")
    // sess.GetMessages() 包含R1!
    return state, nil
})

// Node5: Round3,4 - 需要R1,R2
builder.AddNode("process", graph.NodeTypeLLM, func(ctx, state) (graph.State, error) {
    sess := state["session"].(session.Session)
    r3, _ := sess.Run(ctx, "...")
    r4, _ := sess.Run(ctx, "...")
    // sess.GetMessages() 包含R1,R2,R3!
    return state, nil
})
```

---

### 场景D: ParallelRunner执行5个Graph工作流

```
┌─────────────────┬─────────────────┬─────────────────┬─────────────────┬─────────────────┐
│   Task1/Graph1  │   Task2/Graph2  │   Task3/Graph3  │   Task4/Graph4  │   Task5/Graph5  │
│   CUST001       │   CUST002       │   CUST003       │   CUST004       │   CUST005       │
├─────────────────┼─────────────────┼─────────────────┼─────────────────┼─────────────────┤
│ sess1 (R1,R2,R3)│ sess2 (R1,R2,R3)│ sess3 (R1,R2,R3)│ sess4 (R1,R2,R3)│ sess5 (R1,R2,R3)│
│    in Graph     │    in Graph     │    in Graph     │    in Graph     │    in Graph     │
└─────────────────┴─────────────────┴─────────────────┴─────────────────┴─────────────────┘
                                     ↑
                          ParallelRunner (并行调度)
                          不涉及消息历史管理
```

**是否需要Runner历史**: ❌ 不需要（但Graph内的Session需要）

**代码**:
```go
// 构建5个独立的Graph
graphs := []struct {
    id    string
    graph *graph.Graph
}{
    {"CUST001", buildCustomerServiceGraph()},
    {"CUST002", buildCustomerServiceGraph()},
    // ...
}

// ParallelRunner并行执行
// 每个Graph内部有自己的Session，维护自己的历史
// Runner只负责并行调度，不管历史
```

---

## 6. 设计原则总结

### ✅ DO: 正确的做法

1. **Agent多轮对话**
   ```go
   ag.Run(ctx, "Round1")  // Agent.Context += Round1
   ag.Run(ctx, "Round2")  // Agent.Context += Round1 + Round2
   ```

2. **Session多轮对话**
   ```go
   sess.Run(ctx, "Round1")  // 委托给Agent，保留历史
   sess.Run(ctx, "Round2")  // 委托给Agent，保留历史
   ```

3. **Graph内部使用Session**
   ```go
   builder.AddNode("node1", ..., func(ctx, state) {
       sess := state["session"]  // 从State获取Session引用
       sess.Run(ctx, "...")      // 调用Session方法
       messages := sess.GetMessages()  // 获取历史
   })
   ```

4. **Runner执行Agent/Graph**
   ```go
   runner.Run(ctx, agent, input)       // 历史由Agent维护
   runner.RunGraph(ctx, graph, state)  // 历史由Graph内Session维护
   ```

### ❌ DON'T: 常见错误

1. **为每轮创建新Agent**
   ```go
   ❌ for each round {
        ag := createAgent()  // 新Agent，历史丢失
        ag.Run(ctx, input)
      }
   ```

2. **在Graph.State中存储消息**
   ```go
   ❌ state["all_messages"] = [...]  // State是临时的，不可靠
   ```

3. **Graph中不使用Session**
   ```go
   ❌ ag.Run(ctx, "Round1")  // 每次Run都是独立的，历史丢失
   ```

4. **Runner级别管理消息**
   ```go
   ❌ runner.Run() 不需要关心历史  // 由Agent/Graph负责
   ```

---

## 7. 检查清单

| 问题 | 答案 | 处理方式 |
|------|------|--------|
| 需要多轮对话吗? | ✅ 是 | 使用Session |
| 需要复杂工作流吗? | ✅ 是 | 使用Graph |
| Graph内需要多轮? | ✅ 是 | 在Node内创建Session |
| 需要并发执行吗? | ✅ 是 | 使用ParallelRunner |
| 消息需要在State传递? | ❌ 否 | 由Session维护 |
| Runner需要管理历史? | ❌ 否 | 由Agent/Graph负责 |
