package store

import (
	"context"
	"os"
	"testing"

	"github.com/sweetpotato0/ai-allin/memory"
)

// TestMongoStore tests MongoDB store functionality
// Note: This test requires a running MongoDB server
// Set the MONGODB_URI environment variable to run tests against a real database
func TestMongoStore(t *testing.T) {
	// Skip test if not configured
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		t.Skip("MONGODB_URI not set, skipping MongoDB store tests")
	}

	// Create a test store
	config := &MongoConfig{
		URI:        mongoURI,
		Database:   "ai_allin_test",
		Collection: "memories_test",
	}

	store, err := NewMongoStore(config)
	if err != nil {
		t.Skipf("Failed to connect to MongoDB: %v", err)
	}
	defer store.Close(context.Background())

	// Clear any existing test data
	store.Clear(context.Background())

	t.Run("add and retrieve memory", func(t *testing.T) {
		ctx := context.Background()
		mem := &memory.Memory{
			Content: "Test memory content",
		}

		err := store.AddMemory(ctx, mem)
		if err != nil {
			t.Errorf("AddMemory failed: %v", err)
		}

		if mem.ID == "" {
			t.Error("Memory ID should be generated")
		}

		// Search for the memory
		results, err := store.SearchMemory(ctx, "Test")
		if err != nil {
			t.Errorf("SearchMemory failed: %v", err)
		}

		if len(results) == 0 {
			t.Error("Expected to find the memory")
		}

		if results[0].Content != mem.Content {
			t.Errorf("Expected content %q, got %q", mem.Content, results[0].Content)
		}
	})

	t.Run("search memory", func(t *testing.T) {
		ctx := context.Background()

		memories := []*memory.Memory{
			{Content: "Apple is a fruit"},
			{Content: "Banana is yellow"},
			{Content: "Cherry is small"},
		}

		for _, mem := range memories {
			if err := store.AddMemory(ctx, mem); err != nil {
				t.Fatalf("Failed to add memory: %v", err)
			}
		}

		results, err := store.SearchMemory(ctx, "fruit")
		if err != nil {
			t.Errorf("SearchMemory failed: %v", err)
		}

		if len(results) == 0 {
			t.Error("Expected to find memory with 'fruit'")
		}
	})

	t.Run("count memories", func(t *testing.T) {
		ctx := context.Background()
		store.Clear(ctx)

		count, err := store.Count(ctx)
		if err != nil {
			t.Errorf("Count failed: %v", err)
		}

		if count != 0 {
			t.Errorf("Expected count 0, got %d", count)
		}

		mem := &memory.Memory{Content: "Test"}
		store.AddMemory(ctx, mem)

		count, err = store.Count(ctx)
		if err != nil {
			t.Errorf("Count failed: %v", err)
		}

		if count != 1 {
			t.Errorf("Expected count 1, got %d", count)
		}
	})

	t.Run("delete memory", func(t *testing.T) {
		ctx := context.Background()
		store.Clear(ctx)

		mem := &memory.Memory{Content: "To delete"}
		store.AddMemory(ctx, mem)

		err := store.DeleteMemory(ctx, mem.ID)
		if err != nil {
			t.Errorf("DeleteMemory failed: %v", err)
		}

		count, _ := store.Count(ctx)
		if count != 0 {
			t.Errorf("Expected count 0 after delete, got %d", count)
		}
	})

	t.Run("get memory by id", func(t *testing.T) {
		ctx := context.Background()
		store.Clear(ctx)

		mem := &memory.Memory{Content: "Find me"}
		store.AddMemory(ctx, mem)

		retrieved, err := store.GetMemoryByID(ctx, mem.ID)
		if err != nil {
			t.Errorf("GetMemoryByID failed: %v", err)
		}

		if retrieved.Content != mem.Content {
			t.Errorf("Expected %q, got %q", mem.Content, retrieved.Content)
		}
	})
}
