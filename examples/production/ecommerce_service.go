package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/memory"
	"github.com/sweetpotato0/ai-allin/middleware"
	"github.com/sweetpotato0/ai-allin/middleware/enricher"
	"github.com/sweetpotato0/ai-allin/middleware/errorhandler"
	"github.com/sweetpotato0/ai-allin/middleware/limiter"
	"github.com/sweetpotato0/ai-allin/middleware/logger"
	"github.com/sweetpotato0/ai-allin/prompt"
	"github.com/sweetpotato0/ai-allin/runner"
	"github.com/sweetpotato0/ai-allin/session"
	"github.com/sweetpotato0/ai-allin/tool"
)

// ================================
// 核心业务模型
// ================================

// Customer 客户信息
type Customer struct {
	ID           string
	Name         string
	Email        string
	Phone        string
	VIPLevel     string // "bronze", "silver", "gold", "platinum"
	RegisteredAt time.Time
	TotalSpent   float64
	OrderCount   int
}

// Order 订单信息
type Order struct {
	OrderID     string
	CustomerID  string
	Items       []OrderItem
	TotalAmount float64
	Status      string // "pending", "shipped", "delivered", "returned"
	CreatedAt   time.Time
	UpdatedAt   time.Time
	TrackingURL string
}

type OrderItem struct {
	ProductID   string
	ProductName string
	Quantity    int
	UnitPrice   float64
}

// Ticket 客服工单
type Ticket struct {
	TicketID    string
	CustomerID  string
	Subject     string
	Priority    string // "low", "medium", "high", "critical"
	Status      string // "open", "in_progress", "resolved", "closed"
	CreatedAt   time.Time
	ResolvedAt  time.Time
	AssignedTo  string
	Description string
	Solution    string
}

// KnowledgeBase 知识库
type KnowledgeBase struct {
	articles map[string]KBArticle
	mu       sync.RWMutex
}

type KBArticle struct {
	ID          string
	Title       string
	Content     string
	Category    string
	Tags        []string
	CreatedAt   time.Time
	UpdateedAt  time.Time
	ViewCount   int
	HelpfulRate float64
}

// ECommerceServicePlatform 电商服务平台
type ECommerceServicePlatform struct {
	// 核心组件
	sessionManager  *session.Manager
	agentFactory    *AgentFactory
	vectorStore     *MockVectorStore
	memoryStore     memory.MemoryStore
	knowledgeBase   *KnowledgeBase
	promptTemplates *prompt.Manager
	parallelRunner  *runner.ParallelRunner

	// 业务数据
	customers      map[string]*Customer
	orders         map[string]*Order
	tickets        map[string]*Ticket
	customersMutex sync.RWMutex
	ordersMutex    sync.RWMutex
	ticketsMutex   sync.RWMutex

	// 监控
	metrics *PlatformMetrics
}

type PlatformMetrics struct {
	TotalRequests       int64
	SuccessfulRequests  int64
	FailedRequests      int64
	AverageResponseTime time.Duration
	ActiveSessions      int
	mu                  sync.RWMutex
}

// AgentFactory 高级Agent工厂
type AgentFactory struct {
	llmProvider  agent.LLMClient
	vectorStore  *MockVectorStore
	memoryStore  memory.MemoryStore
	toolRegistry *tool.Registry
}

// ================================
// 初始化和配置
// ================================

// NewECommerceServicePlatform 创建电商服务平台
func NewECommerceServicePlatform(llmProvider agent.LLMClient, memStore memory.MemoryStore) *ECommerceServicePlatform {
	platform := &ECommerceServicePlatform{
		sessionManager: session.NewManager(),
		agentFactory: &AgentFactory{
			llmProvider: llmProvider,
			vectorStore: NewMockVectorStore(),
			memoryStore: memStore,
		},
		vectorStore:     NewMockVectorStore(),
		memoryStore:     memStore,
		knowledgeBase:   &KnowledgeBase{articles: make(map[string]KBArticle)},
		promptTemplates: prompt.NewManager(),
		parallelRunner:  runner.NewParallelRunner(10), // 最大并发10个任务
		customers:       make(map[string]*Customer),
		orders:          make(map[string]*Order),
		tickets:         make(map[string]*Ticket),
		metrics:         &PlatformMetrics{},
	}

	// 初始化提示词模板
	platform.initPromptTemplates()

	// 初始化知识库
	platform.initKnowledgeBase()

	// 初始化演示数据
	platform.initDemoData()

	return platform
}

