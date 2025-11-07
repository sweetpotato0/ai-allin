package groq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sweetpotato0/ai-allin/message"
)

const groqAPIURL = "https://api.groq.com/openai/v1/chat/completions"

// Config holds Groq provider configuration
type Config struct {
	APIKey      string
	Model       string
	MaxTokens   int
	Temperature float64
}

// DefaultConfig returns default Groq configuration
func DefaultConfig(apiKey string) *Config {
	return &Config{
		APIKey:      apiKey,
		Model:       "mixtral-8x7b-32768",
		MaxTokens:   2048,
		Temperature: 0.7,
	}
}

// Provider implements the LLMClient interface for Groq
type Provider struct {
	config *Config
	client *http.Client
}

// New creates a new Groq provider
func New(config *Config) *Provider {
	if config == nil {
		config = DefaultConfig("")
	}

	if config.Model == "" {
		config.Model = "mixtral-8x7b-32768"
	}

	return &Provider{
		config: config,
		client: &http.Client{},
	}
}

// groqMessage represents a message in Groq API format
type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// groqRequest represents a Groq API request
type groqRequest struct {
	Model       string        `json:"model"`
	Messages    []groqMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature"`
}

// groqChoice represents a choice in Groq API response
type groqChoice struct {
	Message groqMessage `json:"message"`
}

// groqResponse represents a Groq API response
type groqResponse struct {
	Choices []groqChoice `json:"choices"`
	Error   *groqError   `json:"error,omitempty"`
}

// groqError represents an error in Groq API response
type groqError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// Generate implements agent.LLMClient interface
func (p *Provider) Generate(ctx context.Context, messages []*message.Message, tools []map[string]any) (*message.Message, error) {
	if p.config.APIKey == "" {
		return nil, fmt.Errorf("Groq API key not configured")
	}

	// Convert messages to Groq format
	groqMessages := make([]groqMessage, len(messages))
	for i, msg := range messages {
		groqMessages[i] = groqMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	// Create request
	req := groqRequest{
		Model:       p.config.Model,
		Messages:    groqMessages,
		MaxTokens:   p.config.MaxTokens,
		Temperature: p.config.Temperature,
	}

	// Marshal request
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", groqAPIURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.APIKey))
	httpReq.Header.Set("Content-Type", "application/json")

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
		return nil, fmt.Errorf("Groq API error (status %d): %s", httpResp.StatusCode, string(respBody))
	}

	// Unmarshal response
	var resp groqResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check for API error
	if resp.Error != nil {
		return nil, fmt.Errorf("Groq API error: %s", resp.Error.Message)
	}

	// Extract message
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return message.NewMessage(message.RoleAssistant, resp.Choices[0].Message.Content), nil
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
