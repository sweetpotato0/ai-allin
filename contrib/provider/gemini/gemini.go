package gemini

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

const geminiAPIURL = "https://generativelanguage.googleapis.com/v1/models"

// Config holds Gemini provider configuration
type Config struct {
	APIKey      string
	Model       string
	MaxTokens   int
	Temperature float32
}

// DefaultConfig returns default Gemini configuration
func DefaultConfig(apiKey string) *Config {
	return &Config{
		APIKey:      apiKey,
		Model:       "gemini-pro",
		MaxTokens:   2048,
		Temperature: 0.7,
	}
}

// Provider implements the LLMClient interface for Google Gemini
type Provider struct {
	config *Config
	client *http.Client
}

// New creates a new Gemini provider
func New(config *Config) *Provider {
	if config == nil {
		config = DefaultConfig("")
	}

	if config.Model == "" {
		config.Model = "gemini-pro"
	}

	return &Provider{
		config: config,
		client: &http.Client{},
	}
}

// geminiMessage represents a message in Gemini API format
type geminiMessage struct {
	Role  string `json:"role"`
	Parts []struct {
		Text string `json:"text"`
	} `json:"parts"`
}

// geminiRequest represents a Gemini API request
type geminiRequest struct {
	Contents    []geminiMessage `json:"contents"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float32         `json:"temperature,omitempty"`
}

// geminiResponse represents a Gemini API response
type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *geminiError `json:"error,omitempty"`
}

// geminiError represents an error in Gemini API response
type geminiError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// Generate implements agent.LLMClient interface
func (p *Provider) Generate(ctx context.Context, req *agent.GenerateRequest) (*agent.GenerateResponse, error) {
	if p.config.APIKey == "" {
		return nil, fmt.Errorf("Gemini API key not configured")
	}
	if req == nil {
		return nil, fmt.Errorf("generate request cannot be nil")
	}

	// Convert messages to Gemini format
	geminiMessages := make([]geminiMessage, len(req.Messages))
	for i, msg := range req.Messages {
		geminiMessages[i] = geminiMessage{
			Role: string(msg.Role),
			Parts: []struct {
				Text string `json:"text"`
			}{
				{Text: msg.Text()},
			},
		}
	}

	// Create request
	payload := geminiRequest{
		Contents:    geminiMessages,
		MaxTokens:   p.config.MaxTokens,
		Temperature: p.config.Temperature,
	}

	// Marshal request
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request URL
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", geminiAPIURL, p.config.Model, p.config.APIKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
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
		return nil, fmt.Errorf("Gemini API error (status %d): %s", httpResp.StatusCode, string(respBody))
	}

	// Unmarshal response
	var resp geminiResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check for API error
	if resp.Error != nil {
		return nil, fmt.Errorf("Gemini API error (code %d): %s", resp.Error.Code, resp.Error.Message)
	}

	// Extract message
	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	if len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content parts in candidate")
	}

	msg := message.NewMessage(message.RoleAssistant, resp.Candidates[0].Content.Parts[0].Text)
	msg.Completed = true
	return &agent.GenerateResponse{
		Message: msg,
	}, nil
}

// SetTemperature updates the temperature setting
func (p *Provider) SetTemperature(temp float64) {
	p.config.Temperature = float32(temp)
}

// SetMaxTokens updates the max tokens setting
func (p *Provider) SetMaxTokens(max int64) {
	p.config.MaxTokens = int(max)
}

// SetModel updates the model
func (p *Provider) SetModel(model string) {
	p.config.Model = model
}
