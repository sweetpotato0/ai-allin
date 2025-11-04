package limiter

import (
	"errors"

	"github.com/sweetpotato0/ai-allin/middleware"
)

var (
	// ErrRateLimitExceeded indicates rate limit has been exceeded
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)

// RateLimiter middleware for rate limiting
type RateLimiter struct {
	maxRequests int
	counter     int
}

// NewRateLimiter creates a rate limiting middleware
func NewRateLimiter(maxRequests int) *RateLimiter {
	return &RateLimiter{maxRequests: maxRequests}
}

// Name returns the middleware name
func (m *RateLimiter) Name() string {
	return "RateLimiter"
}

// Execute checks rate limit
func (m *RateLimiter) Execute(ctx *middleware.Context, next middleware.Handler) error {
	if m.counter >= m.maxRequests {
		return ErrRateLimitExceeded
	}
	m.counter++
	return next(ctx)
}

// Reset resets the rate limiter counter
func (m *RateLimiter) Reset() {
	m.counter = 0
}

// GetCounter returns current request count
func (m *RateLimiter) GetCounter() int {
	return m.counter
}
