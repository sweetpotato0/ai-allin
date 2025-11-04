package middleware

import (
	"context"
	"fmt"

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

// List returns a copy of all middlewares in the chain
func (c *MiddlewareChain) List() []Middleware {
	if c == nil || len(c.middlewares) == 0 {
		return []Middleware{}
	}
	result := make([]Middleware, len(c.middlewares))
	copy(result, c.middlewares)
	return result
}

// Execute runs all middlewares in the chain
func (c *MiddlewareChain) Execute(ctx *Context, finalHandler Handler) error {
	return c.executeMiddleware(ctx, 0, finalHandler)
}

// executeMiddleware recursively executes middlewares in sequence
func (c *MiddlewareChain) executeMiddleware(ctx *Context, index int, finalHandler Handler) error {
	defer func() {
		if r := recover(); r != nil {
			// Handle panic in middleware chain
			ctx.Error = fmt.Errorf("panic in middleware chain: %v", r)
		}
	}()

	if index >= len(c.middlewares) {
		// All middlewares executed, call the final handler
		if err := finalHandler(ctx); err != nil {
			return err
		}
		return ctx.Error
	}

	// Create a handler for the next middleware
	nextHandler := func(ctx *Context) error {
		return c.executeMiddleware(ctx, index+1, finalHandler)
	}

	// Execute current middleware with panic protection
	defer func() {
		if r := recover(); r != nil {
			ctx.Error = fmt.Errorf("panic in middleware %s: %v", c.middlewares[index].Name(), r)
		}
	}()

	return c.middlewares[index].Execute(ctx, nextHandler)
}
