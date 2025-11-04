package limiter

import (
	"testing"

	"github.com/sweetpotato0/ai-allin/middleware"
)

func TestRateLimiter(t *testing.T) {
	t.Run("allows requests within limit", func(t *testing.T) {
		limiter := NewRateLimiter(2)
		ctx := &middleware.Context{}

		// First request
		err1 := limiter.Execute(ctx, func(c *middleware.Context) error { return nil })
		if err1 != nil {
			t.Errorf("first request failed: %v", err1)
		}

		// Second request
		err2 := limiter.Execute(ctx, func(c *middleware.Context) error { return nil })
		if err2 != nil {
			t.Errorf("second request failed: %v", err2)
		}
	})

	t.Run("blocks requests exceeding limit", func(t *testing.T) {
		limiter := NewRateLimiter(1)
		ctx := &middleware.Context{}

		// First request
		limiter.Execute(ctx, func(c *middleware.Context) error { return nil })

		// Second request should fail
		err := limiter.Execute(ctx, func(c *middleware.Context) error { return nil })
		if err == nil {
			t.Error("expected rate limit error")
		}
		if err != ErrRateLimitExceeded {
			t.Errorf("expected ErrRateLimitExceeded, got %v", err)
		}
	})

	t.Run("can reset counter", func(t *testing.T) {
		limiter := NewRateLimiter(1)
		ctx := &middleware.Context{}

		// First request
		limiter.Execute(ctx, func(c *middleware.Context) error { return nil })

		// Reset
		limiter.Reset()

		// Should be able to make another request
		err := limiter.Execute(ctx, func(c *middleware.Context) error { return nil })
		if err != nil {
			t.Errorf("request after reset failed: %v", err)
		}
	})

	t.Run("tracks counter correctly", func(t *testing.T) {
		limiter := NewRateLimiter(5)
		ctx := &middleware.Context{}

		for i := 0; i < 3; i++ {
			limiter.Execute(ctx, func(c *middleware.Context) error { return nil })
		}

		if limiter.GetCounter() != 3 {
			t.Errorf("expected counter to be 3, got %d", limiter.GetCounter())
		}
	})
}
