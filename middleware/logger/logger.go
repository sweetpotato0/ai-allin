package logger

import (
	"fmt"

	"github.com/sweetpotato0/ai-allin/middleware"
)

// LoggerFunc is the logging function signature
type LoggerFunc func(string)

// RequestLogger logs incoming requests
type RequestLogger struct {
	logger LoggerFunc
}

// NewRequestLogger creates a request logging middleware
func NewRequestLogger(logger LoggerFunc) *RequestLogger {
	return &RequestLogger{logger: logger}
}

// Name returns the middleware name
func (m *RequestLogger) Name() string {
	return "RequestLogger"
}

// Execute logs the request
func (m *RequestLogger) Execute(ctx *middleware.Context, next middleware.Handler) error {
	if m.logger != nil {
		m.logger(fmt.Sprintf("[RequestLogger] Input: %s", ctx.Input))
	}
	return next(ctx)
}

// ResponseLogger logs outgoing responses
type ResponseLogger struct {
	logger LoggerFunc
}

// NewResponseLogger creates a response logging middleware
func NewResponseLogger(logger LoggerFunc) *ResponseLogger {
	return &ResponseLogger{logger: logger}
}

// Name returns the middleware name
func (m *ResponseLogger) Name() string {
	return "ResponseLogger"
}

// Execute logs the response
func (m *ResponseLogger) Execute(ctx *middleware.Context, next middleware.Handler) error {
	err := next(ctx)
	if m.logger != nil {
		if ctx.Response != nil {
			m.logger(fmt.Sprintf("[ResponseLogger] Output: %s", ctx.Response.Content))
		} else if err != nil {
			m.logger(fmt.Sprintf("[ResponseLogger] Error: %v", err))
		}
	}
	return err
}
