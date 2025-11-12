package hybrid

import (
	"context"
	"testing"

	"github.com/sweetpotato0/ai-allin/contrib/reranker/mmr"
	"github.com/sweetpotato0/ai-allin/rag/chunking"
	"github.com/sweetpotato0/ai-allin/rag/document"
	"github.com/sweetpotato0/ai-allin/vector"
)

type stubVectorStore struct {
	embeddings map[string]*vector.Embedding
}

func newStubVectorStore() *stubVectorStore {
	return &stubVectorStore{embeddings: make(map[string]*vector.Embedding)}
}

func (s *stubVectorStore) AddEmbedding(ctx context.Context, emb *vector.Embedding) error {
	s.embeddings[emb.ID] = emb
	return nil
}

func (s *stubVectorStore) Search(ctx context.Context, query []float32, topK int) ([]*vector.Embedding, error) {
	results := make([]*vector.Embedding, 0, len(s.embeddings))
	for _, emb := range s.embeddings {
		results = append(results, emb)
	}
	return results, nil
}

func (s *stubVectorStore) DeleteEmbedding(ctx context.Context, id string) error {
	delete(s.embeddings, id)
	return nil
}
func (s *stubVectorStore) GetEmbedding(ctx context.Context, id string) (*vector.Embedding, error) {
	return s.embeddings[id], nil
}
func (s *stubVectorStore) Clear(ctx context.Context) error {
	s.embeddings = make(map[string]*vector.Embedding)
	return nil
}
func (s *stubVectorStore) Count(ctx context.Context) (int, error) { return len(s.embeddings), nil }

type stubEmbedder struct{}

func (s *stubEmbedder) EmbedDocument(ctx context.Context, chunk document.Chunk) ([]float32, error) {
	return []float32{float32(len(chunk.Content))}, nil
}
func (s *stubEmbedder) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	return []float32{float32(len(query))}, nil
}

func TestHybridEngineCombinesSignals(t *testing.T) {
	store := newStubVectorStore()
	emb := &stubEmbedder{}
	engine, err := New(store, emb, WithChunker(chunking.NewSimpleChunker()), WithReranker(mmr.New()))
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	err = engine.IndexDocuments(context.Background(),
		document.Document{ID: "doc-1", Content: "AADDCC 是万能药物，但吃多了会精神异常。"},
		document.Document{ID: "doc-2", Content: "普通感冒的冗长描述。"},
	)
	if err != nil {
		t.Fatalf("index error: %v", err)
	}

	results, err := engine.Search(context.Background(), "AADDCC 副作用")
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("expected hybrid hits")
	}
	// best hit should come from doc-1
	if results[0].Chunk.DocumentID != "doc-1" {
		t.Fatalf("expected doc-1 first, got %+v", results[0])
	}
}