// initPromptTemplates 初始化提示词模板
func (p *ECommerceServicePlatform) initPromptTemplates() {
	// 客服代理提示词
	customerServicePrompt := `你是一个专业的电商客服代理，服务于顶级电商平台。

【核心职责】
1. 快速准确地回答客户问题
2. 处理订单相关问题（查询、修改、取消）
3. 处理退货退款（分析原因、提供方案、执行流程）
4. 管理客户投诉和纠纷
5. 推荐产品和优化购物体验
6. 记录客户信息用于后续跟进

【可用工具】
- query_order: 查询订单详情
- process_refund: 处理退款
- check_vip_status: 检查VIP等级
- search_knowledge_base: 搜索知识库
- create_ticket: 创建工单
- add_customer_note: 添加客户备注

【服务标准】
- 响应时间: < 2秒
- 首次解决率: > 85%
- 客户满意度: > 4.5/5
- 专业友好的语气
- 提供个性化方案`

	p.promptTemplates.RegisterString("customer_service", customerServicePrompt)

	// 运营分析代理提示词
	operationPrompt := `你是一个专业的运营分析专家。

【分析职责】
1. 分析客户行为和购买趋势
2. 识别高价值客户和流失风险客户
3. 生成性能报告和改进建议
4. 预测市场趋势和需求
5. 优化客服流程和效率
6. 识别常见问题和解决方案

【分析工具】
- analyze_customer_segment: 客户分段分析
- predict_churn_risk: 流失风险预测
- generate_report: 生成分析报告
- get_metrics: 获取关键指标`

	p.promptTemplates.RegisterString("operation", operationPrompt)

	// 质量保证代理提示词
	qaPrompt := `你是一个严格的质量保证专家。

【质检职责】
1. 审查客服的回复质量
2. 确保合规性和政策遵守
3. 提供改进建议
4. 监控服务指标
5. 记录问题和模式
6. 推荐培训内容`

	p.promptTemplates.RegisterString("qa", qaPrompt)

	// 知识管理代理提示词
	knowledgePrompt := `你是一个知识管理专家。

【职责】
1. 组织和维护知识库
2. 创建高质量的帮助文档
3. 识别知识库缺口
4. 优化搜索和检索
5. 更新过时内容`

	p.promptTemplates.RegisterString("knowledge", knowledgePrompt)
}

// initKnowledgeBase 初始化知识库
func (p *ECommerceServicePlatform) initKnowledgeBase() {
	articles := []KBArticle{
		{
			ID:       "kb_001",
			Title:    "如何追踪订单",
			Content:  "点击订单详情页面，查看实时追踪信息...",
			Category: "订单",
			Tags:     []string{"订单", "追踪", "物流"},
		},
		{
			ID:       "kb_002",
			Title:    "退货退款流程",
			Content:  "我们提供30天无理由退货，请按照以下步骤...",
			Category: "退货",
			Tags:     []string{"退货", "退款", "流程"},
		},
		{
			ID:       "kb_003",
			Title:    "VIP会员权益",
			Content:  "不同等级的VIP享受不同的权益，详细信息如下...",
			Category: "会员",
			Tags:     []string{"VIP", "会员", "权益"},
		},
		{
			ID:       "kb_004",
			Title:    "支付方式和安全",
			Content:  "我们支持多种安全的支付方式...",
			Category: "支付",
			Tags:     []string{"支付", "安全", "方法"},
		},
		{
			ID:       "kb_005",
			Title:    "产品保修政策",
			Content:  "所有产品享受1年保修服务...",
			Category: "保修",
			Tags:     []string{"保修", "质量", "保障"},
		},
	}

	for _, article := range articles {
		p.knowledgeBase.articles[article.ID] = article
	}
}

