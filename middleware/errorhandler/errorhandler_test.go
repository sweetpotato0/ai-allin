package errorhandler

import (
	"errors"
	"testing"

	"github.com/sweetpotato0/ai-allin/middleware"
)

func TestErrorHandler(t *testing.T) {
	t.Run("catches error from next middleware", func(t *testing.T) {
		errorCaught := false
		handler := NewErrorHandler(func(err error) error {
			errorCaught = true
			return nil // suppress error
		})

		ctx := &middleware.Context{}
		err := handler.Execute(ctx, func(c *middleware.Context) error {
			return errors.New("test error")
		})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !errorCaught {
			t.Error("error was not caught")
		}
	})

	t.Run("passes through non-errors", func(t *testing.T) {
		handlerCalled := false
		handler := NewErrorHandler(func(err error) error {
			handlerCalled = true
			return err
		})

		ctx := &middleware.Context{}
		err := handler.Execute(ctx, func(c *middleware.Context) error {
			return nil
		})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if handlerCalled {
			t.Error("error handler should not be called for nil errors")
		}
	})
}
