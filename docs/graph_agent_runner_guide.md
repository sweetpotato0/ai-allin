# Graph、Agent、Runner、Session 综合使用指南

## 核心概念关系图

```
┌─────────────────────────────────────────────────────────────┐
│                    Customer Request                          │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
        ┌────────────────────────┐
        │   Runner (任务执行器)   │
        │  - ParallelRunner      │
        │  - SequentialRunner    │
        │  - ConditionalRunner   │
        └──────────┬─────────────┘
                   │
        ┌──────────▼──────────┐
        │  Graph (工作流程)    │
        │  - 定义节点和路由   │
        │  - 条件分支        │
        │  - 状态转移        │
        └──────────┬──────────┘
                   │
      ┌────────────▼────────────┐
      │   Node (图节点)          │
      │  ┌──────────────────┐   │
      │  │ Agent Node (LLM) │   │
      │  │ 创建/复用Session │   │
      │  └──────────────────┘   │
      │  ┌──────────────────┐   │
      │  │ Tool Node        │   │
      │  │ 执行业务操作     │   │
      │  └──────────────────┘   │
      │  ┌──────────────────┐   │
      │  │ Condition Node   │   │
      │  │ 决策分支         │   │
      │  └──────────────────┘   │
      └──────────┬───────────────┘
                 │
      ┌──────────▼──────────┐
      │  Session (会话)     │
      │  - 保存消息历史     │
      │  - Agent Context    │
      │  - 多轮对话支持     │
      └─────────────────────┘
```

## 1. Agent 的核心作用

### Agent 职责
- **执行任务**: 接收输入，调用LLM处理，返回输出
- **维护Context**: 保存消息历史供多轮对话
- **工具调用**: 通过工具进行业务操作

### Agent 关键特性
```go
// Agent拥有内部的Context（消息历史）
ag := agent.New(
    agent.WithName("customer_service"),
    agent.WithSystemPrompt("你是客服..."),
    agent.WithMemory(memoryStore),  // 智能记忆
    agent.WithTools(true),           // 启用工具
)

// 每次Run()都会记录消息
response, err := ag.Run(ctx, "客户问题")

// 获取消息历史
messages := ag.GetMessages()  // 包含所有对话轮次
```

## 2. Session 的核心作用

### Session 职责
- **会话隔离**: 为每个客户/对话创建独立命名空间
- **Context委托**: 将消息历史委托给内部的Agent
- **状态管理**: 跟踪会话生命周期

### Session 关键特性
```go
// Session包装Agent，提供会话隔离
sess, err := sessionManager.Create("session_001", agent)

// Session.Run()最终调用Agent.Run()
response, err := sess.Run(ctx, "用户输入")

// 获取消息历史（来自Agent）
messages := sess.GetMessages()  // = agent.GetMessages()

// Session状态管理
state := sess.GetState()  // "active", "inactive", "closed"
```

### 何时需要Session历史

**场景1: 多轮对话**
```
Round 1: 用户 -> "我要查询订单"  -> Agent -> Agent.Context[msg1,msg2]
Round 2: 用户 -> "订单号是什么？" -> Agent -> Agent.Context[msg1,msg2,msg3,msg4]
                                    ↑
                                 需要历史！
```

**场景2: 上下文理解**
```
Assistant: "您可以选择这3个选项"
User: "我要第二个"  <- 需要理解前面提到的3个选项
          ↑
      需要历史！
```

## 3. Graph 的核心作用

### Graph 职责
- **定义工作流**: 多个步骤的执行顺序
- **条件分支**: 根据结果决定下一步
- **状态流转**: 通过State维持上下文

### Graph 关键特性
```go
// Graph由多个Node组成，通过State传递数据
type Node struct {
    Name      string                    // 节点名称
    Type      NodeType                  // "start", "end", "llm", "tool", "condition", "custom"
    Execute   func(ctx, State) State    // 执行函数
    Condition func(ctx, State) string   // 条件函数（返回下一步）
    Next      string                    // 默认下一节点
    NextMap   map[string]string         // 条件 -> 下一节点
}

// Graph执行流程
state := graph.Execute(ctx, initialState)
//
// Node1(初始化) -> State1
//         ↓
// Node2(检查条件)
//    ├─ "优先级高" -> Node3(加急处理)
//    └─ "普通"     -> Node4(标准处理)
//         ↓
// Node5(返回结果)
```

