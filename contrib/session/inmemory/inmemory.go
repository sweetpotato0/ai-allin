package inmemory

import (
	"context"
	"fmt"
	"sync"

	"github.com/sweetpotato0/ai-allin/session"
)

// InMemoryStore implements session storage using in-memory storage
type InMemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*session.Record
}

// NewInMemoryStore creates a new in-memory session store
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		sessions: make(map[string]*session.Record),
	}
}

// Save saves a session to the store
func (s *InMemoryStore) Save(ctx context.Context, record *session.Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if record == nil || record.ID == "" {
		return fmt.Errorf("session record cannot be nil")
	}

	s.sessions[record.ID] = record.Clone()
	return nil
}

// Load loads a session from the store
func (s *InMemoryStore) Load(ctx context.Context, id string) (*session.Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sess, exists := s.sessions[id]
	if !exists {
		return nil, fmt.Errorf("session %s not found", id)
	}

	return sess.Clone(), nil
}

// Delete removes a session from the store
func (s *InMemoryStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[id]; !exists {
		return fmt.Errorf("session %s not found", id)
	}

	delete(s.sessions, id)
	return nil
}

// List returns all session IDs in the store
func (s *InMemoryStore) List(ctx context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]string, 0, len(s.sessions))
	for id := range s.sessions {
		ids = append(ids, id)
	}
	return ids, nil
}

// Count returns the number of sessions in the store
func (s *InMemoryStore) Count(ctx context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions), nil
}

// Exists checks if a session exists in the store
func (s *InMemoryStore) Exists(ctx context.Context, id string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.sessions[id]
	return exists, nil
}

// Clear removes all sessions from the store
func (s *InMemoryStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions = make(map[string]*session.Record)
	return nil
}
