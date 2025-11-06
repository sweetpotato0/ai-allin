package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/message"
)

// SharedSession represents a session that can be used by multiple agents.
// It maintains shared conversation history that can be replayed across agents.
type SharedSession struct {
	Base
	mu       sync.RWMutex
	messages []*message.Message
}

// NewShared creates a new shared session
func NewShared(id string) *SharedSession {
	return &SharedSession{
		Base:     NewBase(id),
		messages: make([]*message.Message, 0),
	}
}

// Run executes the agent with input (implements Session interface)
// For shared sessions, use RunWithAgent instead
func (s *SharedSession) Run(ctx context.Context, input string) (string, error) {
	return "", fmt.Errorf("shared session %s requires an agent, use RunWithAgent instead", s.ID())
}

// RunWithAgent replays the conversation into the provided agent and captures the result.
// It creates a clone of the agent to avoid modifying the original, replays the conversation
// history, executes the agent with the input, and updates the conversation history.
func (s *SharedSession) RunWithAgent(ctx context.Context, ag *agent.Agent, input string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State != StateActive {
		return "", fmt.Errorf("session %s is not active", s.ID())
	}

	// Clone agent to avoid modifying the original
	cloned := ag.Clone()

	// Restore conversation history to the cloned agent
	cloned.ClearMessages()
	for _, msg := range s.messages {
		cloned.AddMessage(message.Clone(msg))
	}

	// Execute the agent with the input
	response, err := cloned.Run(ctx, input)
	if err != nil {
		return "", fmt.Errorf("agent execution failed: %w", err)
	}

	// Update conversation history with all messages from the cloned agent
	s.messages = message.CloneMessages(cloned.GetMessages())
	s.UpdatedAt = time.Now()

	return response, nil
}

// GetMessages returns all messages in the session (implements Session interface)
func (s *SharedSession) GetMessages() []*message.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return message.CloneMessages(s.messages)
}

// GetState returns the current session state
func (s *SharedSession) GetState() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State
}

// Close closes the session (implements Session interface)
func (s *SharedSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.State == StateClosed {
		return fmt.Errorf("session already closed")
	}
	s.SetState(StateClosed)
	return nil
}
