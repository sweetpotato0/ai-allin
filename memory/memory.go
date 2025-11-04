package memory

import (
	"context"
	"fmt"
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

// GenerateMemoryID generates a unique ID for a memory entry using current timestamp
func GenerateMemoryID() string {
	return fmt.Sprintf("mem_%d", time.Now().UnixNano())
}

// MemoryStore defines the interface for storing and retrieving memories.
type MemoryStore interface {
	AddMemory(context.Context, *Memory) error
	SearchMemory(context.Context, string) ([]*Memory, error)
}
