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
	ID        string         `json:"id"`
	Role      Role           `json:"role"`
	Content   string         `json:"content"`
	ToolCalls []ToolCall     `json:"tool_calls,omitempty"`
	ToolID    string         `json:"tool_id,omitempty"` // For tool response messages
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// ToolCall represents a tool invocation request
type ToolCall struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Args     map[string]any `json:"args"`
	Response string         `json:"response,omitempty"` // Filled after tool execution
}

// NewMessage creates a new message with the given role and content
func NewMessage(role Role, content string) *Message {
	return &Message{
		ID:        generateID(),
		Role:      role,
		Content:   content,
		CreatedAt: time.Now(),
		Metadata:  make(map[string]any),
	}
}

// Clone creates a deep copy of the message.
func Clone(msg *Message) *Message {
	if msg == nil {
		return nil
	}
	cloned := *msg
	if msg.Metadata != nil {
		cloned.Metadata = make(map[string]any, len(msg.Metadata))
		for k, v := range msg.Metadata {
			cloned.Metadata[k] = v
		}
	}
	if len(msg.ToolCalls) > 0 {
		cloned.ToolCalls = make([]ToolCall, len(msg.ToolCalls))
		for i, tc := range msg.ToolCalls {
			cloned.ToolCalls[i] = cloneToolCall(tc)
		}
	}
	return &cloned
}

// CloneMessages copies a slice of messages.
func CloneMessages(msgs []*Message) []*Message {
	if len(msgs) == 0 {
		return nil
	}
	clones := make([]*Message, 0, len(msgs))
	for _, msg := range msgs {
		clones = append(clones, Clone(msg))
	}
	return clones
}

func cloneToolCall(call ToolCall) ToolCall {
	cloned := ToolCall{
		ID:   call.ID,
		Name: call.Name,
	}
	if call.Args != nil {
		cloned.Args = make(map[string]any, len(call.Args))
		for k, v := range call.Args {
			cloned.Args[k] = v
		}
	}
	if call.Response != "" {
		cloned.Response = call.Response
	}
	return cloned
}

// NewToolCallMessage creates a message with tool calls
func NewToolCallMessage(toolCalls []ToolCall) *Message {
	return &Message{
		ID:        generateID(),
		Role:      RoleAssistant,
		ToolCalls: toolCalls,
		CreatedAt: time.Now(),
		Metadata:  make(map[string]any),
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
		Metadata:  make(map[string]any),
	}
}

// generateID generates a unique message ID
func generateID() string {
	// Simple implementation using timestamp
	// In production, consider using UUID
	return time.Now().Format("20060102150405.000000")
}
