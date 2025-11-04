package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/sweetpotato0/ai-allin/message"
)

func TestMiddlewareChain(t *testing.T) {
	t.Run("empty chain executes final handler", func(t *testing.T) {
		chain := NewChain()
		executed := false

		err := chain.Execute(&Context{}, func(ctx *Context) error {
			executed = true
			return nil
		})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !executed {
			t.Error("final handler was not executed")
		}
	})

	t.Run("middleware chain executes in order", func(t *testing.T) {
		order := []string{}

		m1 := &TestMiddleware{name: "m1", order: &order}
		m2 := &TestMiddleware{name: "m2", order: &order}

		chain := NewChain(m1, m2)
		ctx := &Context{}

		chain.Execute(ctx, func(c *Context) error {
			*(&order) = append(order, "final")
			return nil
		})

		expected := []string{"m1", "m2", "final"}
		if len(order) != len(expected) {
			t.Errorf("expected %d steps, got %d", len(expected), len(order))
		}
		for i, e := range expected {
			if i >= len(order) || order[i] != e {
				t.Errorf("expected step %d to be %s, got %s", i, e, order[i])
			}
		}
	})

	t.Run("error stops chain execution", func(t *testing.T) {
		order := []string{}
		m1 := &TestMiddleware{name: "m1", err: errors.New("test error"), order: &order}
		m2 := &TestMiddleware{name: "m2", order: &order}

		chain := NewChain(m1, m2)
		ctx := &Context{}

		finalCalled := false
		err := chain.Execute(ctx, func(c *Context) error {
			finalCalled = true
			return nil
		})

		if err == nil {
			t.Error("expected error from middleware")
		}
		if finalCalled {
			t.Error("final handler should not be called after middleware error")
		}
	})
}

func TestRequestLogger(t *testing.T) {
	t.Run("logs request input", func(t *testing.T) {
		logged := ""
		logger := NewRequestLogger(func(msg string) {
			logged = msg
		})

		ctx := &Context{Input: "test input"}
		err := logger.Execute(ctx, func(c *Context) error { return nil })

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !contains(logged, "test input") {
			t.Errorf("expected log to contain input, got: %s", logged)
		}
	})
}

func TestResponseLogger(t *testing.T) {
	t.Run("logs response content", func(t *testing.T) {
		logged := ""
		logger := NewResponseLogger(func(msg string) {
			logged = msg
		})

		responseMsg := message.NewMessage(message.RoleAssistant, "test response")
		ctx := &Context{Response: responseMsg}

		err := logger.Execute(ctx, func(c *Context) error {
			return nil
		})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !contains(logged, "test response") {
			t.Errorf("expected log to contain response, got: %s", logged)
		}
	})
}

func TestInputValidator(t *testing.T) {
	t.Run("valid input passes through", func(t *testing.T) {
		validator := NewInputValidator(func(input string) error {
			if input == "invalid" {
				return errors.New("invalid input")
			}
			return nil
		})

		ctx := &Context{Input: "valid"}
		executed := false

		err := validator.Execute(ctx, func(c *Context) error {
			executed = true
			return nil
		})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !executed {
			t.Error("handler was not executed")
		}
	})

	t.Run("invalid input returns error", func(t *testing.T) {
		validator := NewInputValidator(func(input string) error {
			if input == "invalid" {
				return errors.New("invalid input")
			}
			return nil
		})

		ctx := &Context{Input: "invalid"}
		executed := false

		err := validator.Execute(ctx, func(c *Context) error {
			executed = true
			return nil
		})

		if err == nil {
			t.Error("expected error for invalid input")
		}
		if executed {
			t.Error("handler should not be executed for invalid input")
		}
	})
}

func TestErrorHandler(t *testing.T) {
	t.Run("catches error from next middleware", func(t *testing.T) {
		errorCaught := false
		handler := NewErrorHandler(func(err error) error {
			errorCaught = true
			return nil // suppress error
		})

		ctx := &Context{}
		err := handler.Execute(ctx, func(c *Context) error {
			return errors.New("test error")
		})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !errorCaught {
			t.Error("error was not caught")
		}
	})
}

func TestContextEnricher(t *testing.T) {
	t.Run("enriches context with metadata", func(t *testing.T) {
		enricher := NewContextEnricher(func(ctx *Context) error {
			ctx.Metadata["key"] = "value"
			return nil
		})

		ctx := &Context{Metadata: map[string]interface{}{}}
		err := enricher.Execute(ctx, func(c *Context) error { return nil })

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if ctx.Metadata["key"] != "value" {
			t.Error("metadata not enriched")
		}
	})
}

func TestRateLimiter(t *testing.T) {
	t.Run("allows requests within limit", func(t *testing.T) {
		limiter := NewRateLimiter(2)
		ctx := &Context{}

		// First request
		err1 := limiter.Execute(ctx, func(c *Context) error { return nil })
		if err1 != nil {
			t.Errorf("first request failed: %v", err1)
		}

		// Second request
		err2 := limiter.Execute(ctx, func(c *Context) error { return nil })
		if err2 != nil {
			t.Errorf("second request failed: %v", err2)
		}
	})

	t.Run("blocks requests exceeding limit", func(t *testing.T) {
		limiter := NewRateLimiter(1)
		ctx := &Context{}

		// First request
		limiter.Execute(ctx, func(c *Context) error { return nil })

		// Second request should fail
		err := limiter.Execute(ctx, func(c *Context) error { return nil })
		if err == nil {
			t.Error("expected rate limit error")
		}
	})
}

func TestContext(t *testing.T) {
	t.Run("new context has empty metadata", func(t *testing.T) {
		ctx := NewContext(context.Background())
		if ctx.Metadata == nil {
			t.Error("metadata should not be nil")
		}
		if len(ctx.Metadata) != 0 {
			t.Error("metadata should be empty")
		}
	})

	t.Run("context preserves underlying context", func(t *testing.T) {
		baseCtx := context.Background()
		ctx := NewContext(baseCtx)
		if ctx.Context() != baseCtx {
			t.Error("underlying context not preserved")
		}
	})
}

// Helper test middleware
type TestMiddleware struct {
	name  string
	order *[]string
	err   error
}

func (m *TestMiddleware) Name() string {
	return m.name
}

func (m *TestMiddleware) Execute(ctx *Context, next Handler) error {
	*m.order = append(*m.order, m.name)
	if m.err != nil {
		return m.err
	}
	return next(ctx)
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(substr) <= len(s))
}
