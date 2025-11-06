package tool

import "context"

// Provider supplies tools that can be registered with an agent.
type Provider interface {
	// Tools returns the provider's current tool definitions.
	Tools(ctx context.Context) ([]*Tool, error)
	// Close releases resources owned by the provider.
	Close() error
	// ToolsChanged returns a channel that fires when the tool set is updated.
	// Providers that do not support live updates should return nil.
	ToolsChanged() <-chan struct{}
}
