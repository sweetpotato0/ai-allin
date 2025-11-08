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

// Type represents the concrete implementation for a session.
type Type string

const (
	TypeSingleAgent Type = "single_agent"
	TypeShared      Type = "shared"
)

// Record captures a serializable snapshot for a session.
type Record struct {
	ID           string             `json:"id"`
	Type         Type               `json:"type"`
	State        State              `json:"state"`
	Messages     []*message.Message `json:"messages"`
	LastMessage  *message.Message   `json:"last_message,omitempty"`
	LastDuration time.Duration      `json:"last_duration,omitempty"`
	Metadata     map[string]any     `json:"metadata"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

// Clone returns a deep copy of the record to prevent accidental mutation.
func (r *Record) Clone() *Record {
	if r == nil {
		return nil
	}
	clone := *r
	clone.Messages = message.CloneMessages(r.Messages)
	clone.LastMessage = message.Clone(r.LastMessage)
	clone.Metadata = cloneMetadata(r.Metadata)
	return &clone
}

// Session represents a conversation session with an agent.
type Session interface {
	// ID returns the session ID
	ID() string

	// Type returns the concrete session type
	Type() Type

	// Run executes the agent with input
	Run(ctx context.Context, input string) (string, error)

	// GetMessages returns all messages in the session
	GetMessages() []*message.Message

	// GetState returns the current session state
	GetState() State

	// Close closes the session
	Close() error

	// Snapshot returns a serializable record of the session state
	Snapshot() *Record
}

// Base provides common fields and methods for session implementations
type Base struct {
	id           string
	sessionType  Type
	State        State
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Metadata     map[string]any
	messages     []*message.Message
	lastMessage  *message.Message
	lastDuration time.Duration
}

// NewBase initializes a new base session
func NewBase(id string, sessionType Type) Base {
	return Base{
		id:          id,
		sessionType: sessionType,
		State:       StateActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    make(map[string]any),
		messages:    make([]*message.Message, 0),
	}
}

// ID returns the session ID
func (b *Base) ID() string {
	return b.id
}

// Type returns the concrete session type.
func (b *Base) Type() Type {
	return b.sessionType
}

// SetState updates the session state
func (b *Base) SetState(state State) {
	b.State = state
	b.touch()
}

// SetMetadata sets metadata for the session
func (b *Base) SetMetadata(key string, value any) {
	if b.Metadata == nil {
		b.Metadata = make(map[string]any)
	}
	b.Metadata[key] = value
	b.touch()
}

// GetMetadata returns metadata for the session
func (b *Base) GetMetadata(key string) (any, bool) {
	if b.Metadata == nil {
		return nil, false
	}
	value, ok := b.Metadata[key]
	return value, ok
}

// Messages returns a copy of the tracked messages.
func (b *Base) Messages() []*message.Message {
	return message.CloneMessages(b.messages)
}

// SetMessages replaces the tracked messages with a cloned slice.
func (b *Base) SetMessages(msgs []*message.Message) {
	b.messages = message.CloneMessages(msgs)
	b.touch()
}

// SetLastMessage stores the last assistant message returned by the runtime.
func (b *Base) SetLastMessage(msg *message.Message) {
	if msg == nil {
		b.lastMessage = nil
	} else {
		b.lastMessage = message.Clone(msg)
	}
	b.touch()
}

// SetLastDuration stores the duration of the most recent turn.
func (b *Base) SetLastDuration(d time.Duration) {
	b.lastDuration = d
	b.touch()
}

// Snapshot returns a serializable representation of the base state.
func (b *Base) Snapshot() *Record {
	return &Record{
		ID:           b.id,
		Type:         b.sessionType,
		State:        b.State,
		Messages:     message.CloneMessages(b.messages),
		LastMessage:  message.Clone(b.lastMessage),
		LastDuration: b.lastDuration,
		Metadata:     cloneMetadata(b.Metadata),
		CreatedAt:    b.CreatedAt,
		UpdatedAt:    b.UpdatedAt,
	}
}

func (b *Base) touch() {
	b.UpdatedAt = time.Now()
}

func cloneMetadata(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
