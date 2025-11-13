package session

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/message"
)

func TestNewSession(t *testing.T) {
	ag := agent.New(agent.WithName("TestAgent"))
	sess := New("sess1", ag)

	if sess.ID() != "sess1" {
		t.Errorf("Expected session ID sess1, got %s", sess.ID())
	}

	if sess.GetState() != StateActive {
		t.Errorf("Expected initial state Active, got %s", sess.GetState())
	}
}

func TestSessionRun(t *testing.T) {
	ag := agent.New(agent.WithName("TestAgent"))
	sess := New("sess1", ag)

	// Add a simple message to the agent
	ag.AddMessage(message.NewMessage(message.RoleUser, "test"))

	state := sess.GetState()
	if state != StateActive {
		t.Errorf("Expected Active state, got %s", state)
	}
}

func TestSessionClose(t *testing.T) {
	ag := agent.New()
	sess := New("sess1", ag)

	// Close the session
	err := sess.Close()
	if err != nil {
		t.Errorf("Failed to close session: %v", err)
	}

	// Check state is closed
	if sess.GetState() != StateClosed {
		t.Errorf("Expected Closed state after close, got %s", sess.GetState())
	}

	// Try to close again (should error)
	err = sess.Close()
	if err == nil {
		t.Errorf("Expected error when closing already-closed session")
	}
}

func TestSessionClosedStateRejection(t *testing.T) {
	ag := agent.New()
	sess := New("sess1", ag)

	// Close the session
	sess.Close()

	// Try to run on closed session (should error)
	_, err := sess.Run(context.Background(), "test input")
	if err == nil {
		t.Errorf("Expected error when running on closed session")
	}
}

func TestSessionGetMessages(t *testing.T) {
	ag := agent.New()
	sess := New("sess1", ag)

	// Add a message to the underlying agent
	ag.AddMessage(message.NewMessage(message.RoleUser, "Test message"))

	messages := sess.GetMessages()
	if len(messages) == 0 {
		t.Errorf("Expected at least one message")
	}
}

func TestManagerCreate(t *testing.T) {
	manager := NewManager(WithStore(newTestStore()))
	ag := agent.New()

	sess, err := manager.Create(context.Background(), "sess1", ag)
	if err != nil {
		t.Errorf("Failed to create session: %v", err)
	}

	if sess.ID() != "sess1" {
		t.Errorf("Expected session ID sess1, got %s", sess.ID())
	}
}

func TestManagerCreateDuplicate(t *testing.T) {
	manager := NewManager(WithStore(newTestStore()))
	ag := agent.New()

	// Create first session
	_, err := manager.Create(context.Background(), "sess1", ag)
	if err != nil {
		t.Errorf("Failed to create first session: %v", err)
	}

	// Try to create duplicate (should error)
	_, err = manager.Create(context.Background(), "sess1", ag)
	if err == nil {
		t.Errorf("Expected error when creating duplicate session")
	}
}

func TestManagerGet(t *testing.T) {
	manager := NewManager(WithStore(newTestStore()))
	ag := agent.New()

	// Create a session
	created, _ := manager.Create(context.Background(), "sess1", ag)

	// Retrieve it
	retrieved, err := manager.Get(context.Background(), "sess1")
	if err != nil {
		t.Errorf("Failed to get session: %v", err)
	}

	if retrieved.ID() != created.ID() {
		t.Errorf("Retrieved session ID mismatch")
	}
}

func TestManagerGetNotFound(t *testing.T) {
	manager := NewManager(WithStore(newTestStore()))

	// Try to get non-existent session (should error)
	_, err := manager.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Errorf("Expected error when getting non-existent session")
	}
}

func TestManagerDelete(t *testing.T) {
	manager := NewManager(WithStore(newTestStore()))
	ag := agent.New()

	// Create a session
	manager.Create(context.Background(), "sess1", ag)

	// Delete it
	err := manager.Delete(context.Background(), "sess1")
	if err != nil {
		t.Errorf("Failed to delete session: %v", err)
	}

	// Try to retrieve (should error)
	_, err = manager.Get(context.Background(), "sess1")
	if err == nil {
		t.Errorf("Expected error when getting deleted session")
	}
}

func TestManagerDeleteNotFound(t *testing.T) {
	manager := NewManager(WithStore(newTestStore()))

	// Try to delete non-existent session (should error)
	err := manager.Delete(context.Background(), "nonexistent")
	if err == nil {
		t.Errorf("Expected error when deleting non-existent session")
	}
}

func TestManagerList(t *testing.T) {
	manager := NewManager(WithStore(newTestStore()))
	ag := agent.New()

	// Create multiple sessions
	manager.Create(context.Background(), "sess1", ag)
	manager.Create(context.Background(), "sess2", ag)
	manager.Create(context.Background(), "sess3", ag)

	// List sessions
	sessions, _ := manager.List(context.Background())
	if len(sessions) != 3 {
		t.Errorf("Expected 3 sessions, got %d", len(sessions))
	}
}

func TestManagerListEmpty(t *testing.T) {
	manager := NewManager(WithStore(newTestStore()))

	sessions, _ := manager.List(context.Background())
	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions for empty manager")
	}
}

