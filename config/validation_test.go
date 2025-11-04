package config

import (
	"testing"
)

func TestValidatorRequireNonEmpty(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{
			name:      "non-empty value",
			value:     "valid",
			wantError: false,
		},
		{
			name:      "empty value",
			value:     "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.RequireNonEmpty("test_field", tt.value)
			hasError := v.HasErrors()
			if hasError != tt.wantError {
				t.Errorf("HasErrors() = %v, want %v", hasError, tt.wantError)
			}
		})
	}
}

func TestValidatorRequirePositive(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		wantError bool
	}{
		{
			name:      "positive value",
			value:     10,
			wantError: false,
		},
		{
			name:      "zero value",
			value:     0,
			wantError: true,
		},
		{
			name:      "negative value",
			value:     -5,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.RequirePositive("test_field", tt.value)
			hasError := v.HasErrors()
			if hasError != tt.wantError {
				t.Errorf("HasErrors() = %v, want %v", hasError, tt.wantError)
			}
		})
	}
}

func TestValidatorValidateRange(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		min       int
		max       int
		wantError bool
	}{
		{
			name:      "value in range",
			value:     50,
			min:       0,
			max:       100,
			wantError: false,
		},
		{
			name:      "value below minimum",
			value:     -1,
			min:       0,
			max:       100,
			wantError: true,
		},
		{
			name:      "value above maximum",
			value:     101,
			min:       0,
			max:       100,
			wantError: true,
		},
		{
			name:      "value at minimum boundary",
			value:     0,
			min:       0,
			max:       100,
			wantError: false,
		},
		{
			name:      "value at maximum boundary",
			value:     100,
			min:       0,
			max:       100,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.ValidateRange("test_field", tt.value, tt.min, tt.max)
			hasError := v.HasErrors()
			if hasError != tt.wantError {
				t.Errorf("HasErrors() = %v, want %v", hasError, tt.wantError)
			}
		})
	}
}

func TestValidatorValidateFloatRange(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		min       float64
		max       float64
		wantError bool
	}{
		{
			name:      "value in range",
			value:     0.7,
			min:       0.0,
			max:       2.0,
			wantError: false,
		},
		{
			name:      "value below minimum",
			value:     -0.1,
			min:       0.0,
			max:       2.0,
			wantError: true,
		},
		{
			name:      "value above maximum",
			value:     2.1,
			min:       0.0,
			max:       2.0,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.ValidateFloatRange("test_field", tt.value, tt.min, tt.max)
			hasError := v.HasErrors()
			if hasError != tt.wantError {
				t.Errorf("HasErrors() = %v, want %v", hasError, tt.wantError)
			}
		})
	}
}

func TestValidatorValidatePort(t *testing.T) {
	tests := []struct {
		name      string
		port      int
		wantError bool
	}{
		{
			name:      "valid port",
			port:      8080,
			wantError: false,
		},
		{
			name:      "minimum valid port",
			port:      1,
			wantError: false,
		},
		{
			name:      "maximum valid port",
			port:      65535,
			wantError: false,
		},
		{
			name:      "port too low",
			port:      0,
			wantError: true,
		},
		{
			name:      "port too high",
			port:      65536,
			wantError: true,
		},
		{
			name:      "negative port",
			port:      -1,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.ValidatePort("port", tt.port)
			hasError := v.HasErrors()
			if hasError != tt.wantError {
				t.Errorf("HasErrors() = %v, want %v", hasError, tt.wantError)
			}
		})
	}
}

func TestValidatorValidateDBNumber(t *testing.T) {
	tests := []struct {
		name      string
		db        int
		wantError bool
	}{
		{
			name:      "valid db number",
			db:        5,
			wantError: false,
		},
		{
			name:      "minimum valid db",
			db:        0,
			wantError: false,
		},
		{
			name:      "maximum valid db",
			db:        15,
			wantError: false,
		},
		{
			name:      "db too low",
			db:        -1,
			wantError: true,
		},
		{
			name:      "db too high",
			db:        16,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.ValidateDBNumber("db", tt.db)
			hasError := v.HasErrors()
			if hasError != tt.wantError {
				t.Errorf("HasErrors() = %v, want %v", hasError, tt.wantError)
			}
		})
	}
}

