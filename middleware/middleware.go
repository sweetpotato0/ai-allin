package middleware

import (
	"context"

	"github.com/sweetpotato0/ai-allin/message"
)

// Context represents the middleware execution context
type Context struct {
	// Original user input
	Input string

	// Messages before processing
	Messages []*message.Message

	// Response from LLM
	Response *message.Message

	// Error from execution
	Error error

	// Metadata for passing data between middlewares
	Metadata map[string]interface{}

	// Internal state
	context context.Context
}

// NewContext creates a new middleware context
func NewContext(ctx context.Context) *Context {
	return &Context{
		Metadata: make(map[string]interface{}),
		context:  ctx,
	}
}

// Context returns the underlying context.Context
func (c *Context) Context() context.Context {
	return c.context
}

// Middleware defines the interface for middleware components
// Middlewares can intercept and modify requests/responses in an agent execution pipeline
type Middleware interface {
	// Name returns the name of the middleware for logging and debugging
	Name() string

	// Execute runs the middleware logic
	// It receives the current context and a next handler to continue the chain
	// Returning error will stop the middleware chain
	Execute(ctx *Context, next Handler) error
}

// Handler is the function called to pass control to the next middleware
type Handler func(*Context) error

// MiddlewareChain represents a sequence of middleware to be executed
type MiddlewareChain struct {
	middlewares []Middleware
}

// NewChain creates a new middleware chain
func NewChain(middlewares ...Middleware) *MiddlewareChain {
	return &MiddlewareChain{
		middlewares: middlewares,
	}
}

// Add appends a middleware to the chain
func (c *MiddlewareChain) Add(m Middleware) *MiddlewareChain {
	c.middlewares = append(c.middlewares, m)
	return c
}

// Execute runs all middlewares in the chain
func (c *MiddlewareChain) Execute(ctx *Context, finalHandler Handler) error {
	return c.executeMiddleware(ctx, 0, finalHandler)
}

// executeMiddleware recursively executes middlewares in sequence
func (c *MiddlewareChain) executeMiddleware(ctx *Context, index int, finalHandler Handler) error {
	if index >= len(c.middlewares) {
		// All middlewares executed, call the final handler
		return finalHandler(ctx)
	}

	// Create a handler for the next middleware
	nextHandler := func(ctx *Context) error {
		return c.executeMiddleware(ctx, index+1, finalHandler)
	}

	// Execute current middleware
	return c.middlewares[index].Execute(ctx, nextHandler)
}

// RequestLogger logs incoming requests
type RequestLogger struct {
	logger func(string)
}

// NewRequestLogger creates a request logging middleware
func NewRequestLogger(logger func(string)) *RequestLogger {
	return &RequestLogger{logger: logger}
}

// Name returns the middleware name
func (m *RequestLogger) Name() string {
	return "RequestLogger"
}

// Execute logs the request
func (m *RequestLogger) Execute(ctx *Context, next Handler) error {
	if m.logger != nil {
		m.logger("[RequestLogger] Input: " + ctx.Input)
	}
	return next(ctx)
}

// ResponseLogger logs outgoing responses
type ResponseLogger struct {
	logger func(string)
}

// NewResponseLogger creates a response logging middleware
func NewResponseLogger(logger func(string)) *ResponseLogger {
	return &ResponseLogger{logger: logger}
}

// Name returns the middleware name
func (m *ResponseLogger) Name() string {
	return "ResponseLogger"
}

// Execute logs the response
func (m *ResponseLogger) Execute(ctx *Context, next Handler) error {
	err := next(ctx)
	if m.logger != nil {
		if ctx.Response != nil {
			m.logger("[ResponseLogger] Output: " + ctx.Response.Content)
		}
	}
	return err
}

// ErrorHandler handles errors in the middleware chain
type ErrorHandler struct {
	handler func(error) error
}

// NewErrorHandler creates an error handling middleware
func NewErrorHandler(handler func(error) error) *ErrorHandler {
	return &ErrorHandler{handler: handler}
}

// Name returns the middleware name
func (m *ErrorHandler) Name() string {
	return "ErrorHandler"
}

// Execute handles errors from downstream middlewares
func (m *ErrorHandler) Execute(ctx *Context, next Handler) error {
	err := next(ctx)
	if err != nil && m.handler != nil {
		return m.handler(err)
	}
	return err
}

// InputValidator validates and cleans input
type InputValidator struct {
	validator func(string) error
}

// NewInputValidator creates an input validation middleware
func NewInputValidator(validator func(string) error) *InputValidator {
	return &InputValidator{validator: validator}
}

// Name returns the middleware name
func (m *InputValidator) Name() string {
	return "InputValidator"
}

// Execute validates the input
func (m *InputValidator) Execute(ctx *Context, next Handler) error {
	if m.validator != nil {
		if err := m.validator(ctx.Input); err != nil {
			return err
		}
	}
	return next(ctx)
}

// ResponseFilter filters or transforms the response
type ResponseFilter struct {
	filter func(*message.Message) error
}

// NewResponseFilter creates a response filtering middleware
func NewResponseFilter(filter func(*message.Message) error) *ResponseFilter {
	return &ResponseFilter{filter: filter}
}

// Name returns the middleware name
func (m *ResponseFilter) Name() string {
	return "ResponseFilter"
}

// Execute filters the response
func (m *ResponseFilter) Execute(ctx *Context, next Handler) error {
	err := next(ctx)
	if err != nil {
		return err
	}
	if ctx.Response != nil && m.filter != nil {
		return m.filter(ctx.Response)
	}
	return nil
}

// ContextEnricher adds additional data to the middleware context
type ContextEnricher struct {
	enricher func(*Context) error
}

// NewContextEnricher creates a context enriching middleware
func NewContextEnricher(enricher func(*Context) error) *ContextEnricher {
	return &ContextEnricher{enricher: enricher}
}

// Name returns the middleware name
func (m *ContextEnricher) Name() string {
	return "ContextEnricher"
}

// Execute enriches the context
func (m *ContextEnricher) Execute(ctx *Context, next Handler) error {
	if m.enricher != nil {
		if err := m.enricher(ctx); err != nil {
			return err
		}
	}
	return next(ctx)
}

// RateLimiter middleware for rate limiting (placeholder for future implementation)
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
func (m *RateLimiter) Execute(ctx *Context, next Handler) error {
	if m.counter >= m.maxRequests {
		return ErrRateLimitExceeded
	}
	m.counter++
	return next(ctx)
}
