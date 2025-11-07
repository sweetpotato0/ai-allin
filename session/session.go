package session

import (
	"context"
	"time"

	"github.com/sweetpotato0/ai-allin/message"
)

// State represents the state of a session
type State string

const (
	StateActive   State = "active"
	StateInactive State = "inactive"
	StateClosed   State = "closed"
)

// Session represents a conversation session with an agent.
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

// Base provides common fields and methods for session implementations
type Base struct {
	id        string
	State     State
	CreatedAt time.Time
	UpdatedAt time.Time
	Metadata  map[string]any
}

// NewBase initializes a new base session
func NewBase(id string) Base {
	return Base{
		id:        id,
		State:     StateActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  make(map[string]any),
	}
}

// ID returns the session ID
func (b *Base) ID() string {
	return b.id
}

// SetState updates the session state
func (b *Base) SetState(state State) {
	b.State = state
	b.UpdatedAt = time.Now()
}

// SetMetadata sets metadata for the session
func (b *Base) SetMetadata(key string, value any) {
	if b.Metadata == nil {
		b.Metadata = make(map[string]any)
	}
	b.Metadata[key] = value
	b.UpdatedAt = time.Now()
}

// GetMetadata returns metadata for the session
func (b *Base) GetMetadata(key string) (any, bool) {
	if b.Metadata == nil {
		return nil, false
	}
	value, ok := b.Metadata[key]
	return value, ok
}