func TestValidatorValidateOneOf(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		allowed   []string
		wantError bool
	}{
		{
			name:      "value is allowed",
			value:     "disable",
			allowed:   []string{"disable", "require", "verify-ca"},
			wantError: false,
		},
		{
			name:      "value not allowed",
			value:     "invalid",
			allowed:   []string{"disable", "require", "verify-ca"},
			wantError: true,
		},
		{
			name:      "empty allowed list",
			value:     "any",
			allowed:   []string{},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.ValidateOneOf("field", tt.value, tt.allowed...)
			hasError := v.HasErrors()
			if hasError != tt.wantError {
				t.Errorf("HasErrors() = %v, want %v", hasError, tt.wantError)
			}
		})
	}
}

func TestValidatorValidateMinLength(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		minLen    int
		wantError bool
	}{
		{
			name:      "sufficient length",
			value:     "validkey123",
			minLen:    10,
			wantError: false,
		},
		{
			name:      "exact minimum length",
			value:     "1234567890",
			minLen:    10,
			wantError: false,
		},
		{
			name:      "insufficient length",
			value:     "short",
			minLen:    10,
			wantError: true,
		},
		{
			name:      "empty string with requirement",
			value:     "",
			minLen:    1,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.ValidateMinLength("field", tt.value, tt.minLen)
			hasError := v.HasErrors()
			if hasError != tt.wantError {
				t.Errorf("HasErrors() = %v, want %v", hasError, tt.wantError)
			}
		})
	}
}

func TestValidatorMultipleErrors(t *testing.T) {
	v := NewValidator()
	v.RequireNonEmpty("field1", "")
	v.RequirePositive("field2", 0)
	v.ValidatePort("field3", 99999)

	if !v.HasErrors() {
		t.Errorf("HasErrors() = false, want true")
	}

	errs := v.Errors()
	if len(errs) != 3 {
		t.Errorf("Errors() count = %d, want 3", len(errs))
	}

	err := v.Error()
	if err == nil {
		t.Errorf("Error() = nil, want non-nil error")
	}
}

func TestValidatePostgresConfig(t *testing.T) {
	tests := []struct {
		name      string
		host      string
		port      int
		user      string
		password  string
		dbName    string
		sslMode   string
		wantError bool
	}{
		{
			name:      "valid config",
			host:      "localhost",
			port:      5432,
			user:      "postgres",
			password:  "secure_password",
			dbName:    "ai_allin",
			sslMode:   "disable",
			wantError: false,
		},
		{
			name:      "missing host",
			host:      "",
			port:      5432,
			user:      "postgres",
			password:  "password",
			dbName:    "db",
			sslMode:   "disable",
			wantError: true,
		},
		{
			name:      "invalid port",
			host:      "localhost",
			port:      99999,
			user:      "postgres",
			password:  "password",
			dbName:    "db",
			sslMode:   "disable",
			wantError: true,
		},
		{
			name:      "invalid ssl mode",
			host:      "localhost",
			port:      5432,
			user:      "postgres",
			password:  "password",
			dbName:    "db",
			sslMode:   "invalid",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePostgresConfig(tt.host, tt.port, tt.user, tt.password, tt.dbName, tt.sslMode)
			hasError := err != nil
			if hasError != tt.wantError {
				t.Errorf("ValidatePostgresConfig() error = %v, wantError %v", hasError, tt.wantError)
			}
		})
	}
}

func TestValidateRedisConfig(t *testing.T) {
	tests := []struct {
		name      string
		addr      string
		db        int
		prefix    string
		wantError bool
	}{
		{
			name:      "valid config",
			addr:      "localhost:6379",
			db:        0,
			prefix:    "ai-allin:",
			wantError: false,
		},
		{
			name:      "missing addr",
			addr:      "",
			db:        0,
			prefix:    "ai-allin:",
			wantError: true,
		},
		{
			name:      "invalid db number",
			addr:      "localhost:6379",
			db:        16,
			prefix:    "ai-allin:",
			wantError: true,
		},
		{
			name:      "missing prefix",
			addr:      "localhost:6379",
			db:        0,
			prefix:    "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRedisConfig(tt.addr, tt.db, tt.prefix)
			hasError := err != nil
			if hasError != tt.wantError {
				t.Errorf("ValidateRedisConfig() error = %v, wantError %v", hasError, tt.wantError)
			}
		})
	}
}

