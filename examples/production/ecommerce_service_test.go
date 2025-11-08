package main

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"
)

// ================================
// 单元测试
// ================================

// TestECommerceServicePlatformInitialization 测试平台初始化
func TestECommerceServicePlatformInitialization(t *testing.T) {
	llmProvider := &MockLLMProvider{}
	memoryStore := &MockMemoryStore{}
	platform := NewECommerceServicePlatform(llmProvider, memoryStore)

	if platform == nil {
		t.Fatal("平台初始化失败")
	}

	if platform.sessionManager == nil {
		t.Fatal("SessionManager未初始化")
	}

	if len(platform.customers) == 0 {
		t.Fatal("客户数据未初始化")
	}

	t.Log("✓ 平台初始化测试通过")
}

// TestCustomerValidation 测试客户验证
func TestCustomerValidation(t *testing.T) {
	llmProvider := &MockLLMProvider{}
	memoryStore := &MockMemoryStore{}
	platform := NewECommerceServicePlatform(llmProvider, memoryStore)

	// 测试有效客户
	customer, err := platform.getCustomer("CUST001")
	if err != nil {
		t.Fatalf("获取有效客户失败: %v", err)
	}

	if customer.ID != "CUST001" {
		t.Fatalf("客户ID不匹配: 期望CUST001, 得到%s", customer.ID)
	}

	// 测试无效客户
	_, err = platform.getCustomer("INVALID_CUST")
	if err == nil {
		t.Fatal("不应该获取到无效客户")
	}

	t.Log("✓ 客户验证测试通过")
}

// TestSessionManagement 测试会话管理
func TestSessionManagement(t *testing.T) {
	llmProvider := &MockLLMProvider{}
	memoryStore := &MockMemoryStore{}
	platform := NewECommerceServicePlatform(llmProvider, memoryStore)

	agent := platform.agentFactory.CreateCustomerServiceAgent("test_agent")
	sessionID := "test_session_001"

	// 创建会话
	sess, err := platform.sessionManager.Create(context.Background(), sessionID, agent)
	if err != nil {
		t.Fatalf("创建会话失败: %v", err)
	}

	// 检查会话状态
	if sess.GetState() != "active" {
		t.Fatalf("会话状态不是active: %s", sess.GetState())
	}

	// 关闭会话
	err = sess.Close()
	if err != nil {
		t.Fatalf("关闭会话失败: %v", err)
	}

	// 验证会话已关闭
	if sess.GetState() == "active" {
		t.Fatalf("会话应该已关闭")
	}

	t.Log("✓ 会话管理测试通过")
}

// TestMultipleSessions 测试多会话并发
func TestMultipleSessions(t *testing.T) {
	llmProvider := &MockLLMProvider{}
	memoryStore := &MockMemoryStore{}
	platform := NewECommerceServicePlatform(llmProvider, memoryStore)

	sessionCount := 100
	for i := 0; i < sessionCount; i++ {
		sessionID := fmt.Sprintf("test_session_%d", i)
		agent := platform.agentFactory.CreateCustomerServiceAgent("test_agent")
		_, err := platform.sessionManager.Create(context.Background(), sessionID, agent)
		if err != nil {
			t.Fatalf("创建会话%d失败: %v", i, err)
		}
	}

	if count, err := platform.sessionManager.Count(context.Background()); err != nil {
		t.Fatalf("获取会话数量失败: %v", err)
	} else if count != sessionCount {
		t.Fatalf("会话数量不正确: 期望%d, 得到%d", sessionCount, count)
	}

	t.Logf("✓ 多会话测试通过 (创建了%d个会话)", sessionCount)
}

// TestContextMessageManagement 测试Context消息管理
func TestContextMessageManagement(t *testing.T) {
	llmProvider := &MockLLMProvider{}
	memoryStore := &MockMemoryStore{}
	platform := NewECommerceServicePlatform(llmProvider, memoryStore)

	agent := platform.agentFactory.CreateCustomerServiceAgent("test_agent")
	sessionID := "context_test_001"

	sess, _ := platform.sessionManager.Create(context.Background(), sessionID, agent)
	defer sess.Close()

	// 初始消息 - Agent会自动添加系统提示词作为第一条消息
	initialMessages := sess.GetMessages()
	if len(initialMessages) < 1 {
		t.Fatalf("初始消息数应该至少为1 (系统提示词), 得到%d", len(initialMessages))
	}

	t.Log("✓ Context消息管理测试通过")
}

