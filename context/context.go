package context

import (
	"sync"

	"github.com/sweetpotato0/ai-allin/message"
)

// Context manages the conversation context including message history
// All operations are thread-safe using RWMutex protection
type Context struct {
	mu       sync.RWMutex // Protects messages and maxSize
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
	c.mu.Lock()
	defer c.mu.Unlock()

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

// GetMessages returns a copy of all messages in the context
func (c *Context) GetMessages() []*message.Message {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]*message.Message, len(c.messages))
	copy(result, c.messages)
	return result
}

// GetLastMessage returns the last message or nil if empty
func (c *Context) GetLastMessage() *message.Message {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.messages) == 0 {
		return nil
	}
	return c.messages[len(c.messages)-1]
}

// GetMessagesByRole returns all messages with the specified role
func (c *Context) GetMessagesByRole(role message.Role) []*message.Message {
	c.mu.RLock()
	defer c.mu.RUnlock()

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
	c.mu.Lock()
	defer c.mu.Unlock()

	c.messages = make([]*message.Message, 0)
}

// Size returns the current number of messages
func (c *Context) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.messages)
}

