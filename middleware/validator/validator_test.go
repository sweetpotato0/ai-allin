package validator

import (
	"errors"
	"testing"

	"github.com/sweetpotato0/ai-allin/message"
	"github.com/sweetpotato0/ai-allin/middleware"
)

func TestInputValidator(t *testing.T) {
	t.Run("valid input passes through", func(t *testing.T) {
		validator := NewInputValidator(func(input string) error {
			if input == "invalid" {
				return errors.New("invalid input")
			}
			return nil
		})

		ctx := &middleware.Context{Input: "valid"}
		executed := false

		err := validator.Execute(ctx, func(c *middleware.Context) error {
			executed = true
			return nil
		})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !executed {
			t.Error("handler was not executed")
		}
	})

	t.Run("invalid input returns error", func(t *testing.T) {
		validator := NewInputValidator(func(input string) error {
			if input == "invalid" {
				return errors.New("invalid input")
			}
			return nil
		})

		ctx := &middleware.Context{Input: "invalid"}
		executed := false

		err := validator.Execute(ctx, func(c *middleware.Context) error {
			executed = true
			return nil
		})

		if err == nil {
			t.Error("expected error for invalid input")
		}
		if executed {
			t.Error("handler should not be executed for invalid input")
		}
	})
}

func TestResponseFilter(t *testing.T) {
	t.Run("filters response successfully", func(t *testing.T) {
		filter := NewResponseFilter(func(msg *message.Message) error {
			if len(msg.Text()) > 100 {
				return errors.New("response too long")
			}
			return nil
		})

		responseMsg := message.NewMessage(message.RoleAssistant, "short response")
		ctx := &middleware.Context{Response: responseMsg}

		err := filter.Execute(ctx, func(c *middleware.Context) error {
			return nil
		})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("returns error for invalid response", func(t *testing.T) {
		filter := NewResponseFilter(func(msg *message.Message) error {
			if len(msg.Text()) > 100 {
				return errors.New("response too long")
			}
			return nil
		})

		longResponse := ""
		for i := 0; i < 101; i++ {
			longResponse += "a"
		}
		responseMsg := message.NewMessage(message.RoleAssistant, longResponse)
		ctx := &middleware.Context{Response: responseMsg}

		err := filter.Execute(ctx, func(c *middleware.Context) error {
			return nil
		})

		if err == nil {
			t.Error("expected error for long response")
		}
	})
}
