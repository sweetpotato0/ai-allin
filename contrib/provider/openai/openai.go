package openai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/message"
)

// Config holds OpenAI provider configuration
type Config struct {
	APIKey      string
	BaseURL     string
	Model       string
	MaxTokens   int64
	Temperature float64
}

// WithBaseURL set BaseURL.
func (cfg *Config) WithBaseURL(url string) *Config {
	cfg.BaseURL = url
	return cfg
}

// WithAPIKey set api key.
func (cfg *Config) WithAPIKey(apiKey string) *Config {
	cfg.APIKey = apiKey
	return cfg
}

// DefaultConfig returns default OpenAI configuration
func DefaultConfig() *Config {
	return &Config{
		APIKey:      "",
		Model:       "gpt-4o-mini",
		MaxTokens:   2000,
		Temperature: 0.7,
	}
}

// Provider implements the LLMClient interface for OpenAI
type Provider struct {
	config *Config
	client openai.Client
}

// New creates a new OpenAI provider using official SDK
func New(config *Config) *Provider {
	if config.Model == "" {
		config.Model = "gpt-4o-mini"
	}

	options := []option.RequestOption{option.WithAPIKey(config.APIKey)}
	if config.BaseURL != "" {
		options = append(options, option.WithBaseURL(config.BaseURL))
	}
	client := openai.NewClient(options...)

	return &Provider{
		config: config,
		client: client,
	}
}

// Generate implements agent.LLMClient interface
func (p *Provider) Generate(ctx context.Context, messages []*message.Message, tools []map[string]any) (*message.Message, error) {
	// Convert messages to OpenAI format
	openAIMessages := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case message.RoleSystem:
			openAIMessages = append(openAIMessages, openai.SystemMessage(msg.Content))
		case message.RoleUser:
			openAIMessages = append(openAIMessages, openai.UserMessage(msg.Content))
		case message.RoleAssistant:
			openAIMessages = append(openAIMessages, openai.AssistantMessage(msg.Content))
		case message.RoleTool:
			openAIMessages = append(openAIMessages, openai.ToolMessage(msg.ToolID, msg.Content))
		}
	}

	// Build chat completion request
	params := openai.ChatCompletionNewParams{
		Messages: openAIMessages,
		Model:    openai.ChatModelGPT4oMini,
	}

	// Set temperature if provided
	if p.config.Temperature > 0 {
		params.Temperature = param.NewOpt(p.config.Temperature)
	}

	// Set max tokens if provided
	if p.config.MaxTokens > 0 {
		params.MaxCompletionTokens = param.NewOpt(p.config.MaxTokens)
	}

	// Add tools if provided
	if len(tools) > 0 {
		openAITools := make([]openai.ChatCompletionToolParam, 0, len(tools))
		for _, tool := range tools {
			// Convert tool schema
			toolJSON, err := json.Marshal(tool)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal tool: %w", err)
			}

			var toolParam openai.ChatCompletionToolParam
			if err := json.Unmarshal(toolJSON, &toolParam); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tool param: %w", err)
			}

			openAITools = append(openAITools, toolParam)
		}
		params.Tools = openAITools
	}

	// Call OpenAI API
	completion, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(completion.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from OpenAI")
	}

	// Extract response
	choice := completion.Choices[0]
	responseMsg := message.NewMessage(message.RoleAssistant, choice.Message.Content)

	// Handle tool calls if present
	if len(choice.Message.ToolCalls) > 0 {
		toolCalls := make([]message.ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			var args map[string]any
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				return nil, fmt.Errorf("failed to parse tool arguments: %w", err)
			}

			toolCalls[i] = message.ToolCall{
				ID:   tc.ID,
				Name: tc.Function.Name,
				Args: args,
			}
		}
		responseMsg.ToolCalls = toolCalls
	}

	return responseMsg, nil
}

// SetTemperature updates the temperature setting
func (p *Provider) SetTemperature(temp float64) {
	p.config.Temperature = temp
}

