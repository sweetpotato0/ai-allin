package config

import (
	"fmt"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("config validation failed for field %q: %s", e.Field, e.Message)
}

// Validator provides configuration validation utilities
type Validator struct {
	errors []ValidationError
}

// NewValidator creates a new configuration validator
func NewValidator() *Validator {
	return &Validator{
		errors: []ValidationError{},
	}
}

// RequireNonEmpty validates that a string field is not empty
func (v *Validator) RequireNonEmpty(field, value string) *Validator {
	if value == "" {
		v.errors = append(v.errors, ValidationError{
			Field:   field,
			Message: "value cannot be empty",
		})
	}
	return v
}

// RequirePositive validates that an integer field is greater than 0
func (v *Validator) RequirePositive(field string, value int) *Validator {
	if value <= 0 {
		v.errors = append(v.errors, ValidationError{
			Field:   field,
			Message: fmt.Sprintf("value must be positive, got %d", value),
		})
	}
	return v
}

// ValidateRange validates that an integer field is within a range [min, max]
func (v *Validator) ValidateRange(field string, value, min, max int) *Validator {
	if value < min || value > max {
		v.errors = append(v.errors, ValidationError{
			Field:   field,
			Message: fmt.Sprintf("value must be between %d and %d, got %d", min, max, value),
		})
	}
	return v
}

// ValidateFloatRange validates that a float field is within a range [min, max]
func (v *Validator) ValidateFloatRange(field string, value, min, max float64) *Validator {
	if value < min || value > max {
		v.errors = append(v.errors, ValidationError{
			Field:   field,
			Message: fmt.Sprintf("value must be between %.2f and %.2f, got %.2f", min, max, value),
		})
	}
	return v
}

// ValidatePort validates that a port number is valid (1-65535)
func (v *Validator) ValidatePort(field string, port int) *Validator {
	return v.ValidateRange(field, port, 1, 65535)
}

// ValidateDBNumber validates that a database number is valid (0-15 for Redis)
func (v *Validator) ValidateDBNumber(field string, db int) *Validator {
	return v.ValidateRange(field, db, 0, 15)
}

// ValidateOneOf validates that a string value is one of the allowed options
func (v *Validator) ValidateOneOf(field string, value string, allowed ...string) *Validator {
	for _, a := range allowed {
		if a == value {
			return v
		}
	}
	v.errors = append(v.errors, ValidationError{
		Field:   field,
		Message: fmt.Sprintf("value must be one of %v, got %q", allowed, value),
	})
	return v
}

// ValidateMinLength validates that a string field has minimum length
func (v *Validator) ValidateMinLength(field string, value string, minLen int) *Validator {
	if len(value) < minLen {
		v.errors = append(v.errors, ValidationError{
			Field:   field,
			Message: fmt.Sprintf("value must be at least %d characters long, got %d", minLen, len(value)),
		})
	}
	return v
}

// HasErrors returns true if there are any validation errors
func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

// Error returns a combined error message or nil if no errors
func (v *Validator) Error() error {
	if !v.HasErrors() {
		return nil
	}

	msg := "configuration validation failed:\n"
	for _, e := range v.errors {
		msg += fmt.Sprintf("  - %s: %s\n", e.Field, e.Message)
	}
	return fmt.Errorf(msg)
}

// Errors returns all validation errors
func (v *Validator) Errors() []ValidationError {
	return v.errors
}

// ValidatePostgresConfig validates PostgreSQL configuration
func ValidatePostgresConfig(host string, port int, user string, password string, dbName string, sslMode string) error {
	v := NewValidator()

	v.RequireNonEmpty("host", host)
	v.ValidatePort("port", port)
	v.RequireNonEmpty("user", user)
	v.RequireNonEmpty("password", password)
	v.RequireNonEmpty("dbName", dbName)
	v.ValidateOneOf("sslMode", sslMode, "disable", "require", "verify-ca", "verify-full")

	return v.Error()
}

// ValidateRedisConfig validates Redis configuration
func ValidateRedisConfig(addr string, db int, prefix string) error {
	v := NewValidator()

	v.RequireNonEmpty("addr", addr)
	v.ValidateDBNumber("db", db)
	v.RequireNonEmpty("prefix", prefix)

	return v.Error()
}

// ValidateMongoDBConfig validates MongoDB configuration
func ValidateMongoDBConfig(uri string, database string, collection string) error {
	v := NewValidator()

	v.RequireNonEmpty("uri", uri)
	v.RequireNonEmpty("database", database)
	v.RequireNonEmpty("collection", collection)

	return v.Error()
}

// ValidatePGVectorConfig validates PGVector configuration
func ValidatePGVectorConfig(host string, port int, user string, password string, dbName string,
	sslMode string, dimension int, tableName string, indexType string) error {
	v := NewValidator()

	v.RequireNonEmpty("host", host)
	v.ValidatePort("port", port)
	v.RequireNonEmpty("user", user)
	v.RequireNonEmpty("password", password)
	v.RequireNonEmpty("dbName", dbName)
	v.ValidateOneOf("sslMode", sslMode, "disable", "require", "verify-ca", "verify-full")
	v.ValidateRange("dimension", dimension, 1, 65535)
	v.RequireNonEmpty("tableName", tableName)
	v.ValidateOneOf("indexType", indexType, "HNSW", "IVFFLAT")

	return v.Error()
}

// ValidateLLMConfig validates LLM provider configuration
func ValidateLLMConfig(apiKey string, model string, temperature float64, maxTokens int) error {
	v := NewValidator()

	v.RequireNonEmpty("apiKey", apiKey)
	v.RequireNonEmpty("model", model)
	v.ValidateFloatRange("temperature", temperature, 0.0, 2.0)
	v.RequirePositive("maxTokens", maxTokens)

	return v.Error()
}

// ValidateRunnerConfig validates runner configuration
func ValidateRunnerConfig(maxConcurrency int) error {
	v := NewValidator()
	v.RequirePositive("maxConcurrency", maxConcurrency)
	return v.Error()
}

// ValidateRateLimiterConfig validates rate limiter configuration
func ValidateRateLimiterConfig(maxRequests int) error {
	v := NewValidator()
	v.RequirePositive("maxRequests", maxRequests)
	return v.Error()
}
