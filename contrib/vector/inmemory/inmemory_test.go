package inmemory

import (
	"context"
	"testing"

	"github.com/sweetpotato0/ai-allin/vector"
)

// TestInMemoryVectorStore tests in-memory vector store
func TestInMemoryVectorStore(t *testing.T) {
	store := NewInMemoryVectorStore()
	ctx := context.Background()

	t.Run("add and retrieve embedding", func(t *testing.T) {
		emb := &vector.Embedding{
			ID:     "emb1",
			Text:   "hello world",
			Vector: []float32{0.1, 0.2, 0.3},
		}

		err := store.AddEmbedding(ctx, emb)
		if err != nil {
			t.Errorf("AddEmbedding failed: %v", err)
		}

		retrieved, err := store.GetEmbedding(ctx, "emb1")
		if err != nil {
			t.Errorf("GetEmbedding failed: %v", err)
		}

		if retrieved.Text != emb.Text {
			t.Errorf("Expected text %q, got %q", emb.Text, retrieved.Text)
		}
	})

	t.Run("search embeddings", func(t *testing.T) {
		store.Clear(ctx)

		embeddings := []*vector.Embedding{
			{ID: "emb1", Text: "apple", Vector: []float32{1.0, 0.0, 0.0}},
			{ID: "emb2", Text: "banana", Vector: []float32{0.0, 1.0, 0.0}},
			{ID: "emb3", Text: "orange", Vector: []float32{0.0, 0.0, 1.0}},
		}

		for _, emb := range embeddings {
			store.AddEmbedding(ctx, emb)
		}

		queryVector := []float32{1.0, 0.0, 0.0}
		results, err := store.Search(ctx, queryVector, 2)
		if err != nil {
			t.Errorf("Search failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}

		// First result should be the most similar (emb1)
		if results[0].ID != "emb1" {
			t.Errorf("Expected first result to be emb1, got %s", results[0].ID)
		}
	})

	t.Run("delete embedding", func(t *testing.T) {
		store.Clear(ctx)

		emb := &vector.Embedding{
			ID:     "del1",
			Text:   "to delete",
			Vector: []float32{0.5, 0.5, 0.5},
		}
		store.AddEmbedding(ctx, emb)

		err := store.DeleteEmbedding(ctx, "del1")
		if err != nil {
			t.Errorf("DeleteEmbedding failed: %v", err)
		}

		_, err = store.GetEmbedding(ctx, "del1")
		if err == nil {
			t.Error("Expected error when retrieving deleted embedding")
		}
	})

	t.Run("count embeddings", func(t *testing.T) {
		store.Clear(ctx)

		count, err := store.Count(ctx)
		if err != nil {
			t.Errorf("Count failed: %v", err)
		}

		if count != 0 {
			t.Errorf("Expected count 0, got %d", count)
		}

		emb := &vector.Embedding{
			ID:     "cnt1",
			Text:   "count me",
			Vector: []float32{0.1, 0.2, 0.3},
		}
		store.AddEmbedding(ctx, emb)

		count, err = store.Count(ctx)
		if err != nil {
			t.Errorf("Count failed: %v", err)
		}

		if count != 1 {
			t.Errorf("Expected count 1, got %d", count)
		}
	})
}

// Test vector utility functions
func TestCosineSimilarity(t *testing.T) {
	a := []float32{1.0, 0.0, 0.0}
	b := []float32{1.0, 0.0, 0.0}
	c := []float32{0.0, 1.0, 0.0}

	sim := vector.CosineSimilarity(a, b)
	if sim != 1.0 {
		t.Errorf("Expected similarity 1.0 for identical vectors, got %f", sim)
	}

	sim = vector.CosineSimilarity(a, c)
	if sim != 0.0 {
		t.Errorf("Expected similarity 0.0 for orthogonal vectors, got %f", sim)
	}
}

func TestEuclideanDistance(t *testing.T) {
	a := []float32{0.0, 0.0, 0.0}
	b := []float32{3.0, 4.0, 0.0}

	dist := vector.EuclideanDistance(a, b)
	expected := float32(5.0)
	if dist-expected > 0.1 {
		t.Errorf("Expected distance ~5.0, got %f", dist)
	}
}
