package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	errorskg "github.com/sweetpotato0/ai-allin/errors"
	"github.com/sweetpotato0/ai-allin/memory"
)

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
		config = DefaultPostgresConfig()
	}

	// Build DSN (Data Source Name)
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode)

	// Connect to database
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	store := &PostgresStore{db: db}

	// Create table if it doesn't exist
	if err := store.createTable(context.Background()); err != nil {
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
		mem.ID = fmt.Sprintf("mem:%d", time.Now().UnixNano())
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

	// Insert into database
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

// SearchMemory searches for memories matching the query
func (s *PostgresStore) SearchMemory(ctx context.Context, query string) ([]*memory.Memory, error) {
	var rows *sql.Rows
	var err error

	// If query is empty, return all memories
	if query == "" {
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, content, metadata, created_at, updated_at
			 FROM memories
			 ORDER BY created_at DESC`)
	} else {
		// Search for memories containing the query in content
		searchQuery := fmt.Sprintf("%%%s%%", query)
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, content, metadata, created_at, updated_at
			 FROM memories
			 WHERE content ILIKE $1
			 ORDER BY created_at DESC`,
			searchQuery)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to search memories: %w", err)
	}
	defer rows.Close()

	memories := make([]*memory.Memory, 0)
	for rows.Next() {
		mem := &memory.Memory{}
		var metadataJSON string

		err := rows.Scan(&mem.ID, &mem.Content, &metadataJSON, &mem.CreatedAt, &mem.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan memory: %w", err)
		}

		// Unmarshal metadata JSON
		mem.Metadata = make(map[string]interface{})
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
	mem.Metadata = make(map[string]interface{})
	if metadataJSON != "" && metadataJSON != "{}" {
		err := json.Unmarshal([]byte(metadataJSON), &mem.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}
	return mem, nil
}
