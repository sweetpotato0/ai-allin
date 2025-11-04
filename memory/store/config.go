package store

import (
	"os"
	"strconv"
	"time"
)

// ConfigFromEnv loads configuration from environment variables
// Supports both explicit values and defaults with environment variable overrides

// PostgresConfigFromEnv loads PostgreSQL configuration from environment variables
func PostgresConfigFromEnv() *PostgresConfig {
	return &PostgresConfig{
		Host:     getEnv("POSTGRES_HOST", "localhost"),
		Port:     getEnvInt("POSTGRES_PORT", 5432),
		User:     getEnv("POSTGRES_USER", "postgres"),
		Password: getEnv("POSTGRES_PASSWORD", ""),
		DBName:   getEnv("POSTGRES_DB", "ai_allin"),
		SSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
	}
}

// RedisConfigFromEnv loads Redis configuration from environment variables
func RedisConfigFromEnv() *RedisConfig {
	return &RedisConfig{
		Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       getEnvInt("REDIS_DB", 0),
		Prefix:   getEnv("REDIS_PREFIX", "ai-allin:memory:"),
		TTL:      getEnvDuration("REDIS_TTL", 0),
	}
}

// RedisSessionConfigFromEnv loads Redis session configuration from environment variables
func RedisSessionConfigFromEnv() *RedisConfig {
	return &RedisConfig{
		Addr:     getEnv("REDIS_SESSION_ADDR", "localhost:6379"),
		Password: getEnv("REDIS_SESSION_PASSWORD", ""),
		DB:       getEnvInt("REDIS_SESSION_DB", 1),
		Prefix:   getEnv("REDIS_SESSION_PREFIX", "ai-allin:session:"),
		TTL:      getEnvDuration("REDIS_SESSION_TTL", 24*time.Hour),
	}
}

// MongoConfigFromEnv loads MongoDB configuration from environment variables
func MongoConfigFromEnv() *MongoConfig {
	return &MongoConfig{
		URI:        getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		Database:   getEnv("MONGODB_DB", "ai_allin"),
		Collection: getEnv("MONGODB_COLLECTION", "memories"),
	}
}

// Helper functions for environment variable reading

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
