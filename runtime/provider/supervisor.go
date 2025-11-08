package provider

import (
	"context"
	"fmt"
	"sync"

	"github.com/sweetpotato0/ai-allin/tool"
)

// ToolSupervisor coordinates tool providers, ensuring their tools are registered and refreshed.
type ToolSupervisor struct {
	registry   *tool.Registry
	mu         sync.Mutex
	providers  []tool.Provider
	loaded     map[tool.Provider]bool
	watchers   map[tool.Provider]context.CancelFunc
	errHandler func(error)
}

// Option configures a ToolSupervisor.
type Option func(*ToolSupervisor)

// WithErrorHandler registers a callback for refresh failures.
func WithErrorHandler(handler func(error)) Option {
	return func(s *ToolSupervisor) {
		s.errHandler = handler
	}
}

// NewToolSupervisor constructs a ToolSupervisor bound to the provided registry.
func NewToolSupervisor(registry *tool.Registry, opts ...Option) *ToolSupervisor {
	if registry == nil {
		panic("runtime/provider: registry cannot be nil")
	}
	s := &ToolSupervisor{
		registry: registry,
		loaded:   make(map[tool.Provider]bool),
		watchers: make(map[tool.Provider]context.CancelFunc),
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(s)
	}
	return s
}

// Register adds a provider to the supervisor.
func (s *ToolSupervisor) Register(provider tool.Provider) {
	if provider == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.providers = append(s.providers, provider)
}

// Providers returns a copy of the registered providers.
func (s *ToolSupervisor) Providers() []tool.Provider {
	s.mu.Lock()
	defer s.mu.Unlock()
	copied := make([]tool.Provider, len(s.providers))
	copy(copied, s.providers)
	return copied
}

// Refresh ensures all providers are loaded and watchers started.
func (s *ToolSupervisor) Refresh(ctx context.Context) error {
	for _, provider := range s.Providers() {
		if provider == nil || s.isLoaded(provider) {
			continue
		}
		if err := s.updateProvider(ctx, provider); err != nil {
			return err
		}
		s.markLoaded(provider)
		s.startWatcher(provider)
	}
	return nil
}

// Close stops watchers and closes all providers.
func (s *ToolSupervisor) Close() error {
	s.mu.Lock()
	providers := make([]tool.Provider, len(s.providers))
	copy(providers, s.providers)
	for _, cancel := range s.watchers {
		cancel()
	}
	s.watchers = make(map[tool.Provider]context.CancelFunc)
	s.loaded = make(map[tool.Provider]bool)
	s.mu.Unlock()

	var firstErr error
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		if err := provider.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (s *ToolSupervisor) updateProvider(ctx context.Context, provider tool.Provider) error {
	tools, err := provider.Tools(ctx)
	if err != nil {
		return fmt.Errorf("runtime/provider: load tools: %w", err)
	}

	for _, t := range tools {
		if t == nil || t.Name == "" {
			continue
		}
		if err := s.registry.Upsert(t); err != nil {
			return err
		}
	}
	return nil
}

func (s *ToolSupervisor) startWatcher(provider tool.Provider) {
	ch := provider.ToolsChanged()
	if ch == nil {
		return
	}

	s.mu.Lock()
	if _, exists := s.watchers[provider]; exists {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.watchers[provider] = cancel
	s.mu.Unlock()

	go s.watch(ctx, provider, ch)
}

func (s *ToolSupervisor) watch(ctx context.Context, provider tool.Provider, ch <-chan struct{}) {
	defer s.stopWatcher(provider)

	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-ch:
			if !ok {
				return
			}
			if err := s.updateProvider(ctx, provider); err != nil {
				s.handleError(err)
			}
		}
	}
}

func (s *ToolSupervisor) stopWatcher(provider tool.Provider) {
	s.mu.Lock()
	if cancel, ok := s.watchers[provider]; ok {
		cancel()
		delete(s.watchers, provider)
	}
	s.mu.Unlock()
}

func (s *ToolSupervisor) isLoaded(provider tool.Provider) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loaded[provider]
}

func (s *ToolSupervisor) markLoaded(provider tool.Provider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loaded[provider] = true
}

func (s *ToolSupervisor) handleError(err error) {
	if err == nil || s.errHandler == nil {
		return
	}
	s.errHandler(err)
}
