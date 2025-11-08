package provider

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/sweetpotato0/ai-allin/tool"
)

func TestSupervisorRefreshRegistersTools(t *testing.T) {
	registry := tool.NewRegistry()
	sup := NewToolSupervisor(registry, WithErrorHandler(func(err error) {
		t.Fatalf("unexpected error: %v", err)
	}))

	provider := &stubProvider{
		tools: []*tool.Tool{
			{Name: "echo"},
		},
	}

	sup.Register(provider)
	if err := sup.Refresh(context.Background()); err != nil {
		t.Fatalf("refresh failed: %v", err)
	}

	if _, err := registry.Get("echo"); err != nil {
		t.Fatalf("tool not registered: %v", err)
	}
}

func TestSupervisorHandlesWatcherUpdates(t *testing.T) {
	registry := tool.NewRegistry()
	sup := NewToolSupervisor(registry)

	ch := make(chan struct{}, 1)
	provider := &stubProvider{
		tools: []*tool.Tool{{Name: "foo"}},
		ch:    ch,
	}

	sup.Register(provider)
	if err := sup.Refresh(context.Background()); err != nil {
		t.Fatalf("refresh failed: %v", err)
	}

	provider.setTools([]*tool.Tool{{Name: "bar"}})
	ch <- struct{}{}

	waitForCondition(t, time.Second, func() bool {
		_, err := registry.Get("bar")
		return err == nil
	})
}

func TestSupervisorCloseStopsProviders(t *testing.T) {
	registry := tool.NewRegistry()
	sup := NewToolSupervisor(registry)

	provider := &stubProvider{}
	sup.Register(provider)
	if err := sup.Refresh(context.Background()); err != nil {
		t.Fatalf("refresh failed: %v", err)
	}

	if err := sup.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	if !provider.closed {
		t.Fatalf("expected provider to be closed")
	}
}

type stubProvider struct {
	mu     sync.Mutex
	tools  []*tool.Tool
	ch     chan struct{}
	closed bool
}

func (p *stubProvider) Tools(ctx context.Context) ([]*tool.Tool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	result := make([]*tool.Tool, len(p.tools))
	copy(result, p.tools)
	return result, nil
}

func (p *stubProvider) Close() error {
	p.mu.Lock()
	p.closed = true
	p.mu.Unlock()
	return nil
}

func (p *stubProvider) ToolsChanged() <-chan struct{} {
	return p.ch
}

func (p *stubProvider) setTools(tools []*tool.Tool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.tools = tools
}

func waitForCondition(t *testing.T, timeout time.Duration, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", timeout)
}
