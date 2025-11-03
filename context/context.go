package context

import (
	"github.com/sweetpotato0/ai-allin/message"
)

// Context manages the conversation context including message history
type Context struct {
	messages []*message.Message
	maxSize  int // Maximum number of messages to keep
}

// New creates a new context with default settings
func New() *Context {
	return &Context{
		messages: make([]*message.Message, 0),
		maxSize:  100, // Default max size
	}
}

// NewWithMaxSize creates a new context with specified max size
func NewWithMaxSize(maxSize int) *Context {
	return &Context{
		messages: make([]*message.Message, 0),
		maxSize:  maxSize,
	}
}

// AddMessage adds a message to the context
func (c *Context) AddMessage(msg *message.Message) {
	c.messages = append(c.messages, msg)

	// Trim old messages if exceeds max size
	if len(c.messages) > c.maxSize {
		// Keep system messages and recent messages
		systemMsgs := make([]*message.Message, 0)
		for _, m := range c.messages {
			if m.Role == message.RoleSystem {
				systemMsgs = append(systemMsgs, m)
			}
		}

		// Calculate how many non-system messages to keep
		keepCount := c.maxSize - len(systemMsgs)
		recentMsgs := c.messages[len(c.messages)-keepCount:]

		// Rebuild messages: system messages + recent messages
		newMessages := make([]*message.Message, 0, c.maxSize)
		newMessages = append(newMessages, systemMsgs...)
		for _, m := range recentMsgs {
			if m.Role != message.RoleSystem {
				newMessages = append(newMessages, m)
			}
		}
		c.messages = newMessages
	}
}

// GetMessages returns all messages in the context
func (c *Context) GetMessages() []*message.Message {
	return c.messages
}

// GetLastMessage returns the last message or nil if empty
func (c *Context) GetLastMessage() *message.Message {
	if len(c.messages) == 0 {
		return nil
	}
	return c.messages[len(c.messages)-1]
}

// GetMessagesByRole returns all messages with the specified role
func (c *Context) GetMessagesByRole(role message.Role) []*message.Message {
	result := make([]*message.Message, 0)
	for _, msg := range c.messages {
		if msg.Role == role {
			result = append(result, msg)
		}
	}
	return result
}

// Clear removes all messages from the context
func (c *Context) Clear() {
	c.messages = make([]*message.Message, 0)
}

// Size returns the current number of messages
func (c *Context) Size() int {
	return len(c.messages)
}
