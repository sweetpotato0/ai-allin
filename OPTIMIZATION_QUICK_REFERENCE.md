# QUICK REFERENCE: Critical Fixes Needed

## P0 CRITICAL ISSUES (Fix First)

### 1. RACE CONDITIONS - FIX IMMEDIATELY
**Severity**: CRITICAL - Can cause data corruption and panics

#### 1.1 Context Message Race Condition
- **File**: `context/context.go`, Line 31
- **Issue**: `c.messages` modified without locks
- **Fix**: Add `sync.RWMutex` to Context struct
```go
type Context struct {
    messages []*message.Message
    mu       sync.RWMutex  // ADD THIS
    maxSize  int
}
```

#### 1.2 Agent Message State Not Thread-Safe
- **Files**: `agent/agent.go` Line 173, `agent/stream.go` Line 40
- **Issue**: Concurrent `Run()` calls corrupt message history
- **Fix**: Use mutex in Agent for message access

#### 1.3 Tool Registry Race Condition
- **File**: `tool/tool.go`, Line 92
- **Issue**: Map access without sync in Register/Get
- **Fix**: Add `sync.RWMutex` to Registry

#### 1.4 Session Manager Cleanup Race
- **File**: `session/session.go`, Lines 187-203
- **Issue**: Delete while iterating over map
- **Fix**: Collect IDs first, then delete:
```go
toDelete := make([]string, 0)
for id, sess := range m.sessions {
    if sess.GetState() == StateInactive {
        toDelete = append(toDelete, id)
    }
}
for _, id := range toDelete {
    delete(m.sessions, id)
}
```

#### 1.5 Middleware Chain Not Thread-Safe
- **File**: `middleware/middleware.go`, Line 72
- **Issue**: Append without sync
- **Fix**: Add mutex to MiddlewareChain

#### 1.6 Prompt Manager Not Thread-Safe
- **File**: `prompt/prompt.go`, Line 40
- **Issue**: Map not protected
- **Fix**: Add RWMutex to Manager

---

### 2. HARDCODED DATABASE CREDENTIALS
**Severity**: CRITICAL - Security Vulnerability

#### 2.1 PostgreSQL Default Config
- **File**: `memory/store/postgres.go`, Lines 31-40
- **Current**: `password: "postgres"` hardcoded
- **Fix**: Read from environment
```go
func DefaultPostgresConfig() *PostgresConfig {
    return &PostgresConfig{
        Host:     os.Getenv("DB_HOST"),
        Port:     getEnvInt("DB_PORT", 5432),
        User:     os.Getenv("DB_USER"),
        Password: os.Getenv("DB_PASSWORD"),
        DBName:   os.Getenv("DB_NAME"),
        SSLMode:  os.Getenv("DB_SSLMODE"),
    }
}
```

#### 2.2 PGVector Default Config
- **File**: `vector/store/pgvector.go`, Lines 36-48
- **Fix**: Same as PostgreSQL above

#### 2.3 MongoDB Default Config
- **File**: `memory/store/mongo.go`, Lines 30-36
- **Fix**: Read from env:
```go
func DefaultMongoConfig() *MongoConfig {
    return &MongoConfig{
        URI:        os.Getenv("MONGO_URI"),
        Database:   os.Getenv("MONGO_DATABASE"),
        Collection: os.Getenv("MONGO_COLLECTION"),
    }
}
```

#### 2.4 Redis Default Config
- **File**: `memory/store/redis.go`, Lines 31-52
- **Fix**: Read from env

---

### 3. MISSING DATABASE INDEXES
**Severity**: CRITICAL - Performance bottleneck

#### 3.1 PostgreSQL Content Index Missing
- **File**: `memory/store/postgres.go`, Lines 75-89
- **Current**: Only has `created_at`, `updated_at` indexes
- **Missing**: Index on `content` column
- **Fix**: Add to createTable():
```sql
CREATE INDEX IF NOT EXISTS idx_memories_content_gin ON memories 
USING gin(to_tsvector('english', content));
```

#### 3.2 PGVector Similarity Index Missing
- **File**: `vector/store/pgvector.go`, Lines 104-111
- **Current**: Index creation commented out
- **Critical**: Similarity search will be O(n) without this
- **Fix**: Uncomment and implement:
```sql
CREATE INDEX IF NOT EXISTS idx_embeddings_vector ON vectors 
USING hnsw (embedding vector_cosine_ops) WITH (m = 16, ef_construction = 64);
```

