package validator

import (
	"github.com/sweetpotato0/ai-allin/message"
	"github.com/sweetpotato0/ai-allin/middleware"
)

// ValidatorFunc validates input
type ValidatorFunc func(string) error

// FilterFunc transforms or filters responses
type FilterFunc func(*message.Message) error

// InputValidator validates and cleans input
type InputValidator struct {
	validator ValidatorFunc
}

// NewInputValidator creates an input validation middleware
func NewInputValidator(validator ValidatorFunc) *InputValidator {
	return &InputValidator{validator: validator}
}

// Name returns the middleware name
func (m *InputValidator) Name() string {
	return "InputValidator"
}

// Execute validates the input
func (m *InputValidator) Execute(ctx *middleware.Context, next middleware.Handler) error {
	if m.validator != nil {
		if err := m.validator(ctx.Input); err != nil {
			return err
		}
	}
	return next(ctx)
}

// ResponseFilter filters or transforms the response
type ResponseFilter struct {
	filter FilterFunc
}

// NewResponseFilter creates a response filtering middleware
func NewResponseFilter(filter FilterFunc) *ResponseFilter {
	return &ResponseFilter{filter: filter}
}

// Name returns the middleware name
func (m *ResponseFilter) Name() string {
	return "ResponseFilter"
}

// Execute filters the response
func (m *ResponseFilter) Execute(ctx *middleware.Context, next middleware.Handler) error {
	err := next(ctx)
	if err != nil {
		return err
	}
	if ctx.Response != nil && m.filter != nil {
		return m.filter(ctx.Response)
	}
	return nil
}