### State 的重要性
```go
// State在节点间传递上下文数据
type State map[string]any

// 节点可以访问和修改State
Execute: func(ctx context.Context, state State) (State, error) {
    customerID := state["customer_id"].(string)
    orders := state["orders"].([]Order)

    // 处理逻辑
    result := processCustomer(orders)

    // 更新State传递给下一节点
    state["result"] = result
    state["next_action"] = "send_email"
    return state, nil
}
```

## 4. Runner 的核心作用

### Runner 职责
- **任务执行**: 执行Agent或Graph
- **并发管理**: 控制并发数量和顺序
- **结果收集**: 聚合执行结果

### Runner 类型
```go
// 1. Runner: 基础执行器
runner := runner.New(maxConcurrency)
output, err := runner.Run(ctx, agent, input)
graphState, err := runner.RunGraph(ctx, graph, initialState)

// 2. ParallelRunner: 并行执行多个任务
parallelRunner := runner.NewParallelRunner(10)  // max 10并发
results := parallelRunner.RunParallel(ctx, tasks)
// tasks = []*runner.Task{
//     {ID: "task1", Agent: agent1, Input: "input1"},
//     {ID: "task2", Agent: agent2, Input: "input2"},
// }

// 3. SequentialRunner: 顺序执行（前一个输出作为下一个输入）
seqRunner := runner.NewSequentialRunner()
result, err := seqRunner.RunSequential(ctx, tasks)

// 4. ConditionalRunner: 条件执行
condRunner := runner.NewConditionalRunner()
results, err := condRunner.RunConditional(ctx, tasks)
```

## 5. Graph + Agent 集成方式

### 方式1: Graph节点中直接使用Agent

```go
// 构建图
builder := graph.NewBuilder()

// 节点1: 初始化Agent Session
builder.AddNode("init_session", graph.NodeTypeStart, func(ctx context.Context, state graph.State) (graph.State, error) {
    // 创建Session
    ag := agentFactory.CreateCustomerServiceAgent("cs_agent")
    sess, _ := sessionManager.Create("session_123", ag)

    // 保存到State（传递给后续节点）
    state["session"] = sess
    state["customer_id"] = "CUST001"
    return state, nil
})

// 节点2: 调用Agent处理（保留Session历史）
builder.AddNode("agent_process", graph.NodeTypeLLM, func(ctx context.Context, state graph.State) (graph.State, error) {
    sess := state["session"].(session.Session)

    // 第1轮对话
    response1, _ := sess.Run(ctx, "查询我的订单")

    // 第2轮对话（Session保留第1轮的消息）
    response2, _ := sess.Run(ctx, "订单什么时候到？")

    // 获取完整的对话历史
    messages := sess.GetMessages()  // 包含4条消息（用户+助手 × 2）

    state["responses"] = []string{response1, response2}
    state["full_history"] = messages
    return state, nil
})

// 节点3: 条件判断
builder.AddConditionNode("check_priority", func(ctx context.Context, state graph.State) (string, error) {
    customer := state["customer"].(Customer)
    if customer.VIPLevel == "gold" {
        return "high_priority", nil
    }
    return "normal_priority", nil
}, map[string]string{
    "high_priority": "vip_processing",
    "normal_priority": "standard_processing",
})

// 节点4a: VIP处理
builder.AddNode("vip_processing", graph.NodeTypeCustom, func(ctx context.Context, state graph.State) (graph.State, error) {
    // VIP专用流程
    return state, nil
})

// 节点4b: 标准处理
builder.AddNode("standard_processing", graph.NodeTypeCustom, func(ctx context.Context, state graph.State) (graph.State, error) {
    // 标准流程
    return state, nil
})

// 节点5: 清理和结束
builder.AddNode("cleanup", graph.NodeTypeEnd, func(ctx context.Context, state graph.State) (graph.State, error) {
    sess := state["session"].(session.Session)
    sess.Close()
    return state, nil
})

// 连接节点
builder.AddEdge("init_session", "agent_process")
builder.AddEdge("agent_process", "check_priority")
builder.AddEdge("vip_processing", "cleanup")
builder.AddEdge("standard_processing", "cleanup")
builder.SetStart("init_session")
builder.SetEnd("cleanup")

// 执行Graph
graph := builder.Build()
finalState, err := graph.Execute(ctx, graph.State{})
```

## 6. Runner + Graph 集成方式

### 方式1: Runner执行Graph

```go
// 创建Runner
runner := runner.New(10)

// 创建Graph（包含Agent）
graph := buildCustomerServiceGraph()

// Runner执行Graph
initialState := graph.State{
    "customer_id": "CUST001",
    "inquiry": "我的订单在哪？",
}

finalState, err := runner.RunGraph(ctx, graph, initialState)

// 获取结果
result := finalState["final_response"]
history := finalState["message_history"]
```

