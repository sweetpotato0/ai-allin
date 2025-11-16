package session

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/pkg/logging"
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
	logger        *slog.Logger
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

// WithLogger overrides the logger used by the manager.
func WithLogger(logger *slog.Logger) Option {
	return func(m *Manager) {
		if logger != nil {
			m.logger = logger
		}
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
	if m.logger == nil {
		m.logger = logging.WithComponent("session_manager")
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
	if m.logger != nil {
		m.logger.Info("creating single-agent session", "id", id)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.ensureStore(); err != nil {
		if m.logger != nil {
			m.logger.Error("create session ensure store failed", "id", id, "error", err)
		}
		return nil, err
	}

	exists, err := m.store.Exists(ctx, id)
	if err != nil {
		if m.logger != nil {
			m.logger.Error("create session existence check failed", "id", id, "error", err)
		}
		return nil, fmt.Errorf("failed to check session existence: %w", err)
	}
	if exists {
		if m.logger != nil {
			m.logger.Warn("create session aborted; already exists", "id", id)
		}
		return nil, fmt.Errorf("session %s already exists", id)
	}

	sess := New(id, ag)
	if err := m.persistLocked(ctx, sess); err != nil {
		if m.logger != nil {
			m.logger.Error("create session persist failed", "id", id, "error", err)
		}
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	m.storeSessionLocked(sess)
	if m.logger != nil {
		m.logger.Info("single-agent session created", "id", id)
	}
	return sess, nil
}

// CreateShared creates a new shared (multi-agent) session.
func (m *Manager) CreateShared(ctx context.Context, id string) (*SharedSession, error) {
	if m.logger != nil {
		m.logger.Info("creating shared session", "id", id)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.ensureStore(); err != nil {
		if m.logger != nil {
			m.logger.Error("create shared ensure store failed", "id", id, "error", err)
		}
		return nil, err
	}

	exists, err := m.store.Exists(ctx, id)
	if err != nil {
		if m.logger != nil {
			m.logger.Error("create shared existence check failed", "id", id, "error", err)
		}
		return nil, fmt.Errorf("failed to check session existence: %w", err)
	}
	if exists {
		if m.logger != nil {
			m.logger.Warn("create shared aborted; already exists", "id", id)
		}
		return nil, fmt.Errorf("session %s already exists", id)
	}

	sess := NewShared(id)
	if err := m.persistLocked(ctx, sess); err != nil {
		if m.logger != nil {
			m.logger.Error("create shared persist failed", "id", id, "error", err)
		}
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	m.storeSessionLocked(sess)
	if m.logger != nil {
		m.logger.Info("shared session created", "id", id)
	}
	return sess, nil
}

// Get retrieves a session by ID.
func (m *Manager) Get(ctx context.Context, id string) (Session, error) {
	if m.logger != nil {
		m.logger.Info("loading session", "id", id)
	}
	if sess, ok := m.getCached(id); ok {
		if m.logger != nil {
			m.logger.Debug("session hit cache", "id", id)
		}
		return sess, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if sess, ok := m.sessions[id]; ok {
		if m.logger != nil {
			m.logger.Debug("session found in memory", "id", id)
		}
		return sess, nil
	}

	if err := m.ensureStore(); err != nil {
		if m.logger != nil {
			m.logger.Error("get session ensure store failed", "id", id, "error", err)
		}
		return nil, err
	}

	record, err := m.store.Load(ctx, id)
	if err != nil {
		if m.logger != nil {
			m.logger.Error("get session load failed", "id", id, "error", err)
		}
		return nil, err
	}

	sess, err := m.instantiate(record)
	if err != nil {
		if m.logger != nil {
			m.logger.Error("get session instantiate failed", "id", id, "error", err)
		}
		return nil, err
	}

	m.storeSessionLocked(sess)
	if m.logger != nil {
		m.logger.Info("session loaded", "id", id)
	}
	return sess, nil
}

// GetOrCreate retrieves a session by ID or creates a new single-agent session if it doesn't exist.
func (m *Manager) GetOrCreate(ctx context.Context, id string, ag *agent.Agent) (*SingleAgentSession, error) {
	if m.logger != nil {
		m.logger.Info("get or create single session", "id", id)
	}
	if sess, ok := m.getCached(id); ok {
		if single, ok := sess.(*SingleAgentSession); ok {
			if m.logger != nil {
				m.logger.Debug("get or create single hit cache", "id", id)
			}
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
		if m.logger != nil {
			m.logger.Error("get or create single ensure store failed", "id", id, "error", err)
		}
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
				if m.logger != nil {
					m.logger.Info("rehydrated existing single session", "id", id)
				}
				return single, nil
			}
		}

		newID := id + "_single"
		s := New(newID, ag)
		if err := m.persistLocked(ctx, s); err != nil {
			if m.logger != nil {
				m.logger.Error("persist fallback single session failed", "id", newID, "error", err)
			}
			return nil, fmt.Errorf("failed to save session: %w", err)
		}
		m.storeSessionLocked(s)
		if m.logger != nil {
			m.logger.Info("created fallback single session", "id", newID)
		}
		return s, nil
	}

	s := New(id, ag)
	if err := m.persistLocked(ctx, s); err != nil {
		if m.logger != nil {
			m.logger.Error("persist new single session failed", "id", id, "error", err)
		}
		return nil, fmt.Errorf("failed to save session: %w", err)
	}
	m.storeSessionLocked(s)
	if m.logger != nil {
		m.logger.Info("created new single session", "id", id)
	}
	return s, nil
}

// GetOrCreateShared retrieves a session by ID or creates a new shared session if it doesn't exist.
func (m *Manager) GetOrCreateShared(ctx context.Context, id string) (*SharedSession, error) {
	if m.logger != nil {
		m.logger.Info("get or create shared session", "id", id)
	}
	if sess, ok := m.getCached(id); ok {
		if shared, ok := sess.(*SharedSession); ok {
			if m.logger != nil {
				m.logger.Debug("get or create shared hit cache", "id", id)
			}
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
		if m.logger != nil {
			m.logger.Error("get or create shared ensure store failed", "id", id, "error", err)
		}
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
				if m.logger != nil {
					m.logger.Info("rehydrated shared session", "id", id)
				}
				return shared, nil
			}
		}

		newID := id + "_shared"
		s := NewShared(newID)
		if err := m.persistLocked(ctx, s); err != nil {
			if m.logger != nil {
				m.logger.Error("persist fallback shared session failed", "id", newID, "error", err)
			}
			return nil, fmt.Errorf("failed to save session: %w", err)
		}
		m.storeSessionLocked(s)
		if m.logger != nil {
			m.logger.Info("created fallback shared session", "id", newID)
		}
		return s, nil
	}

	s := NewShared(id)
	if err := m.persistLocked(ctx, s); err != nil {
		if m.logger != nil {
			m.logger.Error("persist shared session failed", "id", id, "error", err)
		}
		return nil, fmt.Errorf("failed to save session: %w", err)
	}
	m.storeSessionLocked(s)
	if m.logger != nil {
		m.logger.Info("created shared session", "id", id)
	}
	return s, nil
}

// Delete removes a session.
func (m *Manager) Delete(ctx context.Context, id string) error {
	if m.logger != nil {
		m.logger.Warn("deleting session", "id", id)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if sess, ok := m.sessions[id]; ok {
		_ = sess.Close()
	}
	delete(m.sessions, id)
	delete(m.sessionAgents, id)

	if err := m.ensureStore(); err != nil {
		if m.logger != nil {
			m.logger.Error("delete session ensure store failed", "id", id, "error", err)
		}
		return err
	}
	if err := m.store.Delete(ctx, id); err != nil {
		if m.logger != nil {
			m.logger.Error("delete session failed", "id", id, "error", err)
		}
		return err
	}
	if m.logger != nil {
		m.logger.Info("session deleted", "id", id)
	}
	return nil
}

// List returns all session IDs.
func (m *Manager) List(ctx context.Context) ([]string, error) {
	if err := m.ensureStore(); err != nil {
		if m.logger != nil {
			m.logger.Error("list sessions ensure store failed", "error", err)
		}
		return nil, err
	}
	ids, err := m.store.List(ctx)
	if err != nil {
		if m.logger != nil {
			m.logger.Error("list sessions failed", "error", err)
		}
		return nil, err
	}
	if m.logger != nil {
		m.logger.Info("listed sessions", "count", len(ids))
	}
	return ids, nil
}

// Count returns the number of active sessions.
func (m *Manager) Count(ctx context.Context) (int, error) {
	if err := m.ensureStore(); err != nil {
		if m.logger != nil {
			m.logger.Error("count sessions ensure store failed", "error", err)
		}
		return 0, err
	}
	count, err := m.store.Count(ctx)
	if err != nil {
		if m.logger != nil {
			m.logger.Error("count sessions failed", "error", err)
		}
		return 0, err
	}
	if m.logger != nil {
		m.logger.Info("counted sessions", "count", count)
	}
	return count, nil
}

// CleanupInactive removes inactive sessions older than the specified duration.
func (m *Manager) CleanupInactive(ctx context.Context, _ time.Duration) (int, error) {
	if err := m.ensureStore(); err != nil {
		if m.logger != nil {
			m.logger.Error("cleanup inactive ensure store failed", "error", err)
		}
		return 0, err
	}

	ids, err := m.store.List(ctx)
	if err != nil {
		if m.logger != nil {
			m.logger.Error("cleanup inactive list failed", "error", err)
		}
		return 0, err
	}

	count := 0
	for _, id := range ids {
		record, err := m.store.Load(ctx, id)
		if err != nil {
			if m.logger != nil {
				m.logger.Warn("cleanup inactive load failed", "id", id, "error", err)
			}
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
				if m.logger != nil {
					m.logger.Info("cleaned inactive session", "id", id)
				}
			}
		}
	}
	if m.logger != nil {
		m.logger.Info("cleanup inactive completed", "removed", count)
	}
	return count, nil
}

// Save saves a session to the store.
func (m *Manager) Save(ctx context.Context, sess Session) error {
	if err := m.ensureStore(); err != nil {
		if m.logger != nil {
			m.logger.Error("save session ensure store failed", "id", sess.ID(), "error", err)
		}
		return err
	}
	if err := m.store.Save(ctx, sess.Snapshot()); err != nil {
		if m.logger != nil {
			m.logger.Error("save session failed", "id", sess.ID(), "error", err)
		}
		return err
	}
	if m.logger != nil {
		m.logger.Info("session saved", "id", sess.ID())
	}
	return nil
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
