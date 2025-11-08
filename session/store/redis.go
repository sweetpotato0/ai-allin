package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sweetpotato0/ai-allin/session"
)

// RedisStore implements session storage using Redis.
type RedisStore struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
}

// RedisConfig holds Redis configuration for sessions.
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	Prefix   string
	TTL      time.Duration
}

// NewRedisStore creates a new Redis-based session store.
func NewRedisStore(config *RedisConfig) *RedisStore {
	if config == nil {
		config = &RedisConfig{
			Addr:   "localhost:6379",
			Prefix: "ai-allin:session:",
			TTL:    24 * time.Hour,
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

// Save persists a session record to Redis.
func (s *RedisStore) Save(ctx context.Context, record *session.Record) error {
	if record == nil || record.ID == "" {
		return fmt.Errorf("session record cannot be nil")
	}

	key := s.sessionKey(record.ID)

	raw, err := json.Marshal(record.Clone())
	if err != nil {
		return fmt.Errorf("failed to marshal session record: %w", err)
	}

	if err := s.client.Set(ctx, key, raw, s.ttl).Err(); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	setKey := s.setKey()
	if err := s.client.SAdd(ctx, setKey, record.ID).Err(); err != nil {
		return fmt.Errorf("failed to add session to index: %w", err)
	}

	return nil
}

// Load loads a session record from Redis.
func (s *RedisStore) Load(ctx context.Context, id string) (*session.Record, error) {
	key := s.sessionKey(id)
	raw, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("session %s not found", id)
		}
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	var record session.Record
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return nil, fmt.Errorf("failed to decode session record: %w", err)
	}

	return record.Clone(), nil
}

// Delete removes a session record from Redis.
func (s *RedisStore) Delete(ctx context.Context, id string) error {
	key := s.sessionKey(id)
	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	setKey := s.setKey()
	if err := s.client.SRem(ctx, setKey, id).Err(); err != nil {
		return fmt.Errorf("failed to update session index: %w", err)
	}
	return nil
}

// List returns all session IDs.
func (s *RedisStore) List(ctx context.Context) ([]string, error) {
	setKey := s.setKey()
	ids, err := s.client.SMembers(ctx, setKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	return ids, nil
}

// Count returns the number of stored sessions.
func (s *RedisStore) Count(ctx context.Context) (int, error) {
	setKey := s.setKey()
	count, err := s.client.SCard(ctx, setKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to count sessions: %w", err)
	}
	return int(count), nil
}

// Exists checks if a session exists.
func (s *RedisStore) Exists(ctx context.Context, id string) (bool, error) {
	key := s.sessionKey(id)
	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check session existence: %w", err)
	}
	return exists > 0, nil
}

// Close closes the underlying Redis client.
func (s *RedisStore) Close() error {
	return s.client.Close()
}

// Ping checks if Redis connection is alive.
func (s *RedisStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

func (s *RedisStore) sessionKey(id string) string {
	return s.prefix + id
}

func (s *RedisStore) setKey() string {
	return s.prefix + "set"
}