### 方式2: Runner并行执行多个Graph

```go
// 为每个客户创建一个Graph
graphs := []struct {
    id    string
    graph *graph.Graph
}{
    {"task1", buildCustomerServiceGraph()},
    {"task2", buildCustomerServiceGraph()},
    {"task3", buildCustomerServiceGraph()},
}

// 使用Runner并行执行
runner := runner.New(10)

// Graph执行需要包装成Task吗？需要检查！
```

## 7. Session历史需求总结

| 组件 | 是否需要历史 | 原因 | 如何处理 |
|------|-----------|------|--------|
| **Agent** | ✅ 必需 | 多轮对话、上下文理解、工具调用需要历史 | Agent内部自动保存 Context |
| **Session** | ✅ 必需 | Session是Agent的包装，代理Agent的历史 | Session.GetMessages() 返回Agent历史 |
| **Graph** | ⚠️ 有时需要 | State只在节点间临时传递；长期对话需要在Node内部使用Session | 在Graph节点中创建/维护Session |
| **Runner** | ❌ 不需要 | Runner只负责执行任务；历史由被执行的Agent/Graph维护 | 无需管理；由子任务管理 |

## 8. 完整示例：图+Agent+Session多轮对话

```go
// 创建复杂的客服工作流
func buildComplexCustomerServiceFlow(sessionMgr *session.Manager, agentFactory *AgentFactory) *graph.Graph {
    builder := graph.NewBuilder()

    // 【阶段1】初始化
    builder.AddNode("init", graph.NodeTypeStart, func(ctx context.Context, state graph.State) (graph.State, error) {
        // 创建Session（保留多轮对话历史）
        ag := agentFactory.CreateCustomerServiceAgent("cs_agent")
        sess, _ := sessionManager.Create(fmt.Sprintf("session_%d", time.Now().Unix()), ag)

        state["session"] = sess
        state["customer_id"] = state["customer_id"]  // 来自外部
        return state, nil
    })

    // 【阶段2】第1轮对话 - 问题分类
    builder.AddNode("round1_classify", graph.NodeTypeLLM, func(ctx context.Context, state graph.State) (graph.State, error) {
        sess := state["session"].(session.Session)

        // 第1轮：用户询问
        userMsg := state["user_inquiry"].(string)  // 外部输入
        response1, _ := sess.Run(ctx, userMsg)

        state["round1_response"] = response1
        state["message_count"] = 2  // 用户 + 助手
        return state, nil
    })

    // 【阶段3】第2轮对话 - 收集更多信息
    builder.AddNode("round2_collect_info", graph.NodeTypeLLM, func(ctx context.Context, state graph.State) (graph.State, error) {
        sess := state["session"].(session.Session)

        // 第2轮：基于第1轮结果继续（历史自动保留！）
        followup := "能否提供您的订单号或邮箱地址？"
        response2, _ := sess.Run(ctx, followup)

        messages := sess.GetMessages()  // 现在包含4条消息
        state["round2_response"] = response2
        state["message_count"] = len(messages)
        return state, nil
    })

    // 【阶段4】条件判断 - 路由到不同处理流程
    builder.AddConditionNode("route_by_type", func(ctx context.Context, state graph.State) (string, error) {
        category := extractCategory(state["round1_response"].(string))
        if category == "refund" {
            return "refund_process", nil
        } else if category == "track" {
            return "track_process", nil
        }
        return "general_support", nil
    }, map[string]string{
        "refund_process": "process_refund",
        "track_process": "track_order",
        "general_support": "provide_solution",
    })

    // 【阶段5a】退款处理 - 继续多轮对话
    builder.AddNode("process_refund", graph.NodeTypeLLM, func(ctx context.Context, state graph.State) (graph.State, error) {
        sess := state["session"].(session.Session)

        // 第3轮：退款特定对话（前2轮对话历史自动可用）
        response3, _ := sess.Run(ctx, "您要求的是全额退款还是部分退款？")

        // 第4轮
        response4, _ := sess.Run(ctx, "退款已处理，预计3-5个工作日到账")

        messages := sess.GetMessages()  // 包含8条消息（4轮）
        state["final_response"] = response4
        state["conversation_history"] = messages
        return state, nil
    })

    // 【阶段5b】查询订单 - 继续多轮对话
    builder.AddNode("track_order", graph.NodeTypeLLM, func(ctx context.Context, state graph.State) (graph.State, error) {
        sess := state["session"].(session.Session)

        // 第3轮：订单追踪
        response3, _ := sess.Run(ctx, "您的订单已在运输中，预计明天送达")

        // 第4轮：追加信息
        response4, _ := sess.Run(ctx, "追踪链接已发送到您的邮箱")

        messages := sess.GetMessages()
        state["final_response"] = response4
        state["conversation_history"] = messages
        return state, nil
    })

    // 【阶段5c】通用支持
    builder.AddNode("provide_solution", graph.NodeTypeLLM, func(ctx context.Context, state graph.State) (graph.State, error) {
        sess := state["session"].(session.Session)

        // 第3轮
        response3, _ := sess.Run(ctx, "如需进一步帮助，请告诉我")

        messages := sess.GetMessages()
        state["final_response"] = response3
        state["conversation_history"] = messages
        return state, nil
    })

    // 【阶段6】清理
    builder.AddNode("end", graph.NodeTypeEnd, func(ctx context.Context, state graph.State) (graph.State, error) {
        sess := state["session"].(session.Session)
        sess.Close()
        return state, nil
    })

    // 连接节点
    builder.AddEdge("init", "round1_classify")
    builder.AddEdge("round1_classify", "round2_collect_info")
    builder.AddEdge("round2_collect_info", "route_by_type")
    builder.AddEdge("process_refund", "end")
    builder.AddEdge("track_order", "end")
    builder.AddEdge("provide_solution", "end")

    builder.SetStart("init")
    builder.SetEnd("end")

    return builder.Build()
}

// 使用Graph
func handleComplexCustomerQuery() {
    ctx := context.Background()

    graph := buildComplexCustomerServiceFlow(sessionMgr, agentFactory)

    initialState := graph.State{
        "customer_id": "CUST001",
        "user_inquiry": "我想退货",
    }

    finalState, err := graph.Execute(ctx, initialState)
    if err != nil {
        log.Printf("处理失败: %v", err)
        return
    }

    // 获取完整对话历史（4轮对话）
    history := finalState["conversation_history"].([]*message.Message)
    log.Printf("总对话轮数: %d", len(history)/2)

    // 打印所有消息
    for _, msg := range history {
        log.Printf("[%s] %s", msg.Role, msg.Content)
    }
}
```