func TestValidatePGVectorConfig(t *testing.T) {
	tests := []struct {
		name      string
		host      string
		port      int
		user      string
		password  string
		dbName    string
		sslMode   string
		dimension int
		tableName string
		indexType string
		wantError bool
	}{
		{
			name:      "valid config",
			host:      "localhost",
			port:      5432,
			user:      "postgres",
			password:  "password",
			dbName:    "db",
			sslMode:   "disable",
			dimension: 1536,
			tableName: "vectors",
			indexType: "HNSW",
			wantError: false,
		},
		{
			name:      "invalid dimension",
			host:      "localhost",
			port:      5432,
			user:      "postgres",
			password:  "password",
			dbName:    "db",
			sslMode:   "disable",
			dimension: 0,
			tableName: "vectors",
			indexType: "HNSW",
			wantError: true,
		},
		{
			name:      "invalid index type",
			host:      "localhost",
			port:      5432,
			user:      "postgres",
			password:  "password",
			dbName:    "db",
			sslMode:   "disable",
			dimension: 1536,
			tableName: "vectors",
			indexType: "INVALID",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePGVectorConfig(tt.host, tt.port, tt.user, tt.password,
				tt.dbName, tt.sslMode, tt.dimension, tt.tableName, tt.indexType)
			hasError := err != nil
			if hasError != tt.wantError {
				t.Errorf("ValidatePGVectorConfig() error = %v, wantError %v", hasError, tt.wantError)
			}
		})
	}
}

func TestValidateLLMConfig(t *testing.T) {
	tests := []struct {
		name        string
		apiKey      string
		model       string
		temperature float64
		maxTokens   int
		wantError   bool
	}{
		{
			name:        "valid config",
			apiKey:      "sk-valid-key",
			model:       "gpt-4",
			temperature: 0.7,
			maxTokens:   2000,
			wantError:   false,
		},
		{
			name:        "missing api key",
			apiKey:      "",
			model:       "gpt-4",
			temperature: 0.7,
			maxTokens:   2000,
			wantError:   true,
		},
		{
			name:        "invalid temperature",
			apiKey:      "sk-valid-key",
			model:       "gpt-4",
			temperature: 2.5,
			maxTokens:   2000,
			wantError:   true,
		},
		{
			name:        "non-positive max tokens",
			apiKey:      "sk-valid-key",
			model:       "gpt-4",
			temperature: 0.7,
			maxTokens:   0,
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLLMConfig(tt.apiKey, tt.model, tt.temperature, tt.maxTokens)
			hasError := err != nil
			if hasError != tt.wantError {
				t.Errorf("ValidateLLMConfig() error = %v, wantError %v", hasError, tt.wantError)
			}
		})
	}
}

func TestValidateRunnerConfig(t *testing.T) {
	tests := []struct {
		name             string
		maxConcurrency   int
		wantError        bool
	}{
		{
			name:           "valid config",
			maxConcurrency: 10,
			wantError:      false,
		},
		{
			name:           "zero concurrency",
			maxConcurrency: 0,
			wantError:      true,
		},
		{
			name:           "negative concurrency",
			maxConcurrency: -5,
			wantError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRunnerConfig(tt.maxConcurrency)
			hasError := err != nil
			if hasError != tt.wantError {
				t.Errorf("ValidateRunnerConfig() error = %v, wantError %v", hasError, tt.wantError)
			}
		})
	}
}

func TestValidateRateLimiterConfig(t *testing.T) {
	tests := []struct {
		name        string
		maxRequests int
		wantError   bool
	}{
		{
			name:        "valid config",
			maxRequests: 100,
			wantError:   false,
		},
		{
			name:        "zero requests",
			maxRequests: 0,
			wantError:   true,
		},
		{
			name:        "negative requests",
			maxRequests: -10,
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRateLimiterConfig(tt.maxRequests)
			hasError := err != nil
			if hasError != tt.wantError {
				t.Errorf("ValidateRateLimiterConfig() error = %v, wantError %v", hasError, tt.wantError)
			}
		})
	}
}
