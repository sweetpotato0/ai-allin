package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sweetpotato0/ai-allin/agent"
)

// Store defines the interface for session storage backends that operate on
// serializable session records.
type Store interface {
	Save(ctx context.Context, record *Record) error
	Load(ctx context.Context, id string) (*Record, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]string, error)
	Count(ctx context.Context) (int, error)
	Exists(ctx context.Context, id string) (bool, error)
}

// AgentResolver resolves the agent prototype for a persisted session that is
// being rehydrated from the store.
type AgentResolver func(sessionID string, record *Record) (*agent.Agent, error)

// Manager manages multiple sessions using a storage backend.
type Manager struct {
	mu            sync.RWMutex
	store         Store
	resolver      AgentResolver
	sessions      map[string]Session
	sessionAgents map[string]*agent.Agent
}

// Option is a function that configures a Manager.
type Option func(*Manager)

// WithStore sets the store for the manager.
func WithStore(s Store) Option {
	return func(m *Manager) {
		m.store = s
	}
}

// WithAgentResolver sets a custom resolver used when rehydrating single-agent
// sessions from persisted records.
func WithAgentResolver(resolver AgentResolver) Option {
	return func(m *Manager) {
		m.resolver = resolver
	}
}

// NewManager creates a new session manager with the given options.
//
// Example:
//
//	mgr := session.NewManager(session.WithStore(store.NewInMemoryStore()))
func NewManager(opts ...Option) *Manager {
	m := &Manager{
		sessions:      make(map[string]Session),
		sessionAgents: make(map[string]*agent.Agent),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// NewManagerWithStore creates a new session manager with a custom store.
// Deprecated: Use NewManager(WithStore(store)) instead.
func NewManagerWithStore(s Store) *Manager {
	return NewManager(WithStore(s))
}

// Create creates a new single-agent session.
func (m *Manager) Create(ctx context.Context, id string, ag *agent.Agent) (*SingleAgentSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.ensureStore(); err != nil {
		return nil, err
	}

	exists, err := m.store.Exists(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to check session existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("session %s already exists", id)
	}

	sess := New(id, ag)
	if err := m.persistLocked(ctx, sess); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	m.storeSessionLocked(sess)
	return sess, nil
}

// CreateShared creates a new shared (multi-agent) session.
func (m *Manager) CreateShared(ctx context.Context, id string) (*SharedSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.ensureStore(); err != nil {
		return nil, err
	}

	exists, err := m.store.Exists(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to check session existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("session %s already exists", id)
	}

	sess := NewShared(id)
	if err := m.persistLocked(ctx, sess); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	m.storeSessionLocked(sess)
	return sess, nil
}

// Get retrieves a session by ID.
func (m *Manager) Get(ctx context.Context, id string) (Session, error) {
	if sess, ok := m.getCached(id); ok {
		return sess, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if sess, ok := m.sessions[id]; ok {
		return sess, nil
	}

	if err := m.ensureStore(); err != nil {
		return nil, err
	}

	record, err := m.store.Load(ctx, id)
	if err != nil {
		return nil, err
	}

	sess, err := m.instantiate(record)
	if err != nil {
		return nil, err
	}

	m.storeSessionLocked(sess)
	return sess, nil
}

// GetOrCreate retrieves a session by ID or creates a new single-agent session if it doesn't exist.
func (m *Manager) GetOrCreate(ctx context.Context, id string, ag *agent.Agent) (*SingleAgentSession, error) {
	if sess, ok := m.getCached(id); ok {
		if single, ok := sess.(*SingleAgentSession); ok {
			return single, nil
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if sess, ok := m.sessions[id]; ok {
		if single, ok := sess.(*SingleAgentSession); ok {
			return single, nil
		}
	}

	if err := m.ensureStore(); err != nil {
		return nil, err
	}

	record, err := m.store.Load(ctx, id)
	if err == nil && record != nil {
		if record.Type == TypeSingleAgent {
			sess, err := m.instantiate(record)
			if err != nil {
				return nil, err
			}
			if single, ok := sess.(*SingleAgentSession); ok {
				m.storeSessionLocked(single)
				return single, nil
			}
		}

		newID := id + "_single"
		s := New(newID, ag)
		if err := m.persistLocked(ctx, s); err != nil {
			return nil, fmt.Errorf("failed to save session: %w", err)
		}
		m.storeSessionLocked(s)
		return s, nil
	}

	s := New(id, ag)
	if err := m.persistLocked(ctx, s); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}
	m.storeSessionLocked(s)
	return s, nil
}

// GetOrCreateShared retrieves a session by ID or creates a new shared session if it doesn't exist.
func (m *Manager) GetOrCreateShared(ctx context.Context, id string) (*SharedSession, error) {
	if sess, ok := m.getCached(id); ok {
		if shared, ok := sess.(*SharedSession); ok {
			return shared, nil
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if sess, ok := m.sessions[id]; ok {
		if shared, ok := sess.(*SharedSession); ok {
			return shared, nil
		}
	}

	if err := m.ensureStore(); err != nil {
		return nil, err
	}

	record, err := m.store.Load(ctx, id)
	if err == nil && record != nil {
		if record.Type == TypeShared {
			sess, err := m.instantiate(record)
			if err != nil {
				return nil, err
			}
			if shared, ok := sess.(*SharedSession); ok {
				m.storeSessionLocked(shared)
				return shared, nil
			}
		}

		newID := id + "_shared"
		s := NewShared(newID)
		if err := m.persistLocked(ctx, s); err != nil {
			return nil, fmt.Errorf("failed to save session: %w", err)
		}
		m.storeSessionLocked(s)
		return s, nil
	}

	s := NewShared(id)
	if err := m.persistLocked(ctx, s); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}
	m.storeSessionLocked(s)
	return s, nil
}

// Delete removes a session.
func (m *Manager) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sess, ok := m.sessions[id]; ok {
		_ = sess.Close()
	}
	delete(m.sessions, id)
	delete(m.sessionAgents, id)

	if err := m.ensureStore(); err != nil {
		return err
	}
	return m.store.Delete(ctx, id)
}

// List returns all session IDs.
func (m *Manager) List(ctx context.Context) ([]string, error) {
	if err := m.ensureStore(); err != nil {
		return nil, err
	}
	return m.store.List(ctx)
}

// Count returns the number of active sessions.
func (m *Manager) Count(ctx context.Context) (int, error) {
	if err := m.ensureStore(); err != nil {
		return 0, err
	}
	return m.store.Count(ctx)
}

// CleanupInactive removes inactive sessions older than the specified duration.
func (m *Manager) CleanupInactive(ctx context.Context, _ time.Duration) (int, error) {
	if err := m.ensureStore(); err != nil {
		return 0, err
	}

	ids, err := m.store.List(ctx)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, id := range ids {
		record, err := m.store.Load(ctx, id)
		if err != nil {
			continue
		}
		if record.State == StateInactive {
			if sess, ok := m.sessions[id]; ok {
				_ = sess.Close()
			}
			if err := m.store.Delete(ctx, id); err == nil {
				count++
				delete(m.sessions, id)
				delete(m.sessionAgents, id)
			}
		}
	}
	return count, nil
}

// Save saves a session to the store.
func (m *Manager) Save(ctx context.Context, sess Session) error {
	if err := m.ensureStore(); err != nil {
		return err
	}
	return m.store.Save(ctx, sess.Snapshot())
}

func (m *Manager) persistLocked(ctx context.Context, sess Session) error {
	if err := m.ensureStore(); err != nil {
		return err
	}
	return m.store.Save(ctx, sess.Snapshot())
}

func (m *Manager) ensureStore() error {
	if m.store == nil {
		return fmt.Errorf("session manager store is not configured")
	}
	return nil
}

func (m *Manager) instantiate(record *Record) (Session, error) {
	if record == nil {
		return nil, fmt.Errorf("session record is nil")
	}

	switch record.Type {
	case TypeSingleAgent:
		ag := m.sessionAgents[record.ID]
		if ag == nil && m.resolver != nil {
			var err error
			ag, err = m.resolver(record.ID, record)
			if err != nil {
				return nil, err
			}
		}
		if ag == nil {
			return nil, fmt.Errorf("no agent registered for session %s", record.ID)
		}
		return NewSingleFromRecord(record, ag), nil
	case TypeShared:
		return NewSharedFromRecord(record), nil
	default:
		return nil, fmt.Errorf("unknown session type %s", record.Type)
	}
}

func (m *Manager) getCached(id string) (Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sess, ok := m.sessions[id]
	return sess, ok
}

func (m *Manager) storeSessionLocked(sess Session) {
	if sess == nil {
		return
	}
	m.sessions[sess.ID()] = sess
	if single, ok := sess.(*SingleAgentSession); ok {
		m.sessionAgents[single.ID()] = single.Agent()
	}
}