## 9. 关键设计模式总结

### ✅ DO: 正确的集成方式

```go
// 1. 为每个客户/会话创建独立的Session + Agent
sessionID := fmt.Sprintf("session_%s_%d", customerID, time.Now().Unix())
agent := agentFactory.CreateCustomerServiceAgent("agent_1")
session, _ := sessionManager.Create(sessionID, agent)

// 2. 在Graph节点内部维护Session生命周期
builder.AddNode("process", graph.NodeTypeLLM, func(ctx, state) (graph.State, error) {
    sess := state["session"].(session.Session)

    // 多轮对话 - 消息自动累积
    r1, _ := sess.Run(ctx, "第1轮")
    r2, _ := sess.Run(ctx, "第2轮")
    r3, _ := sess.Run(ctx, "第3轮")

    // 获取完整历史
    messages := sess.GetMessages()

    state["responses"] = []string{r1, r2, r3}
    state["history"] = messages
    return state, nil
})

// 3. 使用Runner执行Graph/Agent
runner := runner.New(10)
finalState, _ := runner.RunGraph(ctx, graph, initialState)
```

### ❌ DON'T: 常见的错误方式

```go
// ❌ 错误1: 为每轮对话创建新Agent（丢失历史）
for each round {
    agent := agentFactory.CreateAgent()  // ❌ 新Agent，历史丢失！
    response, _ := agent.Run(ctx, userMsg)
}

// ❌ 错误2: State在节点间传递大量消息（State是临时的）
state["all_messages"] = []any{msg1, msg2, ...}  // ❌ 容易丢失

// ❌ 错误3: Graph用于存储永久消息历史
// Graph.State只用于临时传递，不应存储永久数据

// ✅ 正确：使用Session维护永久的消息历史
sess.GetMessages()  // 这是持久的，由Agent维护
```

## 10. 快速决策表

**需要多轮对话吗？**
- ✅ 是 → 使用 **Session** + Agent
- ❌ 否 → 直接使用 Agent

**需要复杂的工作流吗？**
- ✅ 是 → 使用 **Graph** 定义多步流程
- ❌ 否 → 直接使用 Agent

**需要并发执行吗？**
- ✅ 是 → 使用 **Runner**（ParallelRunner）
- ❌ 否 → 直接执行

**Graph内需要多轮对话吗？**
- ✅ 是 → 在Node内创建 Session，Session处理多轮 → 保留历史
- ❌ 否 → 在Node内直接调用 Agent.Run() → 每个Node一轮