// initDemoData 初始化演示数据
func (p *ECommerceServicePlatform) initDemoData() {
	// 创建示例客户
	customers := []*Customer{
		{
			ID:           "CUST001",
			Name:         "张三",
			Email:        "zhangsan@example.com",
			Phone:        "13800138000",
			VIPLevel:     "gold",
			RegisteredAt: time.Now().AddDate(-2, 0, 0),
			TotalSpent:   50000,
			OrderCount:   25,
		},
		{
			ID:           "CUST002",
			Name:         "李四",
			Email:        "lisi@example.com",
			Phone:        "13800138001",
			VIPLevel:     "silver",
			RegisteredAt: time.Now().AddDate(-1, 0, 0),
			TotalSpent:   15000,
			OrderCount:   8,
		},
		{
			ID:           "CUST003",
			Name:         "王五",
			Email:        "wangwu@example.com",
			Phone:        "13800138002",
			VIPLevel:     "bronze",
			RegisteredAt: time.Now().AddDate(0, -3, 0),
			TotalSpent:   5000,
			OrderCount:   2,
		},
	}

	for _, c := range customers {
		p.customers[c.ID] = c
	}

	// 创建示例订单
	orders := []*Order{
		{
			OrderID:     "ORD001",
			CustomerID:  "CUST001",
			TotalAmount: 2999.99,
			Status:      "shipped",
			CreatedAt:   time.Now().AddDate(0, 0, -5),
			TrackingURL: "https://tracking.example.com/ORD001",
			Items: []OrderItem{
				{ProductID: "PROD001", ProductName: "iPhone 15", Quantity: 1, UnitPrice: 2999.99},
			},
		},
		{
			OrderID:     "ORD002",
			CustomerID:  "CUST002",
			TotalAmount: 599.99,
			Status:      "delivered",
			CreatedAt:   time.Now().AddDate(0, 0, -15),
			TrackingURL: "https://tracking.example.com/ORD002",
			Items: []OrderItem{
				{ProductID: "PROD002", ProductName: "AirPods Pro", Quantity: 1, UnitPrice: 599.99},
			},
		},
	}

	for _, o := range orders {
		p.orders[o.OrderID] = o
	}
}

// ================================
// Agent创建方法
// ================================

// CreateCustomerServiceAgent 创建客服Agent
func (af *AgentFactory) CreateCustomerServiceAgent(agentID string) *agent.Agent {
	systemPrompt := `你是一个专业的电商客服代理。

你的核心职责：
1. 快速解决客户问题
2. 提供个性化的服务
3. 记录和追踪问题解决过程
4. 主动提供帮助和建议

你必须：
- 首先查询客户信息和订单历史
- 根据VIP等级提供差异化服务
- 对所有操作创建工单记录
- 保留完整的对话历史供质检和培训`

	ag := agent.New(
		agent.WithName(agentID),
		agent.WithSystemPrompt(systemPrompt),
		agent.WithMaxIterations(8),
		agent.WithTemperature(0.6),
		agent.WithMemory(af.memoryStore),
		agent.WithTools(true),
		agent.WithProvider(af.llmProvider),
	)

	return ag
}

// CreateOperationAgent 创建运营分析Agent
func (af *AgentFactory) CreateOperationAgent(agentID string) *agent.Agent {
	systemPrompt := `你是一个专业的运营分析师。

你的职责：
1. 分析客户数据和行为模式
2. 生成详细的业务报告
3. 提供数据驱动的建议
4. 预测趋势和风险`

	ag := agent.New(
		agent.WithName(agentID),
		agent.WithSystemPrompt(systemPrompt),
		agent.WithTemperature(0.3), // 分析需要低温度
		agent.WithMemory(af.memoryStore),
	)

	return ag
}

// CreateQAAgent 创建质保Agent
func (af *AgentFactory) CreateQAAgent(agentID string) *agent.Agent {
	systemPrompt := `你是一个严格的质量保证专家。

你的职责：
1. 审查服务质量
2. 检查政策遵守
3. 提供改进建议
4. 监控关键指标`

	ag := agent.New(
		agent.WithName(agentID),
		agent.WithSystemPrompt(systemPrompt),
		agent.WithTemperature(0.2), // QA需要非常严格
		agent.WithMemory(af.memoryStore),
	)

	return ag
}

// CreateKnowledgeAgent 创建知识管理Agent
func (af *AgentFactory) CreateKnowledgeAgent(agentID string) *agent.Agent {
	systemPrompt := `你是一个知识管理专家。

你的职责：
1. 组织知识库内容
2. 创建高质量文档
3. 识别知识缺口
4. 优化内容结构`

	ag := agent.New(
		agent.WithName(agentID),
		agent.WithSystemPrompt(systemPrompt),
		agent.WithTemperature(0.4),
		agent.WithMemory(af.memoryStore),
	)

	return ag
}

