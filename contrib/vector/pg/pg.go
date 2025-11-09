package pg

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
	errorskg "github.com/sweetpotato0/ai-allin/pkg/errors"
	"github.com/sweetpotato0/ai-allin/vector"
)

// PGVectorStore implements VectorStore using PostgreSQL with pgvector extension
type PGVectorStore struct {
	db          *sql.DB
	dimension   int
	tableName   string
	indexMethod string // HNSW or IVFFLAT
}

// PGVectorConfig holds pgvector configuration
type PGVectorConfig struct {
	Host      string
	Port      int
	User      string
	Password  string
	DBName    string
	SSLMode   string
	Dimension int    // Embedding dimension (default: 1536 for OpenAI)
	TableName string // Table name (default: vectors)
	IndexType string // HNSW or IVFFLAT (default: HNSW)
}

// DefaultPGVectorConfig returns default pgvector configuration
func DefaultPGVectorConfig() *PGVectorConfig {
	return &PGVectorConfig{
		Host:      "127.0.0.1",
		Port:      5432,
		User:      "postgres",
		Password:  "123456",
		DBName:    "ai_allin",
		SSLMode:   "disable",
		Dimension: 1536,
		TableName: "vectors",
		IndexType: "HNSW",
	}
}

// NewPGVectorStore creates a new pgvector-based vector store
func NewPGVectorStore(config *PGVectorConfig) (*PGVectorStore, error) {
	if config == nil {
		config = DefaultPGVectorConfig()
	}

	// Build DSN
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	store := &PGVectorStore{
		db:          db,
		dimension:   config.Dimension,
		tableName:   config.TableName,
		indexMethod: config.IndexType,
	}

	// Enable pgvector extension and create table
	if err := store.setup(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to setup pgvector: %w", err)
	}

	return store, nil
}

// setup initializes pgvector and creates necessary tables/indexes
func (s *PGVectorStore) setup(ctx context.Context) error {
	// Enable pgvector extension
	if _, err := s.db.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS vector"); err != nil {
		return fmt.Errorf("failed to create vector extension: %w", err)
	}

	// Create table
	createTableSQL := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
		id VARCHAR(255) PRIMARY KEY,
		text TEXT NOT NULL,
		embedding vector(%d) NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`, s.tableName, s.dimension)

	if _, err := s.db.ExecContext(ctx, createTableSQL); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Create index for similarity search (commented out as it depends on pgvector extension version)
	// indexName := fmt.Sprintf("%s_embedding_idx", s.tableName)
	// indexSQL := fmt.Sprintf(`
	// CREATE INDEX IF NOT EXISTS %s ON %s
	// USING %s (embedding %s) WITH (lists = 100)`,
	// 	indexName, s.tableName,
	// 	s.indexMethod, vector.CosineSimilarityOperator())
	// This will be created manually if needed

	return nil
}

// AddEmbedding adds a new embedding to the store
func (s *PGVectorStore) AddEmbedding(ctx context.Context, embedding *vector.Embedding) error {
	if embedding == nil {
		return fmt.Errorf("embedding cannot be nil")
	}

	if embedding.ID == "" {
		return fmt.Errorf("embedding ID cannot be empty")
	}

	if len(embedding.Vector) != s.dimension {
		return fmt.Errorf("embedding dimension mismatch: expected %d, got %d", s.dimension, len(embedding.Vector))
	}

	// Convert vector to string format: [1, 2, 3]
	vectorStr := s.vectorToString(embedding.Vector)

	query := fmt.Sprintf(`
	INSERT INTO %s (id, text, embedding)
	VALUES ($1, $2, $3::vector)
	ON CONFLICT (id) DO UPDATE SET
		text = EXCLUDED.text,
		embedding = EXCLUDED.embedding,
		created_at = CURRENT_TIMESTAMP
	`, s.tableName)

	_, err := s.db.ExecContext(ctx, query, embedding.ID, embedding.Text, vectorStr)
	if err != nil {
		return fmt.Errorf("failed to add embedding: %w", err)
	}

	return nil
}

