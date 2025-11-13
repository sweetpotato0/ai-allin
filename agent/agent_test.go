package agent

import (
	"context"
	"testing"

	"github.com/sweetpotato0/ai-allin/contrib/memory/inmemory"
	"github.com/sweetpotato0/ai-allin/message"
	"github.com/sweetpotato0/ai-allin/tool"
)

// MockLLMClient implements LLMClient for testing
type MockLLMClient struct {
	temperature float64
	maxTokens   int64
	model       string
	response    string
}

func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		temperature: 0.7,
		maxTokens:   2000,
		model:       "gpt-4",
		response:    "Mock response",
	}
}

func (m *MockLLMClient) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	msg := message.NewMessage(message.RoleAssistant, m.response)
	msg.Completed = true
	return &GenerateResponse{Message: msg}, nil
}

func (m *MockLLMClient) SetTemperature(temp float64) {
	m.temperature = temp
}

func (m *MockLLMClient) SetMaxTokens(max int64) {
	m.maxTokens = max
}

func (m *MockLLMClient) SetModel(model string) {
	m.model = model
}

func TestNewAgent(t *testing.T) {
	agent := New(
		WithName("TestAgent"),
		WithSystemPrompt("You are a test assistant"),
	)

	if agent.name != "TestAgent" {
		t.Errorf("Expected name TestAgent, got %s", agent.name)
	}

	if agent.systemPrompt != "You are a test assistant" {
		t.Errorf("Expected system prompt, got %s", agent.systemPrompt)
	}

	if agent.maxIterations != 10 {
		t.Errorf("Expected max iterations 10, got %d", agent.maxIterations)
	}
}

func TestAgentClone(t *testing.T) {
	llm := NewMockLLMClient()
	memoryStore := inmemory.NewInMemoryStore()

	original := New(
		WithName("Original"),
		WithSystemPrompt("Original prompt"),
		WithMaxIterations(5),
		WithTemperature(0.5),
		WithProvider(llm),
		WithMemory(memoryStore),
		WithTools(true),
	)

	cloned := original.Clone()

	if cloned.name != original.name {
		t.Errorf("Clone: name not preserved")
	}

	if cloned.systemPrompt != original.systemPrompt {
		t.Errorf("Clone: system prompt not preserved")
	}

	if cloned.memory != original.memory {
		t.Errorf("Clone: memory not cloned")
	}
}

func TestRegisterTool(t *testing.T) {
	agent := New()
	testTool := &tool.Tool{
		Name:        "test_tool",
		Description: "A test tool",
	}

	err := agent.RegisterTool(testTool)
	if err != nil {
		t.Errorf("Failed to register tool: %v", err)
	}

	err = agent.RegisterTool(testTool)
	if err == nil {
		t.Errorf("Expected error when registering duplicate tool")
	}
}

func TestAddMiddleware(t *testing.T) {
	agent := New()

	err := agent.AddMiddleware(nil)
	if err == nil {
		t.Errorf("Expected error when adding nil middleware")
	}
}

func TestAddMessage(t *testing.T) {
	agent := New()
	msg := message.NewMessage(message.RoleUser, "Hello!")
	agent.AddMessage(msg)

	messages := agent.GetMessages()
	found := false
	for _, m := range messages {
		if m.Role == message.RoleUser && m.Text() == "Hello!" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("User message not found")
	}
}

func TestClearMessages(t *testing.T) {
	agent := New(WithSystemPrompt("Test prompt"))
	agent.AddMessage(message.NewMessage(message.RoleUser, "Test"))

	agent.ClearMessages()
	messages := agent.GetMessages()

	for _, m := range messages {
		if m.Role == message.RoleUser {
			t.Errorf("User message found after clear")
		}
	}
}

func TestSetMemory(t *testing.T) {
	agent := New()
	memoryStore := inmemory.NewInMemoryStore()
	agent.SetMemory(memoryStore)

	if !agent.enableMemory {
		t.Errorf("Memory should be enabled")
	}

	if agent.memory != memoryStore {
		t.Errorf("Memory store not set")
	}
}

func TestRegisterPrompt(t *testing.T) {
	agent := New()

	err := agent.RegisterPrompt("greeting", "Hello {{.name}}")
	if err != nil {
		t.Errorf("Failed to register prompt: %v", err)
	}

	err = agent.RegisterPrompt("", "Empty")
	if err == nil {
		t.Errorf("Expected error for empty name")
	}
}

func TestGetMiddlewareChain(t *testing.T) {
	agent := New()
	chain := agent.GetMiddlewareChain()
	if chain == nil {
		t.Errorf("Middleware chain is nil")
	}
}

func TestAgentWithMemoryOption(t *testing.T) {
	memoryStore := inmemory.NewInMemoryStore()
	agent := New(WithMemory(memoryStore))

	if !agent.enableMemory {
		t.Errorf("Memory not enabled")
	}
}

func TestAgentWithProvider(t *testing.T) {
	llm := NewMockLLMClient()
	agent := New(WithProvider(llm))

	if agent.llm != llm {
		t.Errorf("LLM not set correctly")
	}
}

func TestAgentRestoreMessages(t *testing.T) {
	agent := New(WithSystemPrompt("default"))

	customHistory := []*message.Message{
		message.NewMessage(message.RoleSystem, "override"),
		message.NewMessage(message.RoleUser, "hello"),
	}

	agent.RestoreMessages(customHistory)

	messages := agent.GetMessages()
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages after restore, got %d", len(messages))
	}

	if messages[0].Text() != "override" {
		t.Errorf("expected system prompt to be restored, got %s", messages[0].Text())
	}

	agent.RestoreMessages(nil)
	messages = agent.GetMessages()
	if len(messages) == 0 || messages[0].Text() != "default" {
		t.Errorf("expected fallback to default system prompt, got %+v", messages)
	}
}