// ================================
// 业务流程方法
// ================================

// HandleCustomerInquiry 处理客户咨询（完整流程）
func (p *ECommerceServicePlatform) HandleCustomerInquiry(
	customerID string,
	inquiry string,
) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	startTime := time.Now()
	defer p.recordMetrics(startTime)

	log.Printf("\n=== 客户咨询处理 ===\n")
	log.Printf("客户ID: %s\n", customerID)
	log.Printf("咨询内容: %s\n", inquiry)

	// 1. 验证客户
	customer, err := p.getCustomer(customerID)
	if err != nil {
		return "", fmt.Errorf("客户不存在: %w", err)
	}

	log.Printf("✓ 客户验证成功: %s (VIP等级: %s)\n", customer.Name, customer.VIPLevel)

	// 2. 创建Session（代表这个客户的本次服务会话）
	sessionID := fmt.Sprintf("cs_%s_%d", customerID, time.Now().UnixNano())
	csAgent := p.agentFactory.CreateCustomerServiceAgent("cs_agent")

	// 3. 配置Agent中间件
	p.configureAgentMiddleware(csAgent, sessionID, customerID)

	// 4. 创建Session
	sess, err := p.sessionManager.Create(sessionID, csAgent)
	if err != nil {
		return "", fmt.Errorf("创建session失败: %w", err)
	}
	defer sess.Close()

	log.Printf("✓ Session创建成功: %s\n", sessionID)

	// 5. 执行Agent处理（Session内的Agent Context会记录对话历史）
	log.Printf("\n→ 调用客服Agent处理...\n")
	response, err := sess.Run(ctx, inquiry)
	if err != nil {
		p.metrics.FailedRequests++
		return "", fmt.Errorf("处理失败: %w", err)
	}

	p.metrics.SuccessfulRequests++

	// 6. 生成工单
	subject := inquiry
	if len(inquiry) > 50 {
		subject = inquiry[:50]
	}
	ticket := &Ticket{
		TicketID:    fmt.Sprintf("TKT_%d", time.Now().Unix()),
		CustomerID:  customerID,
		Subject:     subject,
		Priority:    p.determinePriority(customer),
		Status:      "open",
		CreatedAt:   time.Now(),
		AssignedTo:  "cs_agent",
		Description: inquiry,
	}

	p.addTicket(ticket)
	log.Printf("✓ 工单创建成功: %s (优先级: %s)\n", ticket.TicketID, ticket.Priority)

	// 7. 保存对话到知识库/内存
	messages := sess.GetMessages()
	log.Printf("✓ 对话记录保存 (共%d条消息)\n", len(messages))

	log.Printf("\n← Agent回复:\n%s\n", response)

	return response, nil
}

// MultiTurnConversationScenario 多轮对话场景
func (p *ECommerceServicePlatform) MultiTurnConversationScenario(customerID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	log.Printf("\n=== 多轮对话场景 ===\n")
	log.Printf("客户ID: %s\n", customerID)

	// 创建一个长期Session（一个用户会话）
	sessionID := fmt.Sprintf("conv_%s_%d", customerID, time.Now().UnixNano())
	csAgent := p.agentFactory.CreateCustomerServiceAgent("cs_agent")
	p.configureAgentMiddleware(csAgent, sessionID, customerID)

	sess, _ := p.sessionManager.Create(sessionID, csAgent)
	defer sess.Close()

	// 模拟多轮对话
	conversations := []struct {
		Turn     int
		UserMsg  string
		Expected string
	}{
		{
			Turn:    1,
			UserMsg: "你好，我想查询订单ORD001的状态",
		},
		{
			Turn:    2,
			UserMsg: "订单什么时候能到？",
		},
		{
			Turn:    3,
			UserMsg: "如果收到后发现有问题可以退货吗？",
		},
		{
			Turn:    4,
			UserMsg: "好的，谢谢你的帮助！",
		},
	}

	for _, conv := range conversations {
		log.Printf("\n[轮次%d] 用户: %s\n", conv.Turn, conv.UserMsg)

		// 每次Run()都会在Agent的Context中记录对话历史
		response, err := sess.Run(ctx, conv.UserMsg)
		if err != nil {
			log.Printf("错误: %v\n", err)
			continue
		}

		log.Printf("[轮次%d] Agent: %s\n", conv.Turn, response)

		// 关键：显示Context中的消息累积
		messages := sess.GetMessages()
		log.Printf("[轮次%d] Context消息数: %d\n", conv.Turn, len(messages))

		time.Sleep(300 * time.Millisecond)
	}

	log.Printf("\n✓ 多轮对话完成，Session Context记录了%d条消息\n", len(sess.GetMessages()))

	return nil
}

