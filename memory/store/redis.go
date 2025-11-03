package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sweetpotato0/ai-allin/memory"
)

// RedisStore implements MemoryStore using Redis
type RedisStore struct {
	client *redis.Client
	prefix string // Key prefix for namespacing
	ttl    time.Duration
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Addr     string        // Redis server address (e.g., "localhost:6379")
	Password string        // Redis password (if any)
	DB       int           // Redis database number
	Prefix   string        // Key prefix for namespacing
	TTL      time.Duration // Time-to-live for keys (0 means no expiration)
}

// NewRedisStore creates a new Redis-based memory store
func NewRedisStore(config *RedisConfig) *RedisStore {
	if config == nil {
		config = &RedisConfig{
			Addr:   "localhost:6379",
			Prefix: "ai-allin:memory:",
			TTL:    0,
		}
	}

	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	})

	return &RedisStore{
		client: client,
		prefix: config.Prefix,
		ttl:    config.TTL,
	}
}

// AddMemory adds a memory to Redis
func (s *RedisStore) AddMemory(ctx context.Context, mem *memory.Memory) error {
	if mem == nil {
		return fmt.Errorf("memory cannot be nil")
	}

	// Generate a unique key
	key := fmt.Sprintf("%smem:%d", s.prefix, time.Now().UnixNano())

	// Serialize memory to JSON
	data, err := json.Marshal(mem)
	if err != nil {
		return fmt.Errorf("failed to marshal memory: %w", err)
	}

	// Store in Redis
	if err := s.client.Set(ctx, key, data, s.ttl).Err(); err != nil {
		return fmt.Errorf("failed to store memory in Redis: %w", err)
	}

	// Add key to a set for easy retrieval
	setKey := fmt.Sprintf("%sset", s.prefix)
	if err := s.client.SAdd(ctx, setKey, key).Err(); err != nil {
		return fmt.Errorf("failed to add memory key to set: %w", err)
	}

	return nil
}

// SearchMemory searches for memories matching the query
func (s *RedisStore) SearchMemory(ctx context.Context, query string) ([]*memory.Memory, error) {
	// Get all memory keys from the set
	setKey := fmt.Sprintf("%sset", s.prefix)
	keys, err := s.client.SMembers(ctx, setKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory keys: %w", err)
	}

	if len(keys) == 0 {
		return []*memory.Memory{}, nil
	}

	// Retrieve all memories
	memories := make([]*memory.Memory, 0, len(keys))
	for _, key := range keys {
		data, err := s.client.Get(ctx, key).Result()
		if err != nil {
			if err == redis.Nil {
				// Key expired or doesn't exist, remove from set
				s.client.SRem(ctx, setKey, key)
				continue
			}
			return nil, fmt.Errorf("failed to get memory: %w", err)
		}

		var mem memory.Memory
		if err := json.Unmarshal([]byte(data), &mem); err != nil {
			return nil, fmt.Errorf("failed to unmarshal memory: %w", err)
		}

		memories = append(memories, &mem)
	}

	// TODO: Implement proper search/filtering based on query
	return memories, nil
}

// Clear removes all memories from Redis
func (s *RedisStore) Clear(ctx context.Context) error {
	// Get all memory keys
	setKey := fmt.Sprintf("%sset", s.prefix)
	keys, err := s.client.SMembers(ctx, setKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get memory keys: %w", err)
	}

	// Delete all keys
	if len(keys) > 0 {
		if err := s.client.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("failed to delete memory keys: %w", err)
		}
	}

	// Clear the set
	if err := s.client.Del(ctx, setKey).Err(); err != nil {
		return fmt.Errorf("failed to delete memory set: %w", err)
	}

	return nil
}

// Count returns the number of memories in Redis
func (s *RedisStore) Count(ctx context.Context) (int, error) {
	setKey := fmt.Sprintf("%sset", s.prefix)
	count, err := s.client.SCard(ctx, setKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to count memories: %w", err)
	}

	return int(count), nil
}

// Close closes the Redis connection
func (s *RedisStore) Close() error {
	return s.client.Close()
}

// Ping checks if Redis connection is alive
func (s *RedisStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}
