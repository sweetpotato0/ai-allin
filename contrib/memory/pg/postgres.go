package pg

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	cfg "github.com/sweetpotato0/ai-allin/config"
	"github.com/sweetpotato0/ai-allin/memory"
	"github.com/sweetpotato0/ai-allin/pkg/env"
	errorskg "github.com/sweetpotato0/ai-allin/pkg/errors"
)

// PostgresConfigFromEnv loads PostgreSQL configuration from environment variables
func PostgresConfigFromEnv() *PostgresConfig {
	return &PostgresConfig{
		Host:     env.GetEnv("POSTGRES_HOST", "localhost"),
		Port:     env.GetEnvInt("POSTGRES_PORT", 5432),
		User:     env.GetEnv("POSTGRES_USER", "postgres"),
		Password: env.GetEnv("POSTGRES_PASSWORD", ""),
		DBName:   env.GetEnv("POSTGRES_DB", "ai_allin"),
		SSLMode:  env.GetEnv("POSTGRES_SSLMODE", "disable"),
	}
}

// PostgresStore implements MemoryStore using PostgreSQL
type PostgresStore struct {
	db *sql.DB
}

// PostgresConfig holds PostgreSQL connection configuration
type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// DefaultPostgresConfig returns default PostgreSQL configuration
func DefaultPostgresConfig() *PostgresConfig {
	return &PostgresConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "postgres",
		DBName:   "ai_allin",
		SSLMode:  "disable",
	}
}

// NewPostgresStore creates a new PostgreSQL-based memory store
func NewPostgresStore(config *PostgresConfig) (*PostgresStore, error) {
	if config == nil {
		// Try to load from environment variables first
		config = PostgresConfigFromEnv()
	}

	// Validate configuration
	if err := cfg.ValidatePostgresConfig(config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode); err != nil {
		return nil, fmt.Errorf("invalid PostgreSQL configuration: %w", err)
	}

	// Build DSN (Data Source Name)
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode)

	// Connect to database
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	// Configure connection pool for optimal performance
	db.SetMaxOpenConns(25)                 // Max concurrent connections
	db.SetMaxIdleConns(5)                  // Min idle connections for reuse
	db.SetConnMaxLifetime(5 * time.Minute) // Recycle connections after 5 min

	// Test the connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	store := &PostgresStore{db: db}

	// Create table and indexes with timeout
	if err := store.createTable(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return store, nil
}

// createTable creates the memories table if it doesn't exist
func (s *PostgresStore) createTable(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS memories (
		id VARCHAR(255) PRIMARY KEY,
		content TEXT NOT NULL,
		metadata JSONB,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_memories_created_at ON memories(created_at);
	CREATE INDEX IF NOT EXISTS idx_memories_updated_at ON memories(updated_at);
	CREATE INDEX IF NOT EXISTS idx_memories_content_gin ON memories USING GIN (to_tsvector('english', content));
	`

	_, err := s.db.ExecContext(ctx, query)
	return err
}

// AddMemory adds a memory to PostgreSQL
func (s *PostgresStore) AddMemory(ctx context.Context, mem *memory.Memory) error {
	if mem == nil {
		return fmt.Errorf("memory cannot be nil")
	}

	// Generate ID if not provided
	if mem.ID == "" {
		mem.ID = memory.GenerateMemoryID()
	}

	// Set timestamps
	now := time.Now()
	if mem.CreatedAt.IsZero() {
		mem.CreatedAt = now
	}
	mem.UpdatedAt = now

	// Convert metadata to JSON
	var metadataJSON []byte
	if len(mem.Metadata) > 0 {
		var err error
		metadataJSON, err = json.Marshal(mem.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	} else {
		metadataJSON = []byte("{}")
	}

	// Insert into database with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	query := `
	INSERT INTO memories (id, content, metadata, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT (id) DO UPDATE SET
		content = EXCLUDED.content,
		metadata = EXCLUDED.metadata,
		updated_at = EXCLUDED.updated_at
	`

	_, err := s.db.ExecContext(ctx, query,
		mem.ID,
		mem.Content,
		string(metadataJSON),
		mem.CreatedAt,
		mem.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to add memory to PostgreSQL: %w", err)
	}

	return nil
}

// SearchMemory searches for memories matching the query with pagination
// Limit is capped at 10000 to prevent memory exhaustion
func (s *PostgresStore) SearchMemory(ctx context.Context, query string) ([]*memory.Memory, error) {
	return s.SearchMemoryWithLimit(ctx, query, 1000)
}

// SearchMemoryWithLimit searches for memories with a configurable limit
func (s *PostgresStore) SearchMemoryWithLimit(ctx context.Context, query string, limit int) ([]*memory.Memory, error) {
	// Cap limit to prevent memory exhaustion
	if limit <= 0 || limit > 10000 {
		limit = 1000
	}

	// Add timeout to prevent long-running queries
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var rows *sql.Rows
	var err error

	// If query is empty, return all memories with limit
	if query == "" {
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, content, metadata, created_at, updated_at
			 FROM memories
			 ORDER BY created_at DESC
			 LIMIT $1`,
			limit)
	} else {
		// Search for memories containing the query in content using full-text search
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, content, metadata, created_at, updated_at
			 FROM memories
			 WHERE to_tsvector('english', content) @@ plainto_tsquery('english', $1)
			 ORDER BY created_at DESC
			 LIMIT $2`,
			query, limit)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to search memories: %w", err)
	}
	defer rows.Close()

	memories := make([]*memory.Memory, 0, limit)
	for rows.Next() {
		mem := &memory.Memory{}
		var metadataJSON string

		err := rows.Scan(&mem.ID, &mem.Content, &metadataJSON, &mem.CreatedAt, &mem.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan memory: %w", err)
		}

		// Unmarshal metadata JSON
		mem.Metadata = make(map[string]any)
		if metadataJSON != "" && metadataJSON != "{}" {
			err := json.Unmarshal([]byte(metadataJSON), &mem.Metadata)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		memories = append(memories, mem)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating memories: %w", err)
	}

	return memories, nil
}

// Clear removes all memories from PostgreSQL
func (s *PostgresStore) Clear(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM memories")
	if err != nil {
		return fmt.Errorf("failed to clear memories: %w", err)
	}
	return nil
}

// Count returns the number of memories in PostgreSQL
func (s *PostgresStore) Count(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM memories").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count memories: %w", err)
	}
	return count, nil
}

// Close closes the PostgreSQL connection
func (s *PostgresStore) Close() error {
	return s.db.Close()
}

// Ping checks if PostgreSQL connection is alive
func (s *PostgresStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// DeleteMemory deletes a memory by ID
func (s *PostgresStore) DeleteMemory(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM memories WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete memory: %w", err)
	}
	return nil
}

// GetMemoryByID retrieves a specific memory by ID
func (s *PostgresStore) GetMemoryByID(ctx context.Context, id string) (*memory.Memory, error) {
	mem := &memory.Memory{}
	var metadataJSON string

	query := `SELECT id, content, metadata, created_at, updated_at FROM memories WHERE id = $1`
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&mem.ID, &mem.Content, &metadataJSON, &mem.CreatedAt, &mem.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("memory %s: %w", id, errorskg.ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get memory: %w", err)
	}

	// Unmarshal metadata JSON
	mem.Metadata = make(map[string]any)
	if metadataJSON != "" && metadataJSON != "{}" {
		err := json.Unmarshal([]byte(metadataJSON), &mem.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}
	return mem, nil
}
