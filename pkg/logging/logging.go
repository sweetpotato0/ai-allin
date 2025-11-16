package logging

import (
	"log/slog"
	"os"
	"strings"
	"sync"
)

var (
	defaultLogger *slog.Logger
	mu            sync.RWMutex
)

// Logger returns the process-wide logger, lazily initialised using environment
// variables for format and level:
//   - AIALLIN_LOG_FORMAT: "json" (default) or "text"
//   - AIALLIN_LOG_LEVEL: debug|info|warn|error
func Logger() *slog.Logger {
	mu.RLock()
	if defaultLogger != nil {
		defer mu.RUnlock()
		return defaultLogger
	}
	mu.RUnlock()

	mu.Lock()
	defer mu.Unlock()
	if defaultLogger == nil {
		defaultLogger = newLoggerFromEnv()
	}
	return defaultLogger
}

// SetLogger overrides the global logger; mainly useful for tests.
func SetLogger(l *slog.Logger) {
	if l == nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	defaultLogger = l
}

// WithComponent attaches a component field to the shared logger.
func WithComponent(component string) *slog.Logger {
	return Logger().With("component", component)
}

func newLoggerFromEnv() *slog.Logger {
	level := slog.LevelInfo
	if env := strings.ToLower(os.Getenv("AIALLIN_LOG_LEVEL")); env != "" {
		switch env {
		case "debug":
			level = slog.LevelDebug
		case "warn":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		case "info":
			level = slog.LevelInfo
		}
	}
	opts := &slog.HandlerOptions{Level: level}
	format := strings.ToLower(os.Getenv("AIALLIN_LOG_FORMAT"))
	var handler slog.Handler
	switch format {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	default:
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	return slog.New(handler).With("service", "ai-allin")
}
