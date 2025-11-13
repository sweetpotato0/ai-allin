package main

import (
	"context"
	"log"
	"os"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/contrib/provider/claude"
	"github.com/sweetpotato0/ai-allin/memory/store"
	"github.com/sweetpotato0/ai-allin/message"
)

// MockLLMProvider 模拟LLM提供商（备用，当API密钥不可用时使用）
type MockLLMProvider struct{}

func NewMockLLMProvider() *MockLLMProvider {
	return &MockLLMProvider{}
}

func (m *MockLLMProvider) Generate(ctx context.Context, req *agent.GenerateRequest) (*agent.GenerateResponse, error) {
	// 返回一个简单的模拟响应
	response := "感谢您的咨询！我已经查看了您的信息。根据您的问题，我会为您提供最佳解决方案。"
	msg := message.NewMessage(message.RoleAssistant, response)
	msg.Completed = true
	return &agent.GenerateResponse{Message: msg}, nil
}

func (m *MockLLMProvider) SetTemperature(temp float64) {}
func (m *MockLLMProvider) SetMaxTokens(max int64)      {}
func (m *MockLLMProvider) SetModel(model string)       {}

func main() {
	// 配置日志
	log.SetFlags(log.Ltime | log.Lshortfile)
	log.SetOutput(os.Stdout)

	log.Println("╔════════════════════════════════════════════════════════════════╗")
	log.Println("║   生产级电商智能客服平台 - 完整功能演示                      ║")
	log.Println("║   Production-Grade E-Commerce AI Customer Service Platform   ║")
	log.Println("╚════════════════════════════════════════════════════════════════╝")

	// 获取Claude API密钥
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Println("⚠️  警告: 未设置 ANTHROPIC_API_KEY 环境变量")
		log.Println("   使用本地测试模式")
		log.Println("   设置方法: export ANTHROPIC_API_KEY=your-api-key")
	}

	// 获取Claude API密钥
	baseUrl := os.Getenv("ANTHROPIC_BASE_URL")
	// 初始化LLM提供商
	var llmProvider agent.LLMClient

	if apiKey != "" {
		// 使用Claude提供商（框架内置）
		config := claude.DefaultConfig(apiKey, baseUrl)
		config.Model = "claude-3-5-sonnet-20241022"
		config.Temperature = 0.7
		config.MaxTokens = 1024

		llmProvider = claude.New(config)
		log.Println("✓ 使用Claude 3.5 Sonnet LLM提供商（框架内置）")
	} else {
		// 降级到Mock模式
		llmProvider = NewMockLLMProvider()
		log.Println("✓ 使用Mock LLM提供商（测试模式）")
	}

	// 创建内存存储（使用框架内置实现）
	memoryStore := store.NewInMemoryStore()

	// 初始化平台
	platform := NewECommerceServicePlatform(llmProvider, memoryStore)

	log.Println("✓ 平台初始化完成")

	// ================================
	// 演示场景1：单客户咨询
	// ================================
	log.Println("╔════════════════════════════════════════════════════════════════╗")
	log.Println("║ 演示1: 单个客户咨询处理流程                                    ║")
	log.Println("║ Demonstration 1: Single Customer Inquiry Processing           ║")
	log.Println("╚════════════════════════════════════════════════════════════════╝")

	_, err := platform.HandleCustomerInquiry("CUST001",
		"你好，我的订单ORD001已经发货但一直没有进展。我是Gold VIP会员，"+
			"想知道为什么这么慢？是否可以加急？")
	if err != nil {
		log.Printf("错误: %v\n", err)
	}

	// ================================
	// 演示场景2：多轮对话
	// ================================
	log.Println("╔════════════════════════════════════════════════════════════════╗")
	log.Println("║ 演示2: 多轮对话场景（展示Session和Context的交互）            ║")
	log.Println("║ Demonstration 2: Multi-turn Conversation (Session & Context)  ║")
	log.Println("╚════════════════════════════════════════════════════════════════╝")

	if err := platform.MultiTurnConversationScenario("CUST002"); err != nil {
		log.Printf("错误: %v\n", err)
	}

	// ================================
	// 演示场景3：多Agent协调
	// ================================
	log.Println("╔════════════════════════════════════════════════════════════════╗")
	log.Println("║ 演示3: 多Agent协调流程（核心功能展示）                        ║")
	log.Println("║ Demonstration 3: Multi-Agent Orchestration (Core Features)    ║")
	log.Println("╚════════════════════════════════════════════════════════════════╝")

	if err := platform.MultiAgentOrchestration("CUST001"); err != nil {
		log.Printf("错误: %v\n", err)
	}

	// ================================
	// 演示场景4：并行处理
	// ================================
	log.Println("\n╔════════════════════════════════════════════════════════════════╗")
	log.Println("║ 演示4: 并行处理多个客户咨询（高并发场景）                    ║")
	log.Println("║ Demonstration 4: Parallel Processing (High Concurrency)       ║")
	log.Println("╚════════════════════════════════════════════════════════════════╝")

	platform.ParallelCustomerHandling(5)

	// ================================
	// 演示场景5：运营数据分析
	// ================================
	log.Println("\n╔════════════════════════════════════════════════════════════════╗")
	log.Println("║ 演示5: 运营指标和分析                                          ║")
	log.Println("║ Demonstration 5: Operational Metrics and Analytics            ║")
	log.Println("╚════════════════════════════════════════════════════════════════╝")

	platform.AnalyzeOperationalMetrics()

	// ================================
	// 总结
	// ================================
	log.Println("\n╔════════════════════════════════════════════════════════════════╗")
	log.Println("║ 完整演示总结                                                   ║")
	log.Println("║ Complete Demonstration Summary                                ║")
	log.Println("╚════════════════════════════════════════════════════════════════╝")

	log.Println("✓ 框架核心组件使用：")
	log.Println("  1. Session (会话管理)")
	log.Println("     - 管理多个客户的独立会话")
	log.Println("     - SessionManager维护所有活跃Session")
	log.Println("     - 支持并行处理多个Session")

	log.Println("  2. Context (对话历史)")
	log.Println("     - 每个Agent维护一个独立的Context")
	log.Println("     - Context记录消息历史，支持多轮对话")
	log.Println("     - 自动管理消息大小和内存")

	log.Println("  3. Agent (智能代理)")
	log.Println("     - 多个专业Agent执行不同任务")
	log.Println("     - 客服Agent处理客户咨询")
	log.Println("     - 运营Agent分析数据")
	log.Println("     - QA Agent保证质量")
	log.Println("     - 知识Agent管理文档")

	log.Println("  4. Middleware (中间件)")
	log.Println("     - RequestLogger: 记录请求信息")
	log.Println("     - ContextEnricher: 增强上下文信息")
	log.Println("     - RateLimiter: 限制请求速率")
	log.Println("     - ErrorHandler: 统一错误处理")

	log.Println("  5. Memory (智能记忆)")
	log.Println("     - 存储重要客户信息")
	log.Println("     - 支持相似性搜索")
	log.Println("     - 长期学习和改进")

	log.Println("  6. Tool (工具系统)")
	log.Println("     - Agent可以调用各种工具")
	log.Println("     - 查询订单、处理退款等业务操作")

	log.Println("  7. Prompt (提示词管理)")
	log.Println("     - 灵活的提示词模板系统")
	log.Println("     - 支持多种Agent角色")

	log.Println("  8. Vector (向量存储)")
	log.Println("     - 支持相似度搜索")
	log.Println("     - 知识库检索")
	log.Println("     - 语义理解")

	log.Println("✓ 生产级功能特性：")
	log.Println("  - 并发安全: RWMutex保护共享数据")
	log.Println("  - 错误处理: 完整的错误恢复机制")
	log.Println("  - 监控指标: 实时追踪平台性能")
	log.Println("  - 扩展性: 易于添加新Agent和功能")
	log.Println("  - 可靠性: Timeout控制，资源管理")
	log.Println("  - 可观测性: 详细的日志和追踪")

	if apiKey != "" {
		log.Println("\n✓ Claude AI集成（框架内置）：")
		log.Println("  - 使用Claude 3.5 Sonnet模型")
		log.Println("  - 实时API调用")
		log.Println("  - 自然语言理解和生成")
		log.Println("  - 自动错误恢复和降级处理")
		log.Println("  - 支持流式响应（可选）")
	}

	log.Println("\n╔════════════════════════════════════════════════════════════════╗")
	log.Println("║ 演示完成！✓                                                    ║")
	log.Println("╚════════════════════════════════════════════════════════════════╝")
}
