package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sweetpotato0/ai-allin/graph"
	"github.com/sweetpotato0/ai-allin/session"
)

// 【示例】使用Graph+Agent+Session构建复杂的客服工作流

// BuildIntelligentCustomerServiceGraph 构建智能客服工作流
// 
// 工作流特点：
// 1. 使用Graph定义多个处理阶段
// 2. 在Graph节点内维护Session，支持多轮对话
// 3. 根据用户问题类型路由到不同的处理流程
// 4. 完整保留对话历史用于质量检查和学习
func (p *ECommerceServicePlatform) BuildIntelligentCustomerServiceGraph() *graph.Graph {
	builder := graph.NewBuilder()

	// ─────────────────────────────────────────────────
	// 【阶段1】初始化：创建Session和Agent
	// ─────────────────────────────────────────────────
	builder.AddNode("init_session", graph.NodeTypeStart, func(ctx context.Context, state graph.State) (graph.State, error) {
		log.Println("【初始化】创建会话和Agent")

		// 从State获取客户ID（由调用者传入）
		customerID := state["customer_id"].(string)

		// 创建Agent（包含系统提示词和工具）
		agent := p.agentFactory.CreateCustomerServiceAgent("intelligent_cs_agent")

		// 创建Session（为这个客户的本次交互创建独立的会话）
		sessionID := fmt.Sprintf("intelligent_%s_%d", customerID, time.Now().Unix())
		sess, err := p.sessionManager.Create(sessionID, agent)
		if err != nil {
			return state, fmt.Errorf("创建Session失败: %w", err)
		}

		// 配置中间件（日志、限流等）
		p.configureAgentMiddleware(agent, sessionID, customerID)

		// 保存Session到State，后续节点会用到
		state["session"] = sess
		state["session_id"] = sessionID
		state["customer_id"] = customerID

		// 获取客户信息
		customer, _ := p.getCustomer(customerID)
		state["customer"] = customer

		log.Printf("✓ Session创建成功: %s\n", sessionID)
		return state, nil
	})

	// ─────────────────────────────────────────────────
	// 【阶段2】问题分类：第1轮对话
	// 
	// 特点：
	// - 使用Session维持对话历史
	// - Agent会记录这一轮的消息
	// 目的：理解用户的核心问题，分类处理
	// ─────────────────────────────────────────────────
	builder.AddNode("classify_problem", graph.NodeTypeLLM, func(ctx context.Context, state graph.State) (graph.State, error) {
		log.Println("【分类】进行问题分类")

		sess := state["session"].(session.Session)
		customerID := state["customer_id"].(string)

		// 用户的初始问题（由调用者传入）
		userInquiry := state["user_inquiry"].(string)

		// 第1轮对话：Agent分类用户问题
		// Session会自动保存这一轮的[用户消息, 助手回复]
		systemPrompt := fmt.Sprintf(
			"请分析客户 %s 的问题，并分类为以下之一:\n"+
				"1. 订单查询\n"+
				"2. 退货退款\n"+
				"3. 产品咨询\n"+
				"4. 投诉建议\n"+
				"5. 其他\n\n"+
				"客户问题: %s",
			customerID, userInquiry)

		response1, err := sess.Run(ctx, systemPrompt)
		if err != nil {
			return state, fmt.Errorf("分类失败: %w", err)
		}

		// 保存这一轮的回复
		state["classification"] = response1

		// 检查消息历史（现在应该有2条消息）
		messages := sess.GetMessages()
		log.Printf("✓ 问题分类完成，当前消息数: %d (用户消息1 + 助手回复1)\n", len(messages))

		return state, nil
	})

	// ─────────────────────────────────────────────────
	// 【阶段3】信息收集：第2轮对话
	// 
	// 特点：
	// - Session已保留第1轮的消息历史
	// - Agent会在第1轮消息基础上继续对话
	// 目的：收集必要的信息以便精准处理
	// ─────────────────────────────────────────────────
	builder.AddNode("collect_info", graph.NodeTypeLLM, func(ctx context.Context, state graph.State) (graph.State, error) {
		log.Println("【信息收集】进行第2轮对话，收集详细信息")

		sess := state["session"].(session.Session)

		// 第2轮对话：基于分类结果，继续提问
		// Session自动保留第1轮的[用户消息, 助手回复]
		question := "请根据之前的分类，详细说明：您遇到的具体问题是什么？涉及哪些订单或产品？"

		response2, err := sess.Run(ctx, question)
		if err != nil {
			return state, fmt.Errorf("信息收集失败: %w", err)
		}

		state["detailed_info"] = response2

		// 现在消息历史应该有4条
		messages := sess.GetMessages()
		log.Printf("✓ 信息收集完成，当前消息数: %d (已经过2轮对话)\n", len(messages))

		// 打印完整的对话历史
		log.Println("\n【对话历史】:")
		for i, msg := range messages {
			log.Printf("  %d. [%s] %s\n", i+1, msg.Role, msg.Content[:min(len(msg.Content), 80)])
		}

		return state, nil
	})

	// ─────────────────────────────────────────────────
	// 【阶段4】条件判断：根据问题类型路由
	// ─────────────────────────────────────────────────
	builder.AddConditionNode("route_by_type", func(ctx context.Context, state graph.State) (string, error) {
		classification := state["classification"].(string)

		log.Printf("【路由】根据问题类型路由: %s\n", classification[:min(len(classification), 50)])

		// 简单的分类逻辑（实际应该用NLP更精确）
		if contains(classification, "退货") || contains(classification, "退款") {
			return "refund_path", nil
		} else if contains(classification, "订单") || contains(classification, "查询") {
			return "order_tracking_path", nil
		} else if contains(classification, "产品") || contains(classification, "咨询") {
			return "product_path", nil
		}
		return "general_path", nil
	}, map[string]string{
		"refund_path":         "handle_refund",
		"order_tracking_path": "track_order",
		"product_path":        "product_support",
		"general_path":        "general_support",
	})

	// ─────────────────────────────────────────────────
	// 【路径A】退货退款处理：第3、4轮对话
	// ─────────────────────────────────────────────────
	builder.AddNode("handle_refund", graph.NodeTypeLLM, func(ctx context.Context, state graph.State) (graph.State, error) {
		log.Println("【处理】进入退货退款流程")

		sess := state["session"].(session.Session)

		// 第3轮对话：退款专用问询
		response3, _ := sess.Run(ctx, "请告诉我您要求退款的原因（质量问题/尺码不符/不喜欢等）？")

		// 第4轮对话：处理方案
		response4, _ := sess.Run(ctx, "感谢您的信息。我们已为您提交退货申请，预计3-5个工作日处理完毕。")

		messages := sess.GetMessages()
		log.Printf("✓ 退货流程完成，总消息数: %d (已经过4轮对话)\n", len(messages))

		state["final_response"] = response4
		state["conversation_history"] = messages

		return state, nil
	})

	// ─────────────────────────────────────────────────
	// 【路径B】订单追踪：第3、4轮对话
	// ─────────────────────────────────────────────────
	builder.AddNode("track_order", graph.NodeTypeLLM, func(ctx context.Context, state graph.State) (graph.State, error) {
		log.Println("【处理】进入订单追踪流程")

		sess := state["session"].(session.Session)

		// 第3轮：查询订单状态
		response3, _ := sess.Run(ctx, "您的订单ORD001已在运输中，预计明天送达。")

		// 第4轮：追加帮助
		response4, _ := sess.Run(ctx, "我已将追踪链接发送到您的邮箱，您可以随时查看物流状态。")

		messages := sess.GetMessages()
		log.Printf("✓ 订单追踪完成，总消息数: %d\n", len(messages))

		state["final_response"] = response4
		state["conversation_history"] = messages

		return state, nil
	})

	// ─────────────────────────────────────────────────
	// 【路径C】产品支持：第3轮对话
	// ─────────────────────────────────────────────────
	builder.AddNode("product_support", graph.NodeTypeLLM, func(ctx context.Context, state graph.State) (graph.State, error) {
		log.Println("【处理】进入产品支持流程")

		sess := state["session"].(session.Session)

		// 第3轮：产品建议
		response3, _ := sess.Run(ctx, "根据您的需求，我推荐您选择我们的高端系列产品。")

		messages := sess.GetMessages()
		log.Printf("✓ 产品支持完成，总消息数: %d\n", len(messages))

		state["final_response"] = response3
		state["conversation_history"] = messages

		return state, nil
	})

	// ─────────────────────────────────────────────────
	// 【路径D】通用支持：第3轮对话
	// ─────────────────────────────────────────────────
	builder.AddNode("general_support", graph.NodeTypeLLM, func(ctx context.Context, state graph.State) (graph.State, error) {
		log.Println("【处理】进入通用支持流程")

		sess := state["session"].(session.Session)

		// 第3轮：通用回复
		response3, _ := sess.Run(ctx, "感谢您的反馈，我们非常重视您的意见。若有其他问题，欢迎随时联系我们。")

		messages := sess.GetMessages()
		log.Printf("✓ 通用支持完成，总消息数: %d\n", len(messages))

		state["final_response"] = response3
		state["conversation_history"] = messages

		return state, nil
	})

	// ─────────────────────────────────────────────────
	// 【阶段5】清理和结束
	// ─────────────────────────────────────────────────
	builder.AddNode("cleanup", graph.NodeTypeEnd, func(ctx context.Context, state graph.State) (graph.State, error) {
		log.Println("【清理】关闭Session并保存记录")

		sess := state["session"].(session.Session)

		// 获取最终的完整对话历史
		finalMessages := sess.GetMessages()

		// 创建工单记录（用于质检和学习）
		ticket := &Ticket{
			TicketID:    fmt.Sprintf("TKT_%d", time.Now().Unix()),
			CustomerID:  state["customer_id"].(string),
			Subject:     "Graph工作流处理",
			Priority:    "medium",
			Status:      "resolved",
			CreatedAt:   time.Now(),
			ResolvedAt:  time.Now(),
			Description: state["user_inquiry"].(string),
			Solution:    state["final_response"].(string),
		}
		p.addTicket(ticket)

		// 关闭Session
		sess.Close()

		log.Printf("✓ Session已关闭\n")
		log.Printf("✓ 完整对话历史: %d条消息\n", len(finalMessages))
		log.Printf("✓ 工单已创建: %s\n", ticket.TicketID)

		return state, nil
	})

	// ─────────────────────────────────────────────────
	// 连接所有节点
	// ─────────────────────────────────────────────────
	builder.AddEdge("init_session", "classify_problem")
	builder.AddEdge("classify_problem", "collect_info")
	builder.AddEdge("collect_info", "route_by_type")

	// 四个路径都连到cleanup
	builder.AddEdge("handle_refund", "cleanup")
	builder.AddEdge("track_order", "cleanup")
	builder.AddEdge("product_support", "cleanup")
	builder.AddEdge("general_support", "cleanup")

	builder.SetStart("init_session")
	builder.SetEnd("cleanup")

	return builder.Build()
}

