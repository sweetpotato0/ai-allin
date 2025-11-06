package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	// ErrClientClosed is returned when the MCP client has been closed.
	ErrClientClosed = errors.New("mcp client closed")
)

// Option configures optional MCP client behaviour.
type Option func(*clientConfig)

type clientConfig struct {
	implementation    sdkmcp.Implementation
	logger            *log.Logger
	args              []string
	env               []string
	dir               string
	keepAlive         time.Duration
	terminateTimeout  time.Duration
	httpClient        *http.Client
	streamableRetries *int
}

// WithClientInfo sets the client metadata advertised to the MCP server.
func WithClientInfo(info ClientInfo) Option {
	return func(cfg *clientConfig) {
		if info.Name != "" {
			cfg.implementation.Name = info.Name
		}
		if info.Title != "" {
			cfg.implementation.Title = info.Title
		}
		if info.Version != "" {
			cfg.implementation.Version = info.Version
		}
	}
}

// WithLogger configures logging for the MCP client. If nil, logging is discarded.
func WithLogger(logger *log.Logger) Option {
	return func(cfg *clientConfig) {
		cfg.logger = logger
	}
}

// WithCommandArgs configures additional arguments when launching an stdio MCP server.
func WithCommandArgs(args ...string) Option {
	return func(cfg *clientConfig) {
		cfg.args = append(cfg.args, args...)
	}
}

// WithCommandEnv appends environment variables when launching an stdio MCP server.
func WithCommandEnv(env ...string) Option {
	return func(cfg *clientConfig) {
		cfg.env = append(cfg.env, env...)
	}
}

// WithCommandDir sets the working directory for the stdio MCP server process.
func WithCommandDir(dir string) Option {
	return func(cfg *clientConfig) {
		cfg.dir = dir
	}
}

// WithKeepAlive configures periodic ping requests to keep the session healthy.
func WithKeepAlive(interval time.Duration) Option {
	return func(cfg *clientConfig) {
		cfg.keepAlive = interval
	}
}

// WithTerminateTimeout sets how long to wait for graceful server shutdown before sending SIGTERM.
func WithTerminateTimeout(d time.Duration) Option {
	return func(cfg *clientConfig) {
		cfg.terminateTimeout = d
	}
}

// WithHTTPClient supplies a custom HTTP client for streamable (SSE/HTTP) transports.
func WithHTTPClient(client *http.Client) Option {
	return func(cfg *clientConfig) {
		cfg.httpClient = client
	}
}

// WithStreamableMaxRetries overrides the retry count for reconnect attempts when using
// the streamable HTTP transport.
func WithStreamableMaxRetries(retries int) Option {
	return func(cfg *clientConfig) {
		cfg.streamableRetries = &retries
	}
}

// ClientInfo describes the client metadata sent to the MCP server.
type ClientInfo struct {
	Name    string `json:"name"`
	Title   string `json:"title,omitempty"`
	Version string `json:"version"`
}

// ServerInfo contains information about the connected MCP server.
type ServerInfo struct {
	Name    string `json:"name"`
	Title   string `json:"title,omitempty"`
	Version string `json:"version"`
}

// InitializeResult captures the server response during MCP initialization.
type InitializeResult struct {
	ProtocolVersion string
	Capabilities    map[string]any
	ServerInfo      ServerInfo
	Instructions    string
}

// Client wraps the official MCP Go SDK client and session.
type Client struct {
	sdkClient *sdkmcp.Client
	session   *sdkmcp.ClientSession

	logger *log.Logger

	toolsChanged chan struct{}
	done         chan struct{}

	closeOnce sync.Once
	closeErr  error

	initialize *sdkmcp.InitializeResult
}

// NewStdioClient launches an MCP server command using the stdio transport and performs
// the initialization handshake.
func NewStdioClient(ctx context.Context, command string, opts ...Option) (*Client, error) {
	if command == "" {
		return nil, errors.New("mcp: command cannot be empty")
	}

	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	cmd := exec.Command(command, cfg.args...)
	if cfg.dir != "" {
		cmd.Dir = cfg.dir
	}
	if len(cfg.env) > 0 {
		cmd.Env = append(os.Environ(), cfg.env...)
	}
	cmd.Stderr = logWriter{logger: cfg.logger}

	client := &Client{
		logger:       cfg.logger,
		toolsChanged: make(chan struct{}, 1),
		done:         make(chan struct{}),
	}

	clientOpts := &sdkmcp.ClientOptions{
		ToolListChangedHandler: func(context.Context, *sdkmcp.ToolListChangedRequest) {
			select {
			case client.toolsChanged <- struct{}{}:
			default:
			}
		},
		LoggingMessageHandler: func(_ context.Context, req *sdkmcp.LoggingMessageRequest) {
			if client.logger != nil && req != nil && req.Params != nil {
				client.logger.Printf("mcp server log [%s]: %v", req.Params.Level, req.Params.Data)
			}
		},
		KeepAlive: cfg.keepAlive,
	}

	client.sdkClient = sdkmcp.NewClient(&cfg.implementation, clientOpts)

	transport := &sdkmcp.CommandTransport{
		Command:           cmd,
		TerminateDuration: cfg.terminateTimeout,
	}

	session, err := client.sdkClient.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("mcp: connect failed: %w", err)
	}
	client.session = session
	client.initialize = session.InitializeResult()

	go client.monitorSession()

	return client, nil
}

