package main

import (
	"context"
	"fmt"
	"log"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/message"
	"github.com/sweetpotato0/ai-allin/tool"
)

// MockLLMClient simulates an LLM
type MockLLMClient struct{}

func (m *MockLLMClient) Generate(ctx context.Context, messages []*message.Message, tools []map[string]interface{}) (*message.Message, error) {
	// In a real implementation, this would call an actual LLM
	// For demo, we'll return a message indicating the tool was called
	return message.NewMessage(message.RoleAssistant, "I've calculated the result using the calculator tool."), nil
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
	fmt.Println("=== Tools Example ===\n")

	ctx := context.Background()

	// Create agent
	ag := agent.New(
		agent.WithProvider(&MockLLMClient{}),
		agent.WithSystemPrompt("You are a helpful assistant with access to a calculator."),
	)

	// Register calculator tool
	calculatorTool := &tool.Tool{
		Name:        "calculator",
		Description: "Performs basic arithmetic operations",
		Parameters: []tool.Parameter{
			{
				Name:        "operation",
				Type:        "string",
				Description: "The operation to perform: add, subtract, multiply, divide",
				Required:    true,
				Enum:        []string{"add", "subtract", "multiply", "divide"},
			},
			{
				Name:        "a",
				Type:        "number",
				Description: "First number",
				Required:    true,
			},
			{
				Name:        "b",
				Type:        "number",
				Description: "Second number",
				Required:    true,
			},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			op := args["operation"].(string)
			a := args["a"].(float64)
			b := args["b"].(float64)

			var result float64
			switch op {
			case "add":
				result = a + b
			case "subtract":
				result = a - b
			case "multiply":
				result = a * b
			case "divide":
				if b == 0 {
					return "", fmt.Errorf("division by zero")
				}
				result = a / b
			default:
				return "", fmt.Errorf("unknown operation: %s", op)
			}

			return fmt.Sprintf("%.2f", result), nil
		},
	}

	if err := ag.RegisterTool(calculatorTool); err != nil {
		log.Fatalf("Failed to register tool: %v", err)
	}

	fmt.Println("Registered calculator tool with operations: add, subtract, multiply, divide\n")

	// Test tool execution directly
	result, err := calculatorTool.Execute(ctx, map[string]interface{}{
		"operation": "add",
		"a":         float64(10),
		"b":         float64(5),
	})
	if err != nil {
		log.Fatalf("Tool execution failed: %v", err)
	}

	fmt.Printf("Direct tool call: 10 + 5 = %s\n", result)

	// Test multiplication
	result, err = calculatorTool.Execute(ctx, map[string]interface{}{
		"operation": "multiply",
		"a":         float64(7),
		"b":         float64(8),
	})
	if err != nil {
		log.Fatalf("Tool execution failed: %v", err)
	}

	fmt.Printf("Direct tool call: 7 * 8 = %s\n", result)

	// Show tool JSON schema (for LLM)
	fmt.Println("\nTool JSON Schema for LLM:")
	schema := calculatorTool.ToJSONSchema()
	fmt.Printf("%+v\n", schema)
}