// MultiAgentOrchestration 多Agent协调流程
func (p *ECommerceServicePlatform) MultiAgentOrchestration(customerID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	log.Printf("\n=== 多Agent协调流程 ===\n")
	log.Printf("客户ID: %s\n\n", customerID)

	// ===== 第1阶段：客服处理 =====
	log.Printf("【阶段1】客服Agent处理客户问题\n")
	log.Printf("──────────────────────────\n")

	csSessionID := fmt.Sprintf("cs_%s_%d", customerID, time.Now().UnixNano())
	csAgent := p.agentFactory.CreateCustomerServiceAgent("cs_agent")
	p.configureAgentMiddleware(csAgent, csSessionID, customerID)

	csSess, _ := p.sessionManager.Create(csSessionID, csAgent)
	defer csSess.Close()

	csResponse, _ := csSess.Run(ctx,
		"我的订单ORD001已经5天没有更新了，现在还在深圳，能帮我查一下为什么这么慢吗？"+
			"另外我是Gold VIP会员，这个速度是否符合我们的承诺？")

	log.Printf("客服方案: %s\n", csResponse)
	log.Printf("✓ 客服Session Context消息数: %d\n\n", len(csSess.GetMessages()))

	// ===== 第2阶段：运营分析 =====
	log.Printf("【阶段2】运营Agent分析客户价值\n")
	log.Printf("──────────────────────────\n")

	opSessionID := fmt.Sprintf("op_%s_%d", customerID, time.Now().UnixNano())
	opAgent := p.agentFactory.CreateOperationAgent("op_agent")
	p.configureAgentMiddleware(opAgent, opSessionID, customerID)

	opSess, _ := p.sessionManager.Create(opSessionID, opAgent)
	defer opSess.Close()

	opResponse, _ := opSess.Run(ctx,
		"分析客户CUST001的价值等级，包括总消费额(50000元)、"+
			"订单数(25)、VIP等级(Gold)，"+
			"该客户是否是高价值客户？是否有流失风险？")

	log.Printf("分析结果: %s\n", opResponse)
	log.Printf("✓ 运营Session Context消息数: %d\n\n", len(opSess.GetMessages()))

	// ===== 第3阶段：QA审查 =====
	log.Printf("【阶段3】QA Agent审查服务质量\n")
	log.Printf("──────────────────────────\n")

	qaSessionID := fmt.Sprintf("qa_%s_%d", customerID, time.Now().UnixNano())
	qaAgent := p.agentFactory.CreateQAAgent("qa_agent")
	p.configureAgentMiddleware(qaAgent, qaSessionID, customerID)

	qaSess, _ := p.sessionManager.Create(qaSessionID, qaAgent)
	defer qaSess.Close()

	qaResponse, _ := qaSess.Run(ctx,
		"审查以下客服回复的质量："+csResponse+
			"这个回复是否满足公司的服务标准？"+
			"是否有改进空间？"+
			"是否遵守VIP服务政策？")

	log.Printf("QA意见: %s\n", qaResponse)
	log.Printf("✓ QA Session Context消息数: %d\n\n", len(qaSess.GetMessages()))

	// ===== 第4阶段：知识更新 =====
	log.Printf("【阶段4】知识管理Agent更新知识库\n")
	log.Printf("──────────────────────────\n")

	kbSessionID := fmt.Sprintf("kb_%s_%d", customerID, time.Now().UnixNano())
	kbAgent := p.agentFactory.CreateKnowledgeAgent("kb_agent")
	p.configureAgentMiddleware(kbAgent, kbSessionID, customerID)

	kbSess, _ := p.sessionManager.Create(kbSessionID, kbAgent)
	defer kbSess.Close()

	kbResponse, _ := kbSess.Run(ctx,
		"基于以上客户案例，应该在知识库中添加什么内容？"+
			"比如物流延迟的处理流程、VIP权益说明等。"+
			"现在应该添加什么新的知识文章？")

	log.Printf("知识更新建议: %s\n", kbResponse)
	log.Printf("✓ 知识管理Session Context消息数: %d\n\n", len(kbSess.GetMessages()))

	// ===== 总结 =====
	log.Printf("\n【流程总结】\n")
	log.Printf("──────────────────────────\n")
	log.Printf("✓ 4个Agent各自独立维护了自己的Session和Context\n")
	log.Printf("✓ 客服Session (ID: %s) 记录了%d条消息\n", csSessionID, len(csSess.GetMessages()))
	log.Printf("✓ 运营Session (ID: %s) 记录了%d条消息\n", opSessionID, len(opSess.GetMessages()))
	log.Printf("✓ QA Session (ID: %s) 记录了%d条消息\n", qaSessionID, len(qaSess.GetMessages()))
	log.Printf("✓ 知识Session (ID: %s) 记录了%d条消息\n", kbSessionID, len(kbSess.GetMessages()))
	log.Printf("✓ 总活跃Sessions: %d\n", p.sessionManager.Count())

	return nil
}

