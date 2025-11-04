package middleware

import (
	"context"
	"errors"
	"testing"
)

func TestMiddlewareChain(t *testing.T) {
	t.Run("empty chain executes final handler", func(t *testing.T) {
		chain := NewChain()
		executed := false

		err := chain.Execute(&Context{}, func(ctx *Context) error {
			executed = true
			return nil
		})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !executed {
			t.Error("final handler was not executed")
		}
	})

	t.Run("middleware chain executes in order", func(t *testing.T) {
		order := []string{}

		m1 := &TestMiddleware{name: "m1", order: &order}
		m2 := &TestMiddleware{name: "m2", order: &order}

		chain := NewChain(m1, m2)
		ctx := &Context{}

		_ = chain.Execute(ctx, func(c *Context) error {
			order = append(order, "final")
			return nil
		})

		expected := []string{"m1", "m2", "final"}
		if len(order) != len(expected) {
			t.Errorf("expected %d steps, got %d", len(expected), len(order))
		}
		for i, e := range expected {
			if i >= len(order) || order[i] != e {
				t.Errorf("expected step %d to be %s, got %s", i, e, order[i])
			}
		}
	})

	t.Run("error stops chain execution", func(t *testing.T) {
		order := []string{}
		m1 := &TestMiddleware{name: "m1", err: errors.New("test error"), order: &order}
		m2 := &TestMiddleware{name: "m2", order: &order}

		chain := NewChain(m1, m2)
		ctx := &Context{}

		finalCalled := false
		err := chain.Execute(ctx, func(c *Context) error {
			finalCalled = true
			return nil
		})

		if err == nil {
			t.Error("expected error from middleware")
		}
		if finalCalled {
			t.Error("final handler should not be called after middleware error")
		}
	})
}

func TestContext(t *testing.T) {
	t.Run("new context has empty metadata", func(t *testing.T) {
		ctx := NewContext(context.Background())
		if ctx.Metadata == nil {
			t.Error("metadata should not be nil")
		}
		if len(ctx.Metadata) != 0 {
			t.Error("metadata should be empty")
		}
	})

	t.Run("context preserves underlying context", func(t *testing.T) {
		baseCtx := context.Background()
		ctx := NewContext(baseCtx)
		if ctx.Context() != baseCtx {
			t.Error("underlying context not preserved")
		}
	})
}

// Helper test middleware
type TestMiddleware struct {
	name  string
	order *[]string
	err   error
}

func (m *TestMiddleware) Name() string {
	return m.name
}

func (m *TestMiddleware) Execute(ctx *Context, next Handler) error {
	*m.order = append(*m.order, m.name)
	if m.err != nil {
		return m.err
	}
	return next(ctx)
}
