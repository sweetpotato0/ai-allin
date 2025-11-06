package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sweetpotato0/ai-allin/agent"
)

// Store defines the interface for session storage backends
// This interface is defined here to avoid circular imports.
// Implementations are in session/store package.
type Store interface {
	// Save saves a session to the store
	Save(ctx context.Context, sess Session) error

	// Load loads a session from the store
	Load(ctx context.Context, id string) (Session, error)

	// Delete removes a session from the store
	Delete(ctx context.Context, id string) error

	// List returns all session IDs in the store
	List(ctx context.Context) ([]string, error)

	// Count returns the number of sessions in the store
	Count(ctx context.Context) (int, error)

	// Exists checks if a session exists in the store
	Exists(ctx context.Context, id string) (bool, error)
}

// Manager manages multiple sessions using a storage backend
type Manager struct {
	mu    sync.RWMutex
	store Store
}

// Option is a function that configures a Manager
type Option func(*Manager)

// WithStore sets the store for the manager
func WithStore(s Store) Option {
	return func(m *Manager) {
		m.store = s
	}
}

// NewManager creates a new session manager with the given options
// Example:
//
//	mgr := session.NewManager(session.WithStore(store.NewInMemoryStore()))
func NewManager(opts ...Option) *Manager {
	m := &Manager{}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// NewManagerWithStore creates a new session manager with a custom store
// Deprecated: Use NewManager(WithStore(store)) instead
func NewManagerWithStore(s Store) *Manager {
	return NewManager(WithStore(s))
}

// Create creates a new single-agent session
func (m *Manager) Create(ctx context.Context, id string, ag *agent.Agent) (*SingleAgentSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if session already exists
	exists, err := m.store.Exists(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to check session existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("session %s already exists", id)
	}

	sess := New(id, ag)
	if err := m.store.Save(ctx, sess); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return sess, nil
}

// CreateShared creates a new shared (multi-agent) session
func (m *Manager) CreateShared(ctx context.Context, id string) (*SharedSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if session already exists
	exists, err := m.store.Exists(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to check session existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("session %s already exists", id)
	}

	sess := NewShared(id)
	if err := m.store.Save(ctx, sess); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return sess, nil
}

// Get retrieves a session by ID
func (m *Manager) Get(ctx context.Context, id string) (Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sess, err := m.store.Load(ctx, id)
	if err != nil {
		return nil, err
	}

	return sess, nil
}

// GetOrCreate retrieves a session by ID or creates a new single-agent session if it doesn't exist
func (m *Manager) GetOrCreate(ctx context.Context, id string, ag *agent.Agent) (*SingleAgentSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Try to get existing session
	sess, err := m.store.Load(ctx, id)
	if err == nil {
		if s, ok := sess.(*SingleAgentSession); ok {
			return s, nil
		}
		// If it's a shared session, create a new one with a different ID
		newID := id + "_single"
		s := New(newID, ag)
		if err := m.store.Save(ctx, s); err != nil {
			return nil, fmt.Errorf("failed to save session: %w", err)
		}
		return s, nil
	}

	// Create new session
	s := New(id, ag)
	if err := m.store.Save(ctx, s); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return s, nil
}

// GetOrCreateShared retrieves a session by ID or creates a new shared session if it doesn't exist
func (m *Manager) GetOrCreateShared(ctx context.Context, id string) (*SharedSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Try to get existing session
	sess, err := m.store.Load(ctx, id)
	if err == nil {
		if s, ok := sess.(*SharedSession); ok {
			return s, nil
		}
		// If it's a single-agent session, create a new one with a different ID
		newID := id + "_shared"
		s := NewShared(newID)
		if err := m.store.Save(ctx, s); err != nil {
			return nil, fmt.Errorf("failed to save session: %w", err)
		}
		return s, nil
	}

	// Create new session
	s := NewShared(id)
	if err := m.store.Save(ctx, s); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return s, nil
}

// Delete removes a session
func (m *Manager) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Load session to close it first
	sess, err := m.store.Load(ctx, id)
	if err == nil {
		if err := sess.Close(); err != nil {
			// Log error but continue with deletion
		}
	}

	return m.store.Delete(ctx, id)
}

// List returns all session IDs
func (m *Manager) List(ctx context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.store.List(ctx)
}

// Count returns the number of active sessions
func (m *Manager) Count(ctx context.Context) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.store.Count(ctx)
}

// CleanupInactive removes inactive sessions older than the specified duration
func (m *Manager) CleanupInactive(ctx context.Context, maxAge time.Duration) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ids, err := m.store.List(ctx)
	if err != nil {
		return 0, err
	}

	count := 0
	now := time.Now()
	for _, id := range ids {
		sess, err := m.store.Load(ctx, id)
		if err != nil {
			continue
		}

		if sess.GetState() == StateInactive {
			// Check if session is old enough to cleanup
			// This would require additional metadata tracking
			// For now, just mark it for cleanup
			sess.Close()
			if err := m.store.Delete(ctx, id); err == nil {
				count++
			}
		}
	}
	_ = now // Placeholder for future age-based cleanup
	return count, nil
}

// Save saves a session to the store
func (m *Manager) Save(ctx context.Context, sess Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.store.Save(ctx, sess)
}
