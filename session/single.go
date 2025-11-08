package session

import (
	"context"
	"fmt"
	"sync"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/message"
	"github.com/sweetpotato0/ai-allin/runtime"
)

// SingleAgentSession represents a session with a single agent.
// Conversation history is stored on the session itself so it can be snapshotted
// or persisted independently of the underlying agent implementation.
type SingleAgentSession struct {
	Base
	mu        sync.RWMutex
	prototype *agent.Agent
	executor  runtime.Executor
}

// New creates a new single-agent session backed by the provided prototype agent.
func New(id string, ag *agent.Agent) *SingleAgentSession {
	if ag == nil {
		panic("session: agent cannot be nil")
	}

	sess := &SingleAgentSession{
		Base:      NewBase(id, TypeSingleAgent),
		prototype: ag,
		executor:  runtime.NewAgentExecutor(ag),
	}
	sess.Base.SetMessages(ag.GetMessages())
	return sess
}

// NewSingleFromRecord rehydrates a single-agent session from a serialized record.
func NewSingleFromRecord(record *Record, ag *agent.Agent) *SingleAgentSession {
	if record == nil {
		return nil
	}

	sess := &SingleAgentSession{
		Base: Base{
			id:          record.ID,
			sessionType: TypeSingleAgent,
			State:       record.State,
			CreatedAt:   record.CreatedAt,
			UpdatedAt:   record.UpdatedAt,
			Metadata:    cloneMetadata(record.Metadata),
		},
		prototype: ag,
		executor:  runtime.NewAgentExecutor(ag),
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

// Run executes the agent with input and persists the conversation history.
func (s *SingleAgentSession) Run(ctx context.Context, input string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State != StateActive {
		return "", fmt.Errorf("session is not active (state: %s)", s.State)
	}

	if s.executor == nil {
		s.executor = runtime.NewAgentExecutor(s.prototype)
	}

	result, err := s.executor.Execute(ctx, &runtime.Request{
		SessionID: s.ID(),
		Input:     input,
		History:   s.Base.Messages(),
	})
	if err != nil {
		return "", err
	}

	s.Base.SetMessages(result.Messages)
	if result.LastMessage != nil {
		s.Base.SetLastMessage(result.LastMessage)
	} else {
		s.Base.SetLastMessage(nil)
	}
	s.Base.SetLastDuration(result.Duration)
	return result.Output, nil
}

// GetMessages returns all messages in the session
func (s *SingleAgentSession) GetMessages() []*message.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Base.Messages()
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

// Snapshot returns a serializable record of the session.
func (s *SingleAgentSession) Snapshot() *Record {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Base.Snapshot()
}

// Agent returns the agent prototype associated with this session
func (s *SingleAgentSession) Agent() *agent.Agent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.prototype
}
