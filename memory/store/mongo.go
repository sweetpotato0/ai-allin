package store

import (
	"context"
	"fmt"
	"time"

	"github.com/sweetpotato0/ai-allin/memory"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoStore implements MemoryStore using MongoDB
type MongoStore struct {
	client     *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
}

// MongoConfig holds MongoDB connection configuration
type MongoConfig struct {
	URI        string
	Database   string
	Collection string
}

// DefaultMongoConfig returns default MongoDB configuration
func DefaultMongoConfig() *MongoConfig {
	return &MongoConfig{
		URI:        "mongodb://localhost:27017",
		Database:   "ai_allin",
		Collection: "memories",
	}
}

// mongoMemory is the internal representation for MongoDB
type mongoMemory struct {
	ID        string                 `bson:"_id"`
	Content   string                 `bson:"content"`
	Metadata  map[string]interface{} `bson:"metadata"`
	CreatedAt time.Time              `bson:"created_at"`
	UpdatedAt time.Time              `bson:"updated_at"`
}

// NewMongoStore creates a new MongoDB-based memory store
func NewMongoStore(config *MongoConfig) (*MongoStore, error) {
	if config == nil {
		config = DefaultMongoConfig()
	}

	// Create MongoDB client
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.URI))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping MongoDB to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(config.Database)
	collection := db.Collection(config.Collection)

	store := &MongoStore{
		client:     client,
		db:         db,
		collection: collection,
	}

	// Create index
	if err := store.createIndexes(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return store, nil
}

// createIndexes creates indexes for efficient queries
func (s *MongoStore) createIndexes(ctx context.Context) error {
	indexModel := mongo.IndexModel{
		Keys: bson.D{{Key: "created_at", Value: -1}},
	}

	_, err := s.collection.Indexes().CreateOne(ctx, indexModel)
	return err
}

// AddMemory adds a memory to MongoDB
func (s *MongoStore) AddMemory(ctx context.Context, mem *memory.Memory) error {
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

	// Initialize metadata if nil
	if mem.Metadata == nil {
		mem.Metadata = make(map[string]interface{})
	}

	// Convert to MongoDB format
	mongoMem := mongoMemory{
		ID:        mem.ID,
		Content:   mem.Content,
		Metadata:  mem.Metadata,
		CreatedAt: mem.CreatedAt,
		UpdatedAt: mem.UpdatedAt,
	}

	// Use ReplaceOne to upsert
	opts := options.Replace().SetUpsert(true)
	filter := bson.M{"_id": mem.ID}

	_, err := s.collection.ReplaceOne(ctx, filter, mongoMem, opts)
	if err != nil {
		return fmt.Errorf("failed to add memory to MongoDB: %w", err)
	}

	return nil
}

// SearchMemory searches for memories matching the query
func (s *MongoStore) SearchMemory(ctx context.Context, query string) ([]*memory.Memory, error) {
	var filter bson.M

	// If query is empty, return all memories
	if query == "" {
		filter = bson.M{}
	} else {
		// Search for memories containing the query in content
		filter = bson.M{
			"content": bson.M{"$regex": query, "$options": "i"},
		}
	}

	// Find documents
	cursor, err := s.collection.Find(ctx, filter, options.Find().SetSort(bson.M{"created_at": -1}))
	if err != nil {
		return nil, fmt.Errorf("failed to search memories: %w", err)
	}
	defer cursor.Close(ctx)

	// Decode results
	var mongoMemories []mongoMemory
	if err := cursor.All(ctx, &mongoMemories); err != nil {
		return nil, fmt.Errorf("failed to decode memories: %w", err)
	}

	// Convert back to Memory
	memories := make([]*memory.Memory, len(mongoMemories))
	for i, m := range mongoMemories {
		memories[i] = &memory.Memory{
			ID:        m.ID,
			Content:   m.Content,
			Metadata:  m.Metadata,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		}
	}

	return memories, nil
}

// Clear removes all memories from MongoDB
func (s *MongoStore) Clear(ctx context.Context) error {
	_, err := s.collection.DeleteMany(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("failed to clear memories: %w", err)
	}
	return nil
}

// Count returns the number of memories in MongoDB
func (s *MongoStore) Count(ctx context.Context) (int, error) {
	count, err := s.collection.EstimatedDocumentCount(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count memories: %w", err)
	}
	return int(count), nil
}

// Close closes the MongoDB connection
func (s *MongoStore) Close(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	return s.client.Disconnect(ctx)
}

// Ping checks if MongoDB connection is alive
func (s *MongoStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx, nil)
}

// DeleteMemory deletes a memory by ID
func (s *MongoStore) DeleteMemory(ctx context.Context, id string) error {
	result, err := s.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete memory: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("memory not found")
	}

	return nil
}

// GetMemoryByID retrieves a specific memory by ID
func (s *MongoStore) GetMemoryByID(ctx context.Context, id string) (*memory.Memory, error) {
	var mongoMem mongoMemory
	err := s.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&mongoMem)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("memory not found")
		}
		return nil, fmt.Errorf("failed to get memory: %w", err)
	}

	return &memory.Memory{
		ID:        mongoMem.ID,
		Content:   mongoMem.Content,
		Metadata:  mongoMem.Metadata,
		CreatedAt: mongoMem.CreatedAt,
		UpdatedAt: mongoMem.UpdatedAt,
	}, nil
}