// ParallelCustomerHandling 并行处理多个客户
func (p *ECommerceServicePlatform) ParallelCustomerHandling(customerCount int) {
	log.Printf("\n=== 使用ParallelRunner并行处理%d个客户的咨询 ===\n", customerCount)

	inquiries := []string{
		"我的订单什么时候到？",
		"这个产品有问题，我要退货",
		"能给我介绍一下VIP权益吗？",
		"我要投诉快递员的服务态度",
		"这个价格太贵了，能便宜点吗？",
	}

	customerIDs := []string{"CUST001", "CUST002", "CUST003"}

	// 准备任务列表
	tasks := make([]*runner.Task, 0, customerCount)
	for i := 0; i < customerCount; i++ {
		custID := customerIDs[i%len(customerIDs)]
		inquiry := inquiries[i%len(inquiries)]

		// 为每个客户创建一个代理和任务
		csAgent := p.agentFactory.CreateCustomerServiceAgent(fmt.Sprintf("cs_agent_%d", i))
		sessionID := fmt.Sprintf("parallel_%s_%d", custID, time.Now().UnixNano()+int64(i))
		p.configureAgentMiddleware(csAgent, sessionID, custID)

		task := &runner.Task{
			ID:    fmt.Sprintf("task_%d_%s", i, custID),
			Agent: csAgent,
			Input: inquiry,
		}
		tasks = append(tasks, task)
	}

	// 使用ParallelRunner并行执行任务
	log.Printf("启动ParallelRunner执行%d个并发任务...\n", len(tasks))
	startTime := time.Now()

	results := p.parallelRunner.RunParallel(context.Background(), tasks)

	elapsed := time.Since(startTime)
	log.Printf("✓ 并行处理完成，耗时: %.2f秒\n\n", elapsed.Seconds())

	// 分析结果
	successCount := 0
	failureCount := 0

	for _, result := range results {
		if result.Error != nil {
			failureCount++
			log.Printf("❌ 任务%s失败: %v\n", result.TaskID, result.Error)
		} else {
			successCount++
			log.Printf("✓ 任务%s成功 (输出长度: %d字符)\n", result.TaskID, len(result.Output))
		}
	}

	log.Printf("\n【并行执行统计】\n")
	log.Printf("成功任务: %d\n", successCount)
	log.Printf("失败任务: %d\n", failureCount)
	log.Printf("总任务数: %d\n", len(results))
	log.Printf("平均响应时间: %.2f秒/任务\n", elapsed.Seconds()/float64(len(results)))
}

