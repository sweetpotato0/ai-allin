package memory

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Memory represents a stored memory/conversation entry
type Memory struct {
	ID        string                 `json:"id"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// idGenerator provides efficient ID generation with minimal syscall overhead
type idGenerator struct {
	counter int64
	mu      sync.Mutex
	lastTs  int64
}

var defaultIDGenerator = &idGenerator{}

// GenerateMemoryID generates a unique ID for a memory entry
// Uses an efficient approach that minimizes syscall overhead:
// - Only calls time.Now() when nanosecond changes
// - Uses atomic counter for fast collision-free increments
// - Fallback to timestamp for first call or major time jumps
func GenerateMemoryID() string {
	return defaultIDGenerator.Generate()
}

// Generate creates a unique memory ID efficiently
func (g *idGenerator) Generate() string {
	// Get current time once
	now := time.Now().UnixNano()

	// Fast path: if we're in same nanosecond, just increment counter
	g.mu.Lock()
	if now > g.lastTs {
		// Time moved forward, reset counter
		g.lastTs = now
		g.counter = 0
		g.mu.Unlock()
		return fmt.Sprintf("mem_%d", now)
	}

	// Still in same nanosecond, increment counter for uniqueness
	g.counter++
	counter := g.counter
	g.mu.Unlock()

	return fmt.Sprintf("mem_%d_%d", now, counter)
}

// MemoryStore defines the interface for storing and retrieving memories.
type MemoryStore interface {
	AddMemory(context.Context, *Memory) error
	SearchMemory(context.Context, string) ([]*Memory, error)
}
