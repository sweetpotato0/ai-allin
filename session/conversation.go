package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/message"
)

// Conversation maintains shared history that can be replayed across agents.
type Conversation struct {
	id        string
	mu        sync.Mutex
	state     State
	createdAt time.Time
	updatedAt time.Time
	messages  []*message.Message
}

// NewConversation creates a conversation with the supplied identifier.
func NewConversation(id string) *Conversation {
	return &Conversation{
		id:        id,
		state:     StateActive,
		createdAt: time.Now(),
		updatedAt: time.Now(),
	}
}

// ID returns the conversation identifier.
func (c *Conversation) ID() string {
	return c.id
}

// State returns the conversation state.
func (c *Conversation) State() State {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

// Messages returns a copy of the conversation history.
func (c *Conversation) Messages() []*message.Message {
	c.mu.Lock()
	defer c.mu.Unlock()
	return message.CloneMessages(c.messages)
}

// RunAgent replays the conversation into the provided agent and captures the result.
func (c *Conversation) RunAgent(ctx context.Context, ag *agent.Agent, input string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state != StateActive {
		return "", fmt.Errorf("conversation %s is not active", c.id)
	}

	cloned := ag.Clone()
	cloned.ClearMessages()
	for _, msg := range c.messages {
		cloned.AddMessage(message.Clone(msg))
	}

	response, err := cloned.Run(ctx, input)
	if err != nil {
		return "", err
	}

	c.messages = message.CloneMessages(cloned.GetMessages())
	c.updatedAt = time.Now()
	return response, nil
}

// Close marks the conversation as closed.
func (c *Conversation) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state != StateClosed {
		c.state = StateClosed
		c.updatedAt = time.Now()
	}
}

// ConversationManager stores multiple shared conversations.
type ConversationManager struct {
	mu            sync.RWMutex
	conversations map[string]*Conversation
}

// NewConversationManager constructs a manager.
func NewConversationManager() *ConversationManager {
	return &ConversationManager{
		conversations: make(map[string]*Conversation),
	}
}

// GetOrCreate returns an existing conversation or creates one lazily.
func (m *ConversationManager) GetOrCreate(id string) *Conversation {
	m.mu.Lock()
	defer m.mu.Unlock()
	if conv, ok := m.conversations[id]; ok {
		return conv
	}
	conv := NewConversation(id)
	m.conversations[id] = conv
	return conv
}

// Delete removes a conversation.
func (m *ConversationManager) Delete(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if conv, ok := m.conversations[id]; ok {
		conv.Close()
		delete(m.conversations, id)
	}
}