#### 3.3 MongoDB Text Index Missing
- **File**: `memory/store/mongo.go`, Lines 85-92
- **Fix**: Add text index:
```go
indexModel := mongo.IndexModel{
    Keys: bson.D{{Key: "content", Value: "text"}},
}
s.collection.Indexes().CreateOne(ctx, indexModel)
```

---

### 4. CONNECTION POOL NOT CONFIGURED
**Severity**: HIGH - Can cause connection exhaustion

#### 4.1 PostgreSQL Pool Settings
- **Files**: 
  - `memory/store/postgres.go`, Lines 43-61
  - `vector/store/pgvector.go`, Lines 51-82
- **Fix**: Add after `sql.Open()`:
```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
db.SetConnMaxIdleTime(1 * time.Minute)
```

---

### 5. MISSING QUERY RESULT PAGINATION
**Severity**: HIGH - Memory issues with large datasets

#### 5.1 SearchMemory Returns All Results
- **Files**:
  - `memory/store/postgres.go`, Lines 147-200
  - `memory/store/mongo.go`, Lines 138-178
  - `memory/store/redis.go`, Lines 84-137
- **Issue**: No limit on results
- **Fix**: Add limit parameter to SearchMemory signature:
```go
SearchMemory(ctx context.Context, query string, limit int, offset int) ([]*memory.Memory, error)
```

#### 5.2 Vector Search Returns All Results
- **File**: `vector/store/pgvector.go`, Lines 150-207
- **Note**: Has topK parameter (good)
- **InMemory**: Needs topK enforcement

---

## P1 HIGH-PRIORITY ISSUES (Fix Soon)

### 6. STORE INITIALIZATION VALIDATION
**Severity**: MEDIUM-HIGH

#### 6.1 No Config Validation
- **Files**:
  - `memory/store/postgres.go`, Lines 43-46
  - `memory/store/mongo.go`, Lines 48-51
  - `vector/store/pgvector.go`, Lines 50-82
- **Fix**: Use config/validation.go functions

#### 6.2 Redis No Connection Ping
- **File**: `memory/store/redis.go`, Lines 31-52
- **Fix**: Add Ping() check in NewRedisStore

#### 6.3 MongoDB Hardcoded Timeout
- **File**: `memory/store/mongo.go`, Line 54
- **Fix**: Make configurable:
```go
ConnectTimeout: os.Getenv("MONGO_CONNECT_TIMEOUT"), // default 10s
```

---

### 7. ERROR HANDLING IMPROVEMENTS
- Redis key cleanup errors not checked (Lines 98-115)
- MongoDB index creation errors ignored (Lines 85-92)
- Message ID generation has no collision detection (message.go, Line 72)

---

### 8. INCOMPLETE IMPLEMENTATION
- PGVector index creation commented out - UNCOMMENT THIS
- Redis missing DeleteMemory() and GetMemoryByID()
- InMemoryStore missing Ping()

---

### 9. TIMEOUT CONFIGURATION
- Add timeout to all database operations
- Add per-node timeout to graph execution
- Add timeout to agent iteration loop

---

### 10. CODE CONSOLIDATION
- Centralize ID generation - use memory.GenerateMemoryID()
- Centralize JSON handling
- Consistent error handling across stores

---

## QUICK IMPLEMENTATION CHECKLIST

- [ ] Add sync.RWMutex to: Context, Agent, ToolRegistry, PromptManager, MiddlewareChain
- [ ] Fix session cleanup race condition (collect then delete)
- [ ] Move hardcoded DB credentials to environment variables
- [ ] Add database connection pool settings (PostgreSQL)
- [ ] Add missing indexes (PostgreSQL, PGVector, MongoDB)
- [ ] Add result pagination to SearchMemory
- [ ] Validate store configurations on init
- [ ] Add Ping() to all stores on creation
- [ ] Uncomment PGVector index creation
- [ ] Make MongoDB timeout configurable
- [ ] Ensure all stores implement full interface
- [ ] Add timeouts to database queries
- [ ] Consolidate duplicate code patterns

