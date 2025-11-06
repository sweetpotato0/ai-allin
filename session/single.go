package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/message"
)

// SingleAgentSession represents a session with a single agent.
type SingleAgentSession struct {
	Base
	mu    sync.RWMutex
	agent *agent.Agent
}

// New creates a new session with a single agent
func New(id string, ag *agent.Agent) *SingleAgentSession {
	return &SingleAgentSession{
		Base:  NewBase(id),
		agent: ag,
	}
}

// Run executes the agent with input
func (s *SingleAgentSession) Run(ctx context.Context, input string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State != StateActive {
		return "", fmt.Errorf("session is not active (state: %s)", s.State)
	}

	s.UpdatedAt = time.Now()
	return s.agent.Run(ctx, input)
}

// GetMessages returns all messages in the session
func (s *SingleAgentSession) GetMessages() []*message.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.agent.GetMessages()
}

// GetState returns the current session state
func (s *SingleAgentSession) GetState() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State
}

// Close closes the session
func (s *SingleAgentSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State == StateClosed {
		return fmt.Errorf("session already closed")
	}

	s.SetState(StateClosed)
	return nil
}

// Agent returns the agent associated with this session
func (s *SingleAgentSession) Agent() *agent.Agent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.agent
}
