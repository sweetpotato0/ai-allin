package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/message"
	"github.com/sweetpotato0/ai-allin/session"
)

// RedisStore implements session storage using Redis
type RedisStore struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
}

// RedisConfig holds Redis configuration for sessions
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	Prefix   string
	TTL      time.Duration
}

// SessionData represents serializable session data
type SessionData struct {
	ID        string             `json:"id"`
	State     session.State      `json:"state"`
	Messages  []*message.Message `json:"messages"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
	Metadata  map[string]any     `json:"metadata"`
}

// NewRedisStore creates a new Redis-based session store
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

// RedisManager manages sessions with Redis backend
type RedisManager struct {
	store  *RedisStore
	agents map[string]*agent.Agent // Agent registry by ID
}

// NewRedisManager creates a new Redis-based session manager
func NewRedisManager(config *RedisConfig) *RedisManager {
	return &RedisManager{
		store:  NewRedisStore(config),
		agents: make(map[string]*agent.Agent),
	}
}

// RegisterAgent registers an agent for creating sessions
func (m *RedisManager) RegisterAgent(id string, ag *agent.Agent) {
	m.agents[id] = ag
}

// Create creates a new session in Redis
func (m *RedisManager) Create(ctx context.Context, id string, ag *agent.Agent) (session.Session, error) {
	// Check if session already exists
	key := m.store.prefix + id
	exists, err := m.store.client.Exists(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to check session existence: %w", err)
	}
	if exists > 0 {
		return nil, fmt.Errorf("session %s already exists", id)
	}

	// Create session data
	data := &SessionData{
		ID:        id,
		State:     session.StateActive,
		Messages:  ag.GetMessages(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  make(map[string]any),
	}

	// Serialize and store
	if err := m.saveSessionData(ctx, id, data); err != nil {
		return nil, err
	}

	// Create in-memory session wrapper
	return &redisSession{
		id:    id,
		agent: ag,
		store: m.store,
		ctx:   ctx,
	}, nil
}

// Get retrieves a session from Redis
func (m *RedisManager) Get(ctx context.Context, id string, agentID string) (session.Session, error) {
	// Load session data
	data, err := m.loadSessionData(ctx, id)
	if err != nil {
		return nil, err
	}

	// Get agent
	ag, ok := m.agents[agentID]
	if !ok {
		return nil, fmt.Errorf("agent %s not registered", agentID)
	}

	// Restore agent messages
	for _, msg := range data.Messages {
		ag.AddMessage(msg)
	}

	return &redisSession{
		id:    id,
		agent: ag,
		store: m.store,
		ctx:   ctx,
	}, nil
}

// Delete removes a session from Redis
func (m *RedisManager) Delete(ctx context.Context, id string) error {
	key := m.store.prefix + id
	if err := m.store.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// Remove from session set
	setKey := m.store.prefix + "set"
	if err := m.store.client.SRem(ctx, setKey, id).Err(); err != nil {
		return fmt.Errorf("failed to remove from session set: %w", err)
	}

	return nil
}

// List returns all session IDs from Redis
func (m *RedisManager) List(ctx context.Context) ([]string, error) {
	setKey := m.store.prefix + "set"
	ids, err := m.store.client.SMembers(ctx, setKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	return ids, nil
}

// Count returns the number of sessions in Redis
func (m *RedisManager) Count(ctx context.Context) (int, error) {
	setKey := m.store.prefix + "set"
	count, err := m.store.client.SCard(ctx, setKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to count sessions: %w", err)
	}
	return int(count), nil
}

// saveSessionData saves session data to Redis
func (m *RedisManager) saveSessionData(ctx context.Context, id string, data *SessionData) error {
	key := m.store.prefix + id

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	if err := m.store.client.Set(ctx, key, jsonData, m.store.ttl).Err(); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	// Add to session set
	setKey := m.store.prefix + "set"
	if err := m.store.client.SAdd(ctx, setKey, id).Err(); err != nil {
		return fmt.Errorf("failed to add to session set: %w", err)
	}

	return nil
}

// loadSessionData loads session data from Redis
func (m *RedisManager) loadSessionData(ctx context.Context, id string) (*SessionData, error) {
	key := m.store.prefix + id

	jsonData, err := m.store.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("session %s not found", id)
		}
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	var data SessionData
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
	}

	return &data, nil
}

// redisSession implements Session interface with Redis backend
type redisSession struct {
	id    string
	agent *agent.Agent
	store *RedisStore
	ctx   context.Context
}

func (s *redisSession) ID() string {
	return s.id
}

func (s *redisSession) Run(ctx context.Context, input string) (string, error) {
	// Run agent
	result, err := s.agent.Run(ctx, input)
	if err != nil {
		return "", err
	}

	// Update session in Redis
	data := &SessionData{
		ID:        s.id,
		State:     session.StateActive,
		Messages:  s.agent.GetMessages(),
		UpdatedAt: time.Now(),
	}

	key := s.store.prefix + s.id
	jsonData, err := json.Marshal(data)
	if err != nil {
		return result, fmt.Errorf("failed to marshal session data: %w", err)
	}

	if err := s.store.client.Set(ctx, key, jsonData, s.store.ttl).Err(); err != nil {
		return result, fmt.Errorf("failed to update session: %w", err)
	}

	return result, nil
}

func (s *redisSession) GetMessages() []*message.Message {
	return s.agent.GetMessages()
}

func (s *redisSession) GetState() session.State {
	// Load from Redis
	data, err := s.loadData()
	if err != nil {
		return session.StateClosed
	}
	return data.State
}

func (s *redisSession) Close() error {
	// Load current data
	data, err := s.loadData()
	if err != nil {
		return err
	}

	data.State = session.StateClosed
	data.UpdatedAt = time.Now()

	// Save back to Redis
	key := s.store.prefix + s.id
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	if err := s.store.client.Set(s.ctx, key, jsonData, s.store.ttl).Err(); err != nil {
		return fmt.Errorf("failed to close session: %w", err)
	}

	return nil
}

func (s *redisSession) loadData() (*SessionData, error) {
	key := s.store.prefix + s.id
	jsonData, err := s.store.client.Get(s.ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var data SessionData
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return nil, err
	}

	return &data, nil
}

// Close closes the Redis connection
func (m *RedisManager) Close() error {
	return m.store.client.Close()
}

// Ping checks if Redis connection is alive
func (m *RedisManager) Ping(ctx context.Context) error {
	return m.store.client.Ping(ctx).Err()
}
