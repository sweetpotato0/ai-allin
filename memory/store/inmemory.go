package store

import (
	"context"
	"fmt"
	"sync"

	"github.com/sweetpotato0/ai-allin/memory"
)

// InMemoryStore implements MemoryStore using in-memory storage
type InMemoryStore struct {
	memories []*memory.Memory
	mu       sync.RWMutex
}

// NewInMemoryStore creates a new in-memory memory store
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		memories: make([]*memory.Memory, 0),
	}
}

// AddMemory adds a memory to the store
func (s *InMemoryStore) AddMemory(ctx context.Context, mem *memory.Memory) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if mem == nil {
		return fmt.Errorf("memory cannot be nil")
	}

	s.memories = append(s.memories, mem)
	return nil
}

// SearchMemory searches for memories matching the query
func (s *InMemoryStore) SearchMemory(ctx context.Context, query string) ([]*memory.Memory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Simple implementation: return all memories
	// In a real implementation, this would use semantic search, vector similarity, etc.
	if query == "" {
		return s.memories, nil
	}

	// Return all memories for now
	// TODO: Implement proper search logic
	return s.memories, nil
}

// Clear removes all memories from the store
func (s *InMemoryStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.memories = make([]*memory.Memory, 0)
	return nil
}

// Count returns the number of memories in the store
func (s *InMemoryStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.memories)
}
