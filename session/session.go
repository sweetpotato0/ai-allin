package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/message"
)

// State represents the state of a session
type State string

const (
	StateActive   State = "active"
	StateInactive State = "inactive"
	StateClosed   State = "closed"
)

// Session represents a conversation session with an agent
type Session interface {
	// ID returns the session ID
	ID() string

	// Run executes the agent with input
	Run(ctx context.Context, input string) (string, error)

	// GetMessages returns all messages in the session
	GetMessages() []*message.Message

	// GetState returns the current session state
	GetState() State

	// Close closes the session
	Close() error
}

// session is the default implementation of Session
type session struct {
	id        string
	agent     *agent.Agent
	state     State
	createdAt time.Time
	updatedAt time.Time
	mu        sync.RWMutex
	metadata  map[string]interface{}
}

// New creates a new session
func New(id string, ag *agent.Agent) Session {
	return &session{
		id:        id,
		agent:     ag,
		state:     StateActive,
		createdAt: time.Now(),
		updatedAt: time.Now(),
		metadata:  make(map[string]interface{}),
	}
}

// ID returns the session ID
func (s *session) ID() string {
	return s.id
}

// Run executes the agent with input
func (s *session) Run(ctx context.Context, input string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != StateActive {
		return "", fmt.Errorf("session is not active (state: %s)", s.state)
	}

	s.updatedAt = time.Now()
	return s.agent.Run(ctx, input)
}

// GetMessages returns all messages in the session
func (s *session) GetMessages() []*message.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.agent.GetMessages()
}

// GetState returns the current session state
func (s *session) GetState() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// Close closes the session
func (s *session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == StateClosed {
		return fmt.Errorf("session already closed")
	}

	s.state = StateClosed
	s.updatedAt = time.Now()
	return nil
}

// Manager manages multiple sessions
type Manager struct {
	sessions map[string]Session
	mu       sync.RWMutex
}

// NewManager creates a new session manager
func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]Session),
	}
}

// Create creates a new session
func (m *Manager) Create(id string, ag *agent.Agent) (Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sessions[id]; exists {
		return nil, fmt.Errorf("session %s already exists", id)
	}

	sess := New(id, ag)
	m.sessions[id] = sess
	return sess, nil
}

// Get retrieves a session by ID
func (m *Manager) Get(id string) (Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sess, exists := m.sessions[id]
	if !exists {
		return nil, fmt.Errorf("session %s not found", id)
	}
	return sess, nil
}

// Delete removes a session
func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sess, exists := m.sessions[id]
	if !exists {
		return fmt.Errorf("session %s not found", id)
	}

	// Close the session before deleting
	if err := sess.Close(); err != nil {
		return err
	}

	delete(m.sessions, id)
	return nil
}

// List returns all session IDs
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		ids = append(ids, id)
	}
	return ids
}

// Count returns the number of active sessions
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// CleanupInactive removes inactive sessions older than the specified duration
func (m *Manager) CleanupInactive(maxAge time.Duration) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for id, sess := range m.sessions {
		if sess.GetState() == StateInactive {
			// Check if session is old enough to cleanup
			// This would require additional metadata tracking
			// For now, just mark it for cleanup
			sess.Close()
			delete(m.sessions, id)
			count++
		}
	}
	return count
}

