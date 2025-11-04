package limiter

import (
	"errors"
	"sync"

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
	mu          sync.Mutex
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
	m.mu.Lock()
	if m.counter >= m.maxRequests {
		m.mu.Unlock()
		return ErrRateLimitExceeded
	}
	m.counter++
	m.mu.Unlock()
	return next(ctx)
}

// Reset resets the rate limiter counter
func (m *RateLimiter) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counter = 0
}

// GetCounter returns current request count
func (m *RateLimiter) GetCounter() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.counter
}
