package enricher

import (
	"errors"
	"testing"

	"github.com/sweetpotato0/ai-allin/middleware"
)

func TestContextEnricher(t *testing.T) {
	t.Run("enriches context with metadata", func(t *testing.T) {
		enricher := NewContextEnricher(func(ctx *middleware.Context) error {
			ctx.Metadata["key"] = "value"
			return nil
		})

		ctx := &middleware.Context{Metadata: map[string]any{}}
		err := enricher.Execute(ctx, func(c *middleware.Context) error { return nil })

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if ctx.Metadata["key"] != "value" {
			t.Error("metadata not enriched")
		}
	})

	t.Run("returns error if enricher fails", func(t *testing.T) {
		enricher := NewContextEnricher(func(ctx *middleware.Context) error {
			return errors.New("enrichment failed")
		})

		ctx := &middleware.Context{Metadata: map[string]any{}}
		err := enricher.Execute(ctx, func(c *middleware.Context) error { return nil })

		if err == nil {
			t.Error("expected error from enricher")
		}
	})

	t.Run("handles nil enricher function", func(t *testing.T) {
		enricher := NewContextEnricher(nil)

		ctx := &middleware.Context{Metadata: map[string]any{}}
		err := enricher.Execute(ctx, func(c *middleware.Context) error { return nil })

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}
