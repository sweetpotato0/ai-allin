package middleware

import "errors"

var (
	// ErrRateLimitExceeded indicates rate limit has been exceeded
	ErrRateLimitExceeded = errors.New("rate limit exceeded")

	// ErrInvalidInput indicates input validation failed
	ErrInvalidInput = errors.New("invalid input")

	// ErrMiddlewareChainFailed indicates middleware chain execution failed
	ErrMiddlewareChainFailed = errors.New("middleware chain failed")

	// ErrInvalidContext indicates middleware context is invalid
	ErrInvalidContext = errors.New("invalid middleware context")
)
