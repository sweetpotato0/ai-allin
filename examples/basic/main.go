package main

import (
	"context"
	"fmt"
	"log"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/message"
)

// MockLLMClient is a simple mock LLM for demonstration
type MockLLMClient struct{}

func (m *MockLLMClient) Generate(ctx context.Context, messages []*message.Message, tools []map[string]interface{}) (*message.Message, error) {
	return message.NewMessage(message.RoleAssistant, "Hello! I'm a helpful AI assistant. How can I help you today?"), nil
}

func (m *MockLLMClient) SetTemperature(temp float64) {
	// Mock implementation - does nothing
}

func (m *MockLLMClient) SetMaxTokens(max int64) {
	// Mock implementation - does nothing
}

func (m *MockLLMClient) SetModel(model string) {
	// Mock implementation - does nothing
}

func main() {
	fmt.Println("=== Basic Agent Example ===\n")

	ctx := context.Background()

	// Create agent with options pattern
	ag := agent.New(
		agent.WithName("MyAssistant"),
		agent.WithSystemPrompt("You are a helpful AI assistant"),
		agent.WithProvider(&MockLLMClient{}),
		agent.WithMaxIterations(5),
		agent.WithTemperature(0.7),
	)

	// Run the agent
	result, err := ag.Run(ctx, "Hello, how are you?")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Printf("User: Hello, how are you?\n")
	fmt.Printf("Agent: %s\n", result)

	// Show message history
	fmt.Println("\nMessage History:")
	for i, msg := range ag.GetMessages() {
		fmt.Printf("%d. [%s] %s\n", i+1, msg.Role, msg.Content)
	}
}
