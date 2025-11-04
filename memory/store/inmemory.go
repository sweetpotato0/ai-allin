package store

import (
	"context"
	"fmt"
	"strings"
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

	// If query is empty, return all memories
	if query == "" {
		// Return a copy sorted by creation time (newest first)
		results := make([]*memory.Memory, len(s.memories))
		copy(results, s.memories)
		// Sort by CreatedAt descending
		for i := 0; i < len(results)-1; i++ {
			for j := i + 1; j < len(results); j++ {
				if results[j].CreatedAt.After(results[i].CreatedAt) {
					results[i], results[j] = results[j], results[i]
				}
			}
		}
		return results, nil
	}

	// Search memories by content (case-insensitive substring match)
	results := make([]*memory.Memory, 0)
	lowerQuery := strings.ToLower(query)

	for _, mem := range s.memories {
		// Search in content and ID
		if strings.Contains(strings.ToLower(mem.Content), lowerQuery) ||
			strings.Contains(strings.ToLower(mem.ID), lowerQuery) {
			results = append(results, mem)
		}
	}

	// Sort results by CreatedAt descending (newest first)
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].CreatedAt.After(results[i].CreatedAt) {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results, nil
}

// Clear removes all memories from the store
func (s *InMemoryStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.memories = make([]*memory.Memory, 0)
	return nil
}

// Count returns the number of memories in the store
func (s *InMemoryStore) Count(ctx context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.memories), nil
}
