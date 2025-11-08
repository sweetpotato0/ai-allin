package session

import (
	"context"
	"fmt"
	"sync"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/message"
	"github.com/sweetpotato0/ai-allin/runtime"
)

// SharedSession represents a session that can be used by multiple agents.
// It maintains shared conversation history that can be replayed across agents.
type SharedSession struct {
	Base
	mu sync.RWMutex
}

// NewShared creates a new shared session
func NewShared(id string) *SharedSession {
	return &SharedSession{
		Base: NewBase(id, TypeShared),
	}
}

// NewSharedFromRecord reconstructs a shared session from a snapshot.
func NewSharedFromRecord(record *Record) *SharedSession {
	if record == nil {
		return nil
	}
	sess := &SharedSession{
		Base: Base{
			id:          record.ID,
			sessionType: TypeShared,
			State:       record.State,
			CreatedAt:   record.CreatedAt,
			UpdatedAt:   record.UpdatedAt,
			Metadata:    cloneMetadata(record.Metadata),
		},
	}
	sess.Base.SetMessages(record.Messages)
	if record.LastMessage != nil {
		sess.Base.SetLastMessage(record.LastMessage)
	}
	if record.LastDuration > 0 {
		sess.Base.SetLastDuration(record.LastDuration)
	}
	return sess
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

	executor := runtime.NewAgentExecutor(ag)
	result, err := executor.Execute(ctx, &runtime.Request{
		SessionID: s.ID(),
		Input:     input,
		History:   s.Base.Messages(),
	})
	if err != nil {
		return "", fmt.Errorf("agent execution failed: %w", err)
	}

	// Update conversation history with all messages from the cloned agent
	s.Base.SetMessages(result.Messages)
	if result.LastMessage != nil {
		s.Base.SetLastMessage(result.LastMessage)
	} else {
		s.Base.SetLastMessage(nil)
	}
	s.Base.SetLastDuration(result.Duration)

	return result.Output, nil
}

// GetMessages returns all messages in the session (implements Session interface)
func (s *SharedSession) GetMessages() []*message.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Base.Messages()
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

// Snapshot returns a serializable record of the shared session.
func (s *SharedSession) Snapshot() *Record {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Base.Snapshot()
}