// NewStreamableClient connects to an MCP server over the streamable HTTP transport
// (SSE + HTTP POST) as defined by the MCP specification.
func NewStreamableClient(ctx context.Context, endpoint string, opts ...Option) (*Client, error) {
	if strings.TrimSpace(endpoint) == "" {
		return nil, errors.New("mcp: endpoint cannot be empty")
	}

	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	client := &Client{
		logger:       cfg.logger,
		toolsChanged: make(chan struct{}, 1),
		done:         make(chan struct{}),
	}

	clientOpts := &sdkmcp.ClientOptions{
		ToolListChangedHandler: func(context.Context, *sdkmcp.ToolListChangedRequest) {
			select {
			case client.toolsChanged <- struct{}{}:
			default:
			}
		},
		LoggingMessageHandler: func(_ context.Context, req *sdkmcp.LoggingMessageRequest) {
			if client.logger != nil && req != nil && req.Params != nil {
				client.logger.Printf("mcp server log [%s]: %v", req.Params.Level, req.Params.Data)
			}
		},
		KeepAlive: cfg.keepAlive,
	}

	client.sdkClient = sdkmcp.NewClient(&cfg.implementation, clientOpts)

	transport := &sdkmcp.StreamableClientTransport{
		Endpoint: endpoint,
	}
	if cfg.httpClient != nil {
		transport.HTTPClient = cfg.httpClient
	}
	if cfg.streamableRetries != nil {
		transport.MaxRetries = *cfg.streamableRetries
	}

	session, err := client.sdkClient.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("mcp: connect failed: %w", err)
	}
	client.session = session
	client.initialize = session.InitializeResult()

	go client.monitorSession()

	return client, nil
}

// Close terminates the MCP client and underlying transport.
func (c *Client) Close() error {
	c.closeOnce.Do(func() {
		if c.session != nil {
			c.closeErr = c.session.Close()
		}
		close(c.done)
	})
	return c.closeErr
}

// Done returns a channel that is closed when the client shuts down.
func (c *Client) Done() <-chan struct{} {
	return c.done
}

// ToolsChanged reports when the server indicates that the tool list has changed.
func (c *Client) ToolsChanged() <-chan struct{} {
	return c.toolsChanged
}

func (c *Client) monitorSession() {
	if c.session == nil {
		close(c.done)
		return
	}
	if err := c.session.Wait(); err != nil && !errors.Is(err, sdkmcp.ErrConnectionClosed) {
		if c.logger != nil && err != nil {
			c.logger.Printf("mcp: session ended with error: %v", err)
		}
	}
	_ = c.Close()
}

func defaultConfig() clientConfig {
	return clientConfig{
		implementation: sdkmcp.Implementation{
			Name:    "ai-allin",
			Version: "0.1.0",
		},
		logger: log.New(io.Discard, "", 0),
	}
}

type logWriter struct {
	logger *log.Logger
}

func (w logWriter) Write(p []byte) (int, error) {
	if w.logger != nil {
		msg := strings.TrimSpace(string(p))
		if msg != "" {
			w.logger.Printf("mcp server stderr: %s", msg)
		}
	}
	return len(p), nil
}

// InitializeResult returns the negotiated initialization metadata, if available.
func (c *Client) InitializeResult() *InitializeResult {
	if c.initialize == nil {
		return nil
	}
	return convertInitializeResult(c.initialize)
}

func convertInitializeResult(res *sdkmcp.InitializeResult) *InitializeResult {
	if res == nil {
		return nil
	}

	capabilities := map[string]any{}
	if res.Capabilities != nil {
		if data, err := json.Marshal(res.Capabilities); err == nil {
			_ = json.Unmarshal(data, &capabilities)
		}
	}

	server := ServerInfo{}
	if res.ServerInfo != nil {
		server = ServerInfo{
			Name:    res.ServerInfo.Name,
			Title:   res.ServerInfo.Title,
			Version: res.ServerInfo.Version,
		}
	}

	return &InitializeResult{
		ProtocolVersion: res.ProtocolVersion,
		Capabilities:    capabilities,
		ServerInfo:      server,
		Instructions:    res.Instructions,
	}
}
