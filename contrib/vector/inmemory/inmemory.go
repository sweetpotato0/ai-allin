package inmemory

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/sweetpotato0/ai-allin/vector"
)

// InMemoryVectorStore implements VectorStore using in-memory storage
type InMemoryVectorStore struct {
	embeddings map[string]*vector.Embedding
	mu         sync.RWMutex
}

// NewInMemoryVectorStore creates a new in-memory vector store
func NewInMemoryVectorStore() *InMemoryVectorStore {
	return &InMemoryVectorStore{
		embeddings: make(map[string]*vector.Embedding),
	}
}

// AddEmbedding adds a new embedding to the store
func (s *InMemoryVectorStore) AddEmbedding(ctx context.Context, embedding *vector.Embedding) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if embedding == nil {
		return fmt.Errorf("embedding cannot be nil")
	}

	if embedding.ID == "" {
		return fmt.Errorf("embedding ID cannot be empty")
	}

	if len(embedding.Vector) == 0 {
		return fmt.Errorf("embedding vector cannot be empty")
	}

	s.embeddings[embedding.ID] = embedding
	return nil
}

// Search finds embeddings similar to the query vector
func (s *InMemoryVectorStore) Search(ctx context.Context, queryVector []float32, topK int) ([]*vector.Embedding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(queryVector) == 0 {
		return nil, fmt.Errorf("query vector cannot be empty")
	}

	if topK <= 0 {
		topK = 10
	}

	// Calculate similarity for all embeddings
	type result struct {
		embedding  *vector.Embedding
		similarity float32
	}

	results := make([]result, 0, len(s.embeddings))
	for _, emb := range s.embeddings {
		if len(emb.Vector) != len(queryVector) {
			continue
		}

		// Use cosine similarity for comparison
		similarity := vector.CosineSimilarity(queryVector, emb.Vector)
		results = append(results, result{
			embedding:  emb,
			similarity: similarity,
		})
	}

	// Sort by similarity (highest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].similarity > results[j].similarity
	})

	// Extract top K results
	limit := topK
	if limit > len(results) {
		limit = len(results)
	}

	embeddings := make([]*vector.Embedding, limit)
	for i := 0; i < limit; i++ {
		embeddings[i] = results[i].embedding
	}

	return embeddings, nil
}

// DeleteEmbedding removes an embedding by ID
func (s *InMemoryVectorStore) DeleteEmbedding(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.embeddings[id]; !exists {
		return fmt.Errorf("embedding not found")
	}

	delete(s.embeddings, id)
	return nil
}

// GetEmbedding retrieves a specific embedding by ID
func (s *InMemoryVectorStore) GetEmbedding(ctx context.Context, id string) (*vector.Embedding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	emb, exists := s.embeddings[id]
	if !exists {
		return nil, fmt.Errorf("embedding not found")
	}

	return emb, nil
}

// Clear removes all embeddings
func (s *InMemoryVectorStore) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.embeddings = make(map[string]*vector.Embedding)
	return nil
}

// Count returns the number of embeddings
func (s *InMemoryVectorStore) Count(ctx context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.embeddings), nil
}
