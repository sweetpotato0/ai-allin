package message

import "time"

// Role represents the role of the message sender
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleTool      Role = "tool"
)

// Message represents a single message in a conversation
type Message struct {
	ID        string                 `json:"id"`
	Role      Role                   `json:"role"`
	Content   string                 `json:"content"`
	ToolCalls []ToolCall             `json:"tool_calls,omitempty"`
	ToolID    string                 `json:"tool_id,omitempty"` // For tool response messages
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// ToolCall represents a tool invocation request
type ToolCall struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Args     map[string]interface{} `json:"args"`
	Response string                 `json:"response,omitempty"` // Filled after tool execution
}

// NewMessage creates a new message with the given role and content
func NewMessage(role Role, content string) *Message {
	return &Message{
		ID:        generateID(),
		Role:      role,
		Content:   content,
		CreatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
}

// NewToolCallMessage creates a message with tool calls
func NewToolCallMessage(toolCalls []ToolCall) *Message {
	return &Message{
		ID:        generateID(),
		Role:      RoleAssistant,
		ToolCalls: toolCalls,
		CreatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
}

// NewToolResponseMessage creates a tool response message
func NewToolResponseMessage(toolID, content string) *Message {
	return &Message{
		ID:        generateID(),
		Role:      RoleTool,
		Content:   content,
		ToolID:    toolID,
		CreatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
}

// generateID generates a unique message ID
func generateID() string {
	// Simple implementation using timestamp
	// In production, consider using UUID
	return time.Now().Format("20060102150405.000000")
}
