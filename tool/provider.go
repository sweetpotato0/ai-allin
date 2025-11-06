package tool

import "context"

// Provider supplies tools that can be registered with an agent.
type Provider interface {
	// Tools returns the provider's current tool definitions.
	Tools(ctx context.Context) ([]*Tool, error)
	// Close releases resources owned by the provider.
	Close() error
}