// AnalyzeOperationalMetrics 分析运营指标
func (p *ECommerceServicePlatform) AnalyzeOperationalMetrics() {
	log.Printf("\n=== 平台运营指标分析 ===\n")
	log.Printf("────────────────────────\n")

	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()

	log.Printf("总请求数:        %d\n", p.metrics.TotalRequests)
	log.Printf("成功请求:        %d (%.1f%%)\n",
		p.metrics.SuccessfulRequests,
		float64(p.metrics.SuccessfulRequests)*100/float64(p.metrics.TotalRequests))
	log.Printf("失败请求:        %d (%.1f%%)\n",
		p.metrics.FailedRequests,
		float64(p.metrics.FailedRequests)*100/float64(p.metrics.TotalRequests))
	log.Printf("平均响应时间:    %.2f ms\n", float64(p.metrics.AverageResponseTime.Milliseconds()))
	log.Printf("当前活跃Sessions: %d\n\n", p.sessionManager.Count())

	// 客户和订单统计
	p.customersMutex.RLock()
	customerCount := len(p.customers)
	totalSpent := 0.0
	for _, c := range p.customers {
		totalSpent += c.TotalSpent
	}
	p.customersMutex.RUnlock()

	p.ordersMutex.RLock()
	orderCount := len(p.orders)
	p.ordersMutex.RUnlock()

	log.Printf("客户总数:        %d\n", customerCount)
	log.Printf("订单总数:        %d\n", orderCount)
	log.Printf("总销售额:        ¥%.2f\n", totalSpent)
	log.Printf("人均消费:        ¥%.2f\n\n", totalSpent/float64(customerCount))

	// 工单统计
	p.ticketsMutex.RLock()
	openTickets := 0
	for _, t := range p.tickets {
		if t.Status == "open" {
			openTickets++
		}
	}
	p.ticketsMutex.RUnlock()

	log.Printf("开放工单数:      %d\n", openTickets)
	log.Printf("已解决工单数:    %d\n", len(p.tickets)-openTickets)
}

// ================================
// 辅助方法
// ================================

func (p *ECommerceServicePlatform) configureAgentMiddleware(
	ag *agent.Agent,
	sessionID string,
	customerID string,
) {
	// 请求日志中间件
	ag.AddMiddleware(logger.NewRequestLogger(func(msg string) {
		log.Println(msg)
	}))

	// 上下文增强中间件
	ag.AddMiddleware(enricher.NewContextEnricher(func(ctx *middleware.Context) error {
		// Enrich context with metadata
		return nil
	}))

	// 速率限制中间件
	ag.AddMiddleware(limiter.NewRateLimiter(10))

	// 错误处理中间件
	ag.AddMiddleware(errorhandler.NewErrorHandler(func(err error) error {
		return err
	}))
}

func (p *ECommerceServicePlatform) getCustomer(customerID string) (*Customer, error) {
	p.customersMutex.RLock()
	defer p.customersMutex.RUnlock()

	customer, exists := p.customers[customerID]
	if !exists {
		return nil, fmt.Errorf("客户不存在")
	}
	return customer, nil
}

func (p *ECommerceServicePlatform) addTicket(ticket *Ticket) {
	p.ticketsMutex.Lock()
	defer p.ticketsMutex.Unlock()
	p.tickets[ticket.TicketID] = ticket
}

func (p *ECommerceServicePlatform) determinePriority(customer *Customer) string {
	switch customer.VIPLevel {
	case "platinum":
		return "critical"
	case "gold":
		return "high"
	case "silver":
		return "medium"
	default:
		return "low"
	}
}

func (p *ECommerceServicePlatform) recordMetrics(startTime time.Time) {
	duration := time.Since(startTime)
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()

	p.metrics.TotalRequests++
	p.metrics.AverageResponseTime = (p.metrics.AverageResponseTime + duration) / 2
}

// ================================
// Mock实现
// ================================

type MockVectorStore struct{}

func NewMockVectorStore() *MockVectorStore {
	return &MockVectorStore{}
}

func (m *MockVectorStore) SaveEmbedding(ctx context.Context, id string, content string, embedding []float32, metadata map[string]interface{}) error {
	return nil
}

func (m *MockVectorStore) DeleteEmbedding(ctx context.Context, id string) error {
	return nil
}

func (m *MockVectorStore) SearchSimilar(ctx context.Context, query string, k int) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
}

// MockMemoryStore 模拟内存存储实现（用于测试）
type MockMemoryStore struct{}

func (m *MockMemoryStore) AddMemory(ctx context.Context, mem *memory.Memory) error {
	return nil
}

func (m *MockMemoryStore) SearchMemory(ctx context.Context, query string) ([]*memory.Memory, error) {
	return []*memory.Memory{}, nil
}