// ExecuteIntelligentCustomerServiceGraph 执行智能客服工作流
func (p *ECommerceServicePlatform) ExecuteIntelligentCustomerServiceGraph(
	customerID string,
	inquiry string,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	log.Printf("\n╔════════════════════════════════════════════════════════════════╗\n")
	log.Printf("║   使用Graph+Agent+Session的智能客服工作流                      ║\n")
	log.Printf("║   Graph定义工作流，Agent处理逻辑，Session保留历史              ║\n")
	log.Printf("╚════════════════════════════════════════════════════════════════╝\n\n")

	// 构建工作流
	serviceGraph := p.BuildIntelligentCustomerServiceGraph()

	// 初始化State（包含客户ID和用户问题）
	initialState := graph.State{
		"customer_id":   customerID,
		"user_inquiry":  inquiry,
	}

	log.Printf("客户ID: %s\n", customerID)
	log.Printf("问题: %s\n\n", inquiry)

	// 执行工作流
	startTime := time.Now()
	finalState, err := serviceGraph.Execute(ctx, initialState)
	elapsed := time.Since(startTime)

	if err != nil {
		log.Printf("❌ 工作流执行失败: %v\n", err)
		return err
	}

	// 获取结果
	finalResponse := finalState["final_response"].(string)
	conversationHistory := finalState["conversation_history"].([]*message.Message)

	log.Printf("\n╔════════════════════════════════════════════════════════════════╗\n")
	log.Printf("║                         执行完成                               ║\n")
	log.Printf("╚════════════════════════════════════════════════════════════════╝\n\n")

	log.Printf("【最终回复】\n%s\n\n", finalResponse)

	log.Printf("【对话统计】\n")
	log.Printf("  总消息数: %d\n", len(conversationHistory))
	log.Printf("  对话轮数: %d\n", len(conversationHistory)/2)
	log.Printf("  执行耗时: %.2f秒\n\n", elapsed.Seconds())

	log.Printf("【完整对话记录】\n")
	for i, msg := range conversationHistory {
		if i%2 == 0 {
			log.Printf("\n[轮次%d]\n", i/2+1)
		}
		log.Printf("  %s: %s\n", msg.Role, msg.Content[:min(len(msg.Content), 100)])
	}

	return nil
}

// 辅助函数
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || (len(s) > len(substr)))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