// Search finds embeddings similar to the query vector
func (s *PGVectorStore) Search(ctx context.Context, queryVector []float32, topK int) ([]*vector.Embedding, error) {
	if len(queryVector) == 0 {
		return nil, fmt.Errorf("query vector cannot be empty")
	}

	if len(queryVector) != s.dimension {
		return nil, fmt.Errorf("query vector dimension mismatch: expected %d, got %d", s.dimension, len(queryVector))
	}

	if topK <= 0 {
		topK = 10
	}

	// Convert query vector to string format
	vectorStr := s.vectorToString(queryVector)

	query := fmt.Sprintf(`
	SELECT id, text, embedding
	FROM %s
	ORDER BY embedding <-> $1::vector
	LIMIT $2
	`, s.tableName)

	rows, err := s.db.QueryContext(ctx, query, vectorStr, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to search embeddings: %w", err)
	}
	defer rows.Close()

	embeddings := make([]*vector.Embedding, 0, topK)
	for rows.Next() {
		var id, text string
		var vectorStr string

		err := rows.Scan(&id, &text, &vectorStr)
		if err != nil {
			return nil, fmt.Errorf("failed to scan embedding: %w", err)
		}

		vec, err := s.stringToVector(vectorStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse vector for embedding %s: %w", id, err)
		}

		embeddings = append(embeddings, &vector.Embedding{
			ID:     id,
			Text:   text,
			Vector: vec,
		})
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating embeddings: %w", err)
	}

	return embeddings, nil
}

// DeleteEmbedding removes an embedding by ID
func (s *PGVectorStore) DeleteEmbedding(ctx context.Context, id string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = $1", s.tableName)
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete embedding: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("embedding %s: %w", id, errorskg.ErrNotFound)
	}

	return nil
}

// GetEmbedding retrieves a specific embedding by ID
func (s *PGVectorStore) GetEmbedding(ctx context.Context, id string) (*vector.Embedding, error) {
	query := fmt.Sprintf(`
	SELECT id, text, embedding
	FROM %s
	WHERE id = $1
	`, s.tableName)

	var embID, text, vectorStr string
	err := s.db.QueryRowContext(ctx, query, id).Scan(&embID, &text, &vectorStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("embedding %s: %w", id, errorskg.ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get embedding: %w", err)
	}

	vec, err := s.stringToVector(vectorStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse vector: %w", err)
	}

	return &vector.Embedding{
		ID:     embID,
		Text:   text,
		Vector: vec,
	}, nil
}

// Clear removes all embeddings
func (s *PGVectorStore) Clear(ctx context.Context) error {
	query := fmt.Sprintf("TRUNCATE TABLE %s", s.tableName)
	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to clear embeddings: %w", err)
	}
	return nil
}

// Count returns the number of embeddings
func (s *PGVectorStore) Count(ctx context.Context) (int, error) {
	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", s.tableName)
	err := s.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count embeddings: %w", err)
	}
	return count, nil
}

// Close closes the database connection
func (s *PGVectorStore) Close() error {
	return s.db.Close()
}

// Helper functions

func (s *PGVectorStore) vectorToString(vec []float32) string {
	parts := make([]string, len(vec))
	for i, v := range vec {
		parts[i] = fmt.Sprintf("%f", v)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func (s *PGVectorStore) stringToVector(str string) ([]float32, error) {
	// Simple parsing: remove brackets and convert
	str = strings.TrimPrefix(str, "[")
	str = strings.TrimSuffix(str, "]")
	parts := strings.Split(str, ",")

	vec := make([]float32, 0, len(parts))
	for i, part := range parts {
		var v float32
		n, err := fmt.Sscanf(strings.TrimSpace(part), "%f", &v)
		if err != nil || n != 1 {
			return nil, fmt.Errorf("failed to parse vector component at index %d: %q", i, part)
		}
		vec = append(vec, v)
	}
	return vec, nil
}