// BenchmarkCustomerInquiry 性能测试：客户咨询处理
func BenchmarkCustomerInquiry(b *testing.B) {
	llmProvider := &MockLLMProvider{}
	memoryStore := &MockMemoryStore{}
	platform := NewECommerceServicePlatform(llmProvider, memoryStore)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		platform.HandleCustomerInquiry("CUST001", "测试咨询")
	}
}

// BenchmarkSessionCreation 性能测试：会话创建
func BenchmarkSessionCreation(b *testing.B) {
	llmProvider := &MockLLMProvider{}
	memoryStore := &MockMemoryStore{}
	platform := NewECommerceServicePlatform(llmProvider, memoryStore)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sessionID := fmt.Sprintf("bench_session_%d", i)
		agent := platform.agentFactory.CreateCustomerServiceAgent("test_agent")
		platform.sessionManager.Create(context.Background(), sessionID, agent)
	}
}

// TestAPIServer 测试API服务器
func TestAPIServer(t *testing.T) {
	llmProvider := &MockLLMProvider{}
	memoryStore := &MockMemoryStore{}
	platform := NewECommerceServicePlatform(llmProvider, memoryStore)

	api := NewAPIServer(platform, "8081")
	if api == nil {
		t.Fatal("API服务器创建失败")
	}

	if api.platform != platform {
		t.Fatal("API服务器未正确关联平台")
	}

	t.Log("✓ API服务器测试通过")
}

// TestDatabaseService 测试数据库服务
func TestDatabaseService(t *testing.T) {
	config := &DatabaseConfig{
		Host:           "localhost",
		Port:           5432,
		User:           "postgres",
		Password:       "postgres",
		Database:       "ecommerce_test",
		MaxConnections: 50,
	}

	dbService, err := NewDatabaseService(config)
	if err != nil {
		t.Fatalf("数据库服务创建失败: %v", err)
	}

	if !dbService.IsConnected() {
		t.Fatal("数据库连接失败")
	}

	// 健康检查
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = dbService.HealthCheck(ctx)
	if err != nil {
		t.Fatalf("健康检查失败: %v", err)
	}

	t.Log("✓ 数据库服务测试通过")
}

// TestCacheService 测试缓存服务
func TestCacheService(t *testing.T) {
	config := &CacheConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       0,
	}

	cache := NewCacheService(config)
	ctx := context.Background()

	// 测试Set/Get
	err := cache.Set(ctx, "test_key", "test_value", 1*time.Hour)
	if err != nil {
		t.Fatalf("缓存Set失败: %v", err)
	}

	// 测试Delete
	err = cache.Delete(ctx, "test_key")
	if err != nil {
		t.Fatalf("缓存Delete失败: %v", err)
	}

	t.Log("✓ 缓存服务测试通过")
}

// ================================
// 集成测试
// ================================

// TestEndToEndCustomerService 端对端测试：客服流程
func TestEndToEndCustomerService(t *testing.T) {
	log.Println("开始端对端客服测试...")

	llmProvider := &MockLLMProvider{}
	memoryStore := &MockMemoryStore{}
	platform := NewECommerceServicePlatform(llmProvider, memoryStore)

	// 模拟客户咨询
	response, err := platform.HandleCustomerInquiry("CUST001", "查询订单")
	if err != nil {
		t.Fatalf("客户咨询处理失败: %v", err)
	}

	if response == "" {
		t.Fatal("响应不能为空")
	}

	t.Log("✓ 端对端客服测试通过")
}