func TestManagerCount(t *testing.T) {
	manager := NewManager(WithStore(newTestStore()))
	ag := agent.New()

	if count, _ := manager.Count(context.Background()); count != 0 {
		t.Errorf("Expected 0 sessions initially")
	}

	manager.Create(context.Background(), "sess1", ag)
	if count, _ := manager.Count(context.Background()); count != 1 {
		t.Errorf("Expected 1 session after create")
	}

	manager.Create(context.Background(), "sess2", ag)
	if count, _ := manager.Count(context.Background()); count != 2 {
		t.Errorf("Expected 2 sessions after second create")
	}

	manager.Delete(context.Background(), "sess1")
	if count, _ := manager.Count(context.Background()); count != 1 {
		t.Errorf("Expected 1 session after delete")
	}
}

func TestManagerClear(t *testing.T) {
	manager := NewManager(WithStore(newTestStore()))
	ag := agent.New()

	// Create multiple sessions
	manager.Create(context.Background(), "sess1", ag)
	manager.Create(context.Background(), "sess2", ag)

	if count, _ := manager.Count(context.Background()); count != 2 {
		t.Errorf("Expected 2 sessions before cleanup")
	}

	// Delete all using Delete method
	manager.Delete(context.Background(), "sess1")
	manager.Delete(context.Background(), "sess2")

	if count, _ := manager.Count(context.Background()); count != 0 {
		t.Errorf("Expected 0 sessions after cleanup")
	}
}

func TestSessionStates(t *testing.T) {
	ag := agent.New()
	sess := New("sess1", ag)

	// Check state transitions
	if sess.GetState() != StateActive {
		t.Errorf("Initial state should be Active")
	}

	// Close changes state to closed
	sess.Close()
	if sess.GetState() != StateClosed {
		t.Errorf("State should be Closed after closing")
	}
}

func TestMultipleSessions(t *testing.T) {
	manager := NewManager(WithStore(newTestStore()))
	ag1 := agent.New(agent.WithName("Agent1"))
	ag2 := agent.New(agent.WithName("Agent2"))

	// Create sessions with different agents
	sess1, err := manager.Create(context.Background(), "sess1", ag1)
	if err != nil {
		t.Errorf("Failed to create sess1: %v", err)
	}

	sess2, err := manager.Create(context.Background(), "sess2", ag2)
	if err != nil {
		t.Errorf("Failed to create sess2: %v", err)
	}

	// Verify they are different sessions
	if sess1.ID() == sess2.ID() {
		t.Errorf("Sessions should have different IDs")
	}

	// Verify list contains both
	sessions, _ := manager.List(context.Background())
	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions in list")
	}
}

func TestSingleSessionSnapshotIsolation(t *testing.T) {
	ag := agent.New(agent.WithSystemPrompt("sys"))
	sess := New("sess1", ag)

	sess.Base.SetMessages([]*message.Message{
		message.NewMessage(message.RoleSystem, "sys"),
		message.NewMessage(message.RoleUser, "hello"),
	})

	snapshot := sess.Snapshot()
	snapshot.Messages[1].SetText("mutated")

	messages := sess.GetMessages()
	if messages[1].Text() != "hello" {
		t.Errorf("expected snapshot mutations to not affect session state")
	}
}

func TestSingleSessionNewFromRecord(t *testing.T) {
	record := &Record{
		ID:        "sess-record",
		Type:      TypeSingleAgent,
		State:     StateActive,
		Messages:  []*message.Message{message.NewMessage(message.RoleUser, "hi")},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	ag := agent.New(agent.WithName("rehydrated"))
	sess := NewSingleFromRecord(record, ag)

	if sess.ID() != record.ID {
		t.Fatalf("expected session ID %s, got %s", record.ID, sess.ID())
	}

	if len(sess.GetMessages()) != len(record.Messages) {
		t.Fatalf("expected %d messages, got %d", len(record.Messages), len(sess.GetMessages()))
	}
}

func TestSharedSessionSnapshotRoundTrip(t *testing.T) {
	sess := NewShared("shared-1")
	sess.Base.SetMessages([]*message.Message{
		message.NewMessage(message.RoleSystem, "sys"),
		message.NewMessage(message.RoleUser, "Hello"),
	})

	snap := sess.Snapshot()
	if snap.Type != TypeShared {
		t.Fatalf("expected snapshot type shared, got %s", snap.Type)
	}

	rehydrated := NewSharedFromRecord(snap)
	if len(rehydrated.GetMessages()) != len(snap.Messages) {
		t.Fatalf("expected %d messages, got %d", len(snap.Messages), len(rehydrated.GetMessages()))
	}
}

// newTestStore creates a simple test store implementation
func newTestStore() Store {
	return &testStore{
		records: make(map[string]*Record),
	}
}

// testStore is a simple in-memory implementation of Store for tests.
type testStore struct {
	mu      sync.RWMutex
	records map[string]*Record
}

func (s *testStore) Save(ctx context.Context, record *Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if record == nil {
		return fmt.Errorf("record cannot be nil")
	}
	s.records[record.ID] = record.Clone()
	return nil
}

func (s *testStore) Load(ctx context.Context, id string) (*Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, exists := s.records[id]
	if !exists {
		return nil, fmt.Errorf("session %s not found", id)
	}
	return record.Clone(), nil
}

func (s *testStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.records[id]; !exists {
		return fmt.Errorf("session %s not found", id)
	}
	delete(s.records, id)
	return nil
}

func (s *testStore) List(ctx context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.records))
	for id := range s.records {
		ids = append(ids, id)
	}
	return ids, nil
}

func (s *testStore) Count(ctx context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.records), nil
}

func (s *testStore) Exists(ctx context.Context, id string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.records[id]
	return exists, nil
}
