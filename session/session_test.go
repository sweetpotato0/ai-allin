package session

import (
	"context"
	"testing"

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
	manager := NewManager()
	ag := agent.New()

	sess, err := manager.Create("sess1", ag)
	if err != nil {
		t.Errorf("Failed to create session: %v", err)
	}

	if sess.ID() != "sess1" {
		t.Errorf("Expected session ID sess1, got %s", sess.ID())
	}
}

func TestManagerCreateDuplicate(t *testing.T) {
	manager := NewManager()
	ag := agent.New()

	// Create first session
	_, err := manager.Create("sess1", ag)
	if err != nil {
		t.Errorf("Failed to create first session: %v", err)
	}

	// Try to create duplicate (should error)
	_, err = manager.Create("sess1", ag)
	if err == nil {
		t.Errorf("Expected error when creating duplicate session")
	}
}

func TestManagerGet(t *testing.T) {
	manager := NewManager()
	ag := agent.New()

	// Create a session
	created, _ := manager.Create("sess1", ag)

	// Retrieve it
	retrieved, err := manager.Get("sess1")
	if err != nil {
		t.Errorf("Failed to get session: %v", err)
	}

	if retrieved.ID() != created.ID() {
		t.Errorf("Retrieved session ID mismatch")
	}
}

func TestManagerGetNotFound(t *testing.T) {
	manager := NewManager()

	// Try to get non-existent session (should error)
	_, err := manager.Get("nonexistent")
	if err == nil {
		t.Errorf("Expected error when getting non-existent session")
	}
}

func TestManagerDelete(t *testing.T) {
	manager := NewManager()
	ag := agent.New()

	// Create a session
	manager.Create("sess1", ag)

	// Delete it
	err := manager.Delete("sess1")
	if err != nil {
		t.Errorf("Failed to delete session: %v", err)
	}

	// Try to retrieve (should error)
	_, err = manager.Get("sess1")
	if err == nil {
		t.Errorf("Expected error when getting deleted session")
	}
}

func TestManagerDeleteNotFound(t *testing.T) {
	manager := NewManager()

	// Try to delete non-existent session (should error)
	err := manager.Delete("nonexistent")
	if err == nil {
		t.Errorf("Expected error when deleting non-existent session")
	}
}

func TestManagerList(t *testing.T) {
	manager := NewManager()
	ag := agent.New()

	// Create multiple sessions
	manager.Create("sess1", ag)
	manager.Create("sess2", ag)
	manager.Create("sess3", ag)

	// List sessions
	sessions := manager.List()
	if len(sessions) != 3 {
		t.Errorf("Expected 3 sessions, got %d", len(sessions))
	}
}

func TestManagerListEmpty(t *testing.T) {
	manager := NewManager()

	sessions := manager.List()
	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions for empty manager")
	}
}

func TestManagerCount(t *testing.T) {
	manager := NewManager()
	ag := agent.New()

	if manager.Count() != 0 {
		t.Errorf("Expected 0 sessions initially")
	}

	manager.Create("sess1", ag)
	if manager.Count() != 1 {
		t.Errorf("Expected 1 session after create")
	}

	manager.Create("sess2", ag)
	if manager.Count() != 2 {
		t.Errorf("Expected 2 sessions after second create")
	}

	manager.Delete("sess1")
	if manager.Count() != 1 {
		t.Errorf("Expected 1 session after delete")
	}
}

func TestManagerClear(t *testing.T) {
	manager := NewManager()
	ag := agent.New()

	// Create multiple sessions
	manager.Create("sess1", ag)
	manager.Create("sess2", ag)

	if manager.Count() != 2 {
		t.Errorf("Expected 2 sessions before cleanup")
	}

	// Delete all using Delete method
	manager.Delete("sess1")
	manager.Delete("sess2")

	if manager.Count() != 0 {
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
	manager := NewManager()
	ag1 := agent.New(agent.WithName("Agent1"))
	ag2 := agent.New(agent.WithName("Agent2"))

	// Create sessions with different agents
	sess1, err := manager.Create("sess1", ag1)
	if err != nil {
		t.Errorf("Failed to create sess1: %v", err)
	}

	sess2, err := manager.Create("sess2", ag2)
	if err != nil {
		t.Errorf("Failed to create sess2: %v", err)
	}

	// Verify they are different sessions
	if sess1.ID() == sess2.ID() {
		t.Errorf("Sessions should have different IDs")
	}

	// Verify list contains both
	sessions := manager.List()
	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions in list")
	}
}