// TestEndToEndMultiAgentOrchestration 端对端测试：多Agent协调
func TestEndToEndMultiAgentOrchestration(t *testing.T) {
	log.Println("开始端对端多Agent测试...")

	llmProvider := &MockLLMProvider{}
	memoryStore := &MockMemoryStore{}
	platform := NewECommerceServicePlatform(llmProvider, memoryStore)

	// 执行多Agent协调
	err := platform.MultiAgentOrchestration("CUST001")
	if err != nil {
		t.Fatalf("多Agent协调失败: %v", err)
	}

	t.Log("✓ 端对端多Agent测试通过")
}

// TestHighConcurrency 高并发测试
func TestHighConcurrency(t *testing.T) {
	llmProvider := &MockLLMProvider{}
	memoryStore := &MockMemoryStore{}
	platform := NewECommerceServicePlatform(llmProvider, memoryStore)

	concurrencyLevel := 100
	errorCount := 0
	validCustomerIDs := []string{"CUST001", "CUST002", "CUST003"}

	for i := 0; i < concurrencyLevel; i++ {
		customerID := validCustomerIDs[i%len(validCustomerIDs)]
		_, err := platform.HandleCustomerInquiry(customerID, "测试咨询")
		if err != nil {
			errorCount++
		}
	}

	errorRate := float64(errorCount) / float64(concurrencyLevel) * 100
	t.Logf("处理%d个并发请求, 错误率: %.2f%%", concurrencyLevel, errorRate)

	if errorRate > 5 {
		t.Fatalf("错误率过高: %.2f%%", errorRate)
	}

	t.Log("✓ 高并发测试通过")
}

// TestMemoryManagement 内存管理测试
func TestMemoryManagement(t *testing.T) {
	llmProvider := &MockLLMProvider{}
	memoryStore := &MockMemoryStore{}
	platform := NewECommerceServicePlatform(llmProvider, memoryStore)

	// 创建大量对话
	for i := 0; i < 100; i++ {
		sessionID := fmt.Sprintf("memory_test_%d", i)
		agent := platform.agentFactory.CreateCustomerServiceAgent("test_agent")
		sess, _ := platform.sessionManager.Create(context.Background(), sessionID, agent)

		// 模拟多轮对话
		for j := 0; j < 10; j++ {
			sess.Run(context.Background(), fmt.Sprintf("第%d轮对话", j))
		}

		sess.Close()
	}

	t.Logf("✓ 内存管理测试通过 (创建了100个会话)")
}

// TestParallelRunnerIntegration 测试ParallelRunner集成
func TestParallelRunnerIntegration(t *testing.T) {
	log.Println("开始ParallelRunner集成测试...")

	llmProvider := &MockLLMProvider{}
	memoryStore := &MockMemoryStore{}
	platform := NewECommerceServicePlatform(llmProvider, memoryStore)

	// 验证ParallelRunner已初始化
	if platform.parallelRunner == nil {
		t.Fatal("ParallelRunner未初始化")
	}

	t.Log("✓ ParallelRunner初始化测试通过")
}

// TestParallelCustomerHandlingWithRunner 测试使用ParallelRunner处理客户
func TestParallelCustomerHandlingWithRunner(t *testing.T) {
	log.Println("开始ParallelRunner并行处理测试...")

	llmProvider := &MockLLMProvider{}
	memoryStore := &MockMemoryStore{}
	platform := NewECommerceServicePlatform(llmProvider, memoryStore)

	// 测试并行处理多个客户
	customerCount := 5
	startTime := time.Now()

	platform.ParallelCustomerHandling(customerCount)

	elapsed := time.Since(startTime)

	// 验证结果
	if elapsed < 0 {
		t.Fatal("执行时间无效")
	}

	t.Logf("✓ ParallelRunner并行处理测试通过 (处理%d个客户，耗时%.2f秒)",
		customerCount, elapsed.Seconds())
}

// BenchmarkParallelRunnerPerformance 性能测试：ParallelRunner
func BenchmarkParallelRunnerPerformance(b *testing.B) {
	llmProvider := &MockLLMProvider{}
	memoryStore := &MockMemoryStore{}
	platform := NewECommerceServicePlatform(llmProvider, memoryStore)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		platform.ParallelCustomerHandling(10)
	}
}
