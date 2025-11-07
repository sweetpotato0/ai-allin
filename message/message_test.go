package message

import (
	"testing"
)

func TestNewMessage(t *testing.T) {
	msg := NewMessage(RoleUser, "Hello, world!")

	if msg.Role != RoleUser {
		t.Errorf("Expected role %s, got %s", RoleUser, msg.Role)
	}

	if msg.Content != "Hello, world!" {
		t.Errorf("Expected content 'Hello, world!', got '%s'", msg.Content)
	}

	if msg.ID == "" {
		t.Error("Expected non-empty ID")
	}

	if msg.CreatedAt.IsZero() {
		t.Error("Expected non-zero created time")
	}
}

func TestNewToolCallMessage(t *testing.T) {
	toolCalls := []ToolCall{
		{ID: "call1", Name: "tool1", Args: map[string]any{"arg1": "value1"}},
	}

	msg := NewToolCallMessage(toolCalls)

	if msg.Role != RoleAssistant {
		t.Errorf("Expected role %s, got %s", RoleAssistant, msg.Role)
	}

	if len(msg.ToolCalls) != 1 {
		t.Errorf("Expected 1 tool call, got %d", len(msg.ToolCalls))
	}

	if msg.ToolCalls[0].Name != "tool1" {
		t.Errorf("Expected tool name 'tool1', got '%s'", msg.ToolCalls[0].Name)
	}
}

func TestNewToolResponseMessage(t *testing.T) {
	msg := NewToolResponseMessage("call1", "result")

	if msg.Role != RoleTool {
		t.Errorf("Expected role %s, got %s", RoleTool, msg.Role)
	}

	if msg.Content != "result" {
		t.Errorf("Expected content 'result', got '%s'", msg.Content)
	}

	if msg.ToolID != "call1" {
		t.Errorf("Expected tool ID 'call1', got '%s'", msg.ToolID)
	}
}