// SetMaxTokens updates the max tokens setting
func (p *Provider) SetMaxTokens(max int64) {
	p.config.MaxTokens = max
}

// SetModel updates the model
func (p *Provider) SetModel(model string) {
	p.config.Model = model
}

// GenerateStream implements agent.StreamLLMClient interface for streaming responses
func (p *Provider) GenerateStream(ctx context.Context, messages []*message.Message, tools []map[string]any, callback agent.StreamCallback) (*message.Message, error) {
	// Convert messages to OpenAI format
	openAIMessages := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case message.RoleSystem:
			openAIMessages = append(openAIMessages, openai.SystemMessage(msg.Content))
		case message.RoleUser:
			openAIMessages = append(openAIMessages, openai.UserMessage(msg.Content))
		case message.RoleAssistant:
			openAIMessages = append(openAIMessages, openai.AssistantMessage(msg.Content))
		case message.RoleTool:
			openAIMessages = append(openAIMessages, openai.ToolMessage(msg.ToolID, msg.Content))
		}
	}

	// Build chat completion request with streaming
	params := openai.ChatCompletionNewParams{
		Messages: openAIMessages,
		Model:    openai.ChatModelGPT4oMini,
	}

	// Set temperature if provided
	if p.config.Temperature > 0 {
		params.Temperature = param.NewOpt(p.config.Temperature)
	}

	// Set max tokens if provided
	if p.config.MaxTokens > 0 {
		params.MaxCompletionTokens = param.NewOpt(p.config.MaxTokens)
	}

	// Add tools if provided
	if len(tools) > 0 {
		openAITools := make([]openai.ChatCompletionToolParam, 0, len(tools))
		for _, tool := range tools {
			// Convert tool schema
			toolJSON, err := json.Marshal(tool)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal tool: %w", err)
			}

			var toolParam openai.ChatCompletionToolParam
			if err := json.Unmarshal(toolJSON, &toolParam); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tool param: %w", err)
			}

			openAITools = append(openAITools, toolParam)
		}
		params.Tools = openAITools
	}

	// Create streaming client
	stream := p.client.Chat.Completions.NewStreaming(ctx, params)
	defer stream.Close()

	var responseText string
	var accumulatedToolCalls []message.ToolCall

	// Process the stream
	for stream.Next() {
		event := stream.Current()

		if len(event.Choices) > 0 {
			choice := event.Choices[0]

			// Handle text delta
			if choice.Delta.Content != "" {
				responseText += choice.Delta.Content
				// Call the callback with the token
				if err := callback(choice.Delta.Content); err != nil {
					stream.Close()
					return nil, fmt.Errorf("callback error: %w", err)
				}
			}

			// Handle tool calls (if streaming provides them)
			if len(choice.Delta.ToolCalls) > 0 {
				for _, tc := range choice.Delta.ToolCalls {
					idx := tc.Index
					// Ensure we have enough space
					for len(accumulatedToolCalls) <= int(idx) {
						accumulatedToolCalls = append(accumulatedToolCalls, message.ToolCall{})
					}
					// Update tool call details
					if tc.ID != "" {
						accumulatedToolCalls[idx].ID = tc.ID
					}
					if tc.Function.Name != "" {
						accumulatedToolCalls[idx].Name = tc.Function.Name
					}
					if tc.Function.Arguments != "" {
						// Parse arguments
						var args map[string]any
						if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err == nil {
							accumulatedToolCalls[idx].Args = args
						}
					}
				}
			}
		}
	}

	if err := stream.Err(); err != nil {
		return nil, fmt.Errorf("OpenAI streaming error: %w", err)
	}

	// Create response message
	responseMsg := message.NewMessage(message.RoleAssistant, responseText)
	if len(accumulatedToolCalls) > 0 {
		responseMsg.ToolCalls = accumulatedToolCalls
	}

	return responseMsg, nil
}
