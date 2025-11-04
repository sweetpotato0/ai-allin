package logger

import (
	"testing"

	"github.com/sweetpotato0/ai-allin/message"
	"github.com/sweetpotato0/ai-allin/middleware"
)

func TestRequestLogger(t *testing.T) {
	t.Run("logs request input", func(t *testing.T) {
		logged := ""
		logger := NewRequestLogger(func(msg string) {
			logged = msg
		})

		ctx := &middleware.Context{Input: "test input"}
		err := logger.Execute(ctx, func(c *middleware.Context) error { return nil })

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if logged == "" {
			t.Error("nothing was logged")
		}
	})

	t.Run("handles nil logger function", func(t *testing.T) {
		logger := NewRequestLogger(nil)

		ctx := &middleware.Context{Input: "test"}
		err := logger.Execute(ctx, func(c *middleware.Context) error { return nil })

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestResponseLogger(t *testing.T) {
	t.Run("logs response content", func(t *testing.T) {
		logged := ""
		logger := NewResponseLogger(func(msg string) {
			logged = msg
		})

		responseMsg := message.NewMessage(message.RoleAssistant, "test response")
		ctx := &middleware.Context{Response: responseMsg}

		err := logger.Execute(ctx, func(c *middleware.Context) error {
			return nil
		})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if logged == "" {
			t.Error("nothing was logged")
		}
	})

	t.Run("logs error when response is nil", func(t *testing.T) {
		logged := ""
		logger := NewResponseLogger(func(msg string) {
			logged = msg
		})

		ctx := &middleware.Context{Response: nil}

		err := logger.Execute(ctx, func(c *middleware.Context) error {
			return nil
		})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if logged != "" {
			t.Errorf("expected no log when response is nil, but got: %s", logged)
		}
	})
}
