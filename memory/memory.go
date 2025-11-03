package memory

import "context"

type Memory struct{}

// MemoryStore defines the interface for storing and retrieving memories.
type MemoryStore interface {
	AddMemory(context.Context, *Memory) error
	SearchMemory(context.Context, string) ([]*Memory, error)
}
