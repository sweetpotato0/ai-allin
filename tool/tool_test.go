package tool

import (
	"context"
	"testing"
)

func TestToolExecution(t *testing.T) {
	ctx := context.Background()

	tool := &Tool{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters: []Parameter{
			{Name: "input", Type: "string", Description: "Test input", Required: true},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			return args["input"].(string) + "_processed", nil
		},
	}

	result, err := tool.Execute(ctx, map[string]interface{}{"input": "test"})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result != "test_processed" {
		t.Errorf("Expected 'test_processed', got '%s'", result)
	}
}

func TestToolValidation(t *testing.T) {
	ctx := context.Background()

	tool := &Tool{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters: []Parameter{
			{Name: "required_param", Type: "string", Description: "Required parameter", Required: true},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			return "ok", nil
		},
	}

	// Test with missing required parameter
	_, err := tool.Execute(ctx, map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for missing required parameter, got nil")
	}

	// Test with required parameter
	_, err = tool.Execute(ctx, map[string]interface{}{"required_param": "value"})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestRegistry(t *testing.T) {
	registry := NewRegistry()

	tool1 := &Tool{Name: "tool1", Description: "First tool"}
	tool2 := &Tool{Name: "tool2", Description: "Second tool"}

	// Register tools
	if err := registry.Register(tool1); err != nil {
		t.Fatalf("Failed to register tool1: %v", err)
	}

	if err := registry.Register(tool2); err != nil {
		t.Fatalf("Failed to register tool2: %v", err)
	}

	// Test duplicate registration
	if err := registry.Register(tool1); err == nil {
		t.Error("Expected error for duplicate registration, got nil")
	}

	// Test Get
	retrieved, err := registry.Get("tool1")
	if err != nil {
		t.Fatalf("Failed to get tool1: %v", err)
	}

	if retrieved.Name != "tool1" {
		t.Errorf("Expected tool name 'tool1', got '%s'", retrieved.Name)
	}

	// Test List
	tools := registry.List()
	if len(tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(tools))
	}
}
