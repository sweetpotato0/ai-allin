package errorhandler

import (
	"github.com/sweetpotato0/ai-allin/middleware"
)

// ErrorHandlerFunc handles errors
type ErrorHandlerFunc func(error) error

// ErrorHandler handles errors in the middleware chain
type ErrorHandler struct {
	handler ErrorHandlerFunc
}

// NewErrorHandler creates an error handling middleware
func NewErrorHandler(handler ErrorHandlerFunc) *ErrorHandler {
	return &ErrorHandler{handler: handler}
}

// Name returns the middleware name
func (m *ErrorHandler) Name() string {
	return "ErrorHandler"
}

// Execute handles errors from downstream middlewares
func (m *ErrorHandler) Execute(ctx *middleware.Context, next middleware.Handler) error {
	err := next(ctx)
	if err != nil && m.handler != nil {
		return m.handler(err)
	}
	return err
}
