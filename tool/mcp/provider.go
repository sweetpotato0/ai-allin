package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/sweetpotato0/ai-allin/tool"
)

// Provider exposes MCP tools through the generic tool.Provider interface.
type Provider interface {
	tool.Provider
	// Client returns the underlying MCP client for advanced use cases.
	Client() *Client
}

// Transport enumerates the supported MCP transport types.
type Transport string

const (
	// TransportStreamable indicates the streamable HTTP (SSE) transport.
	TransportStreamable Transport = "streamable"
	// TransportCommand indicates the stdio/command transport.
	TransportCommand Transport = "command"
)

// Config describes how to connect to an MCP server.
type Config struct {
	// Transport selects how to connect to the MCP server. If empty, defaults to
	// streamable HTTP when Endpoint is provided, otherwise command transport.
	Transport Transport
	// Endpoint is required for streamable HTTP connections.
	Endpoint string
	// Command is required for command transport connections.
	Command string
}

type provider struct {
	client *Client
}

// NewProvider constructs a Provider based on the supplied configuration.
func NewProvider(ctx context.Context, cfg Config, opts ...Option) (Provider, error) {
	transport := cfg.Transport
	if transport == "" {
		if cfg.Command != "" {
			transport = TransportCommand
		} else {
			transport = TransportStreamable
		}
	}

	var (
		client *Client
		err    error
	)

	switch transport {
	case TransportStreamable:
		if strings.TrimSpace(cfg.Endpoint) == "" {
			return nil, errors.New("mcp: endpoint is required for streamable transport")
		}
		client, err = NewStreamableClient(ctx, cfg.Endpoint, opts...)
	case TransportCommand:
		if strings.TrimSpace(cfg.Command) == "" {
			return nil, errors.New("mcp: command is required for command transport")
		}
		client, err = NewStdioClient(ctx, cfg.Command, opts...)
	default:
		return nil, fmt.Errorf("mcp: unsupported transport %q", transport)
	}
	if err != nil {
		return nil, err
	}

	p := &provider{client: client}
	// Fail fast if we cannot list tools.
	if _, err := p.Tools(ctx); err != nil {
		_ = client.Close()
		return nil, err
	}

	return p, nil
}

func (p *provider) Tools(ctx context.Context) ([]*tool.Tool, error) {
	if p == nil || p.client == nil {
		return nil, errors.New("mcp: provider is not initialized")
	}
	return p.client.BuildTools(ctx)
}

func (p *provider) Close() error {
	if p == nil || p.client == nil {
		return nil
	}
	return p.client.Close()
}

func (p *provider) Client() *Client {
	if p == nil {
		return nil
	}
	return p.client
}

func (p *provider) ToolsChanged() <-chan struct{} {
	if p == nil || p.client == nil {
		return nil
	}
	return p.client.ToolsChanged()
}
