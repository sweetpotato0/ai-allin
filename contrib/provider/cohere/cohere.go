package cohere

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/message"
)

const cohereAPIURL = "https://api.cohere.ai/v1/chat"

// Config holds Cohere provider configuration
type Config struct {
	APIKey      string
	Model       string
	MaxTokens   int
	Temperature float64
}

// DefaultConfig returns default Cohere configuration
func DefaultConfig(apiKey string) *Config {
	return &Config{
		APIKey:      apiKey,
		Model:       "command",
		MaxTokens:   2048,
		Temperature: 0.7,
	}
}

var _ agent.LLMClient = (*Provider)(nil)

// Provider implements the LLMClient interface for Cohere
type Provider struct {
	config *Config
	client *http.Client
}

// New creates a new Cohere provider
func New(config *Config) *Provider {
	if config == nil {
		config = DefaultConfig("")
	}

	if config.Model == "" {
		config.Model = "command"
	}

	return &Provider{
		config: config,
		client: &http.Client{},
	}
}

// cohereMessage represents a message in Cohere API format
type cohereMessage struct {
	Role    string `json:"role"`
	Message string `json:"message"`
}

// cohereRequest represents a Cohere API request
type cohereRequest struct {
	Model          string          `json:"model"`
	Messages       []cohereMessage `json:"messages"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	Temperature    float64         `json:"temperature,omitempty"`
	ConversationID string          `json:"conversation_id,omitempty"`
}

// cohereResponse represents a Cohere API response
type cohereResponse struct {
	Text           string       `json:"text"`
	ConversationID string       `json:"conversation_id"`
	Error          *cohereError `json:"error,omitempty"`
}

// cohereError represents an error in Cohere API response
type cohereError struct {
	Message string `json:"message"`
}

// Generate implements agent.LLMClient interface
func (p *Provider) Generate(ctx context.Context, req *agent.GenerateRequest) (*agent.GenerateResponse, error) {
	if p.config.APIKey == "" {
		return nil, fmt.Errorf("Cohere API key not configured")
	}
	if req == nil {
		return nil, fmt.Errorf("generate request cannot be nil")
	}

	// Convert messages to Cohere format
	cohereMessages := make([]cohereMessage, len(req.Messages))
	for i, msg := range req.Messages {
		cohereMessages[i] = cohereMessage{
			Role:    string(msg.Role),
			Message: msg.Text(),
		}
	}

	// Create request
	payload := cohereRequest{
		Model:       p.config.Model,
		Messages:    cohereMessages,
		MaxTokens:   p.config.MaxTokens,
		Temperature: p.config.Temperature,
	}

	// Marshal request
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", cohereAPIURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.APIKey))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "ai-allin-client")

	// Send request
	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Cohere API error (status %d): %s", httpResp.StatusCode, string(respBody))
	}

	// Unmarshal response
	var resp cohereResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check for API error
	if resp.Error != nil {
		return nil, fmt.Errorf("Cohere API error: %s", resp.Error.Message)
	}

	msg := message.NewMessage(message.RoleAssistant, resp.Text)
	msg.Completed = true
	return &agent.GenerateResponse{Message: msg}, nil
}

// SetTemperature updates the temperature setting
func (p *Provider) SetTemperature(temp float64) {
	p.config.Temperature = temp
}

// SetMaxTokens updates the max tokens setting
func (p *Provider) SetMaxTokens(max int64) {
	p.config.MaxTokens = int(max)
}

// SetModel updates the model
func (p *Provider) SetModel(model string) {
	p.config.Model = model
}
