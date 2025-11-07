package claude

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/message"
)

// Config holds Claude provider configuration
type Config struct {
	APIKey      string
	Model       string
	BaseURL     string
	MaxTokens   int64
	Temperature float64
}

// DefaultConfig returns default Claude configuration
func DefaultConfig(apiKey, baseURL string) *Config {
	return &Config{
		APIKey:      apiKey,
		BaseURL:     baseURL,
		Model:       "claude-3-5-sonnet-20241022",
		MaxTokens:   4096,
		Temperature: 0.7,
	}
}

// Provider implements the LLMClient interface for Claude
type Provider struct {
	config *Config
	client anthropic.Client
}

// New creates a new Claude provider using official SDK
func New(config *Config) *Provider {
	if config.Model == "" {
		config.Model = "claude-sonnet-4-5-20250929"
	}

	options := []option.RequestOption{
		option.WithAPIKey(config.APIKey),
		option.WithAuthToken(""),
	}

	if config.BaseURL != "" {
		options = append(options, option.WithBaseURL(config.BaseURL))
	}

	client := anthropic.NewClient(options...)

	return &Provider{
		config: config,
		client: client,
	}
}

// Generate implements agent.LLMClient interface
func (p *Provider) Generate(ctx context.Context, messages []*message.Message, tools []map[string]any) (*message.Message, error) {
	// Separate system messages from conversation
	var systemPrompts []string
	conversationMessages := make([]anthropic.MessageParam, 0)

	for _, msg := range messages {
		if msg.Role == message.RoleSystem {
			systemPrompts = append(systemPrompts, msg.Content)
		} else {
			switch msg.Role {
			case message.RoleUser:
				conversationMessages = append(conversationMessages,
					anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content)))
			case message.RoleAssistant:
				conversationMessages = append(conversationMessages,
					anthropic.NewAssistantMessage(anthropic.NewTextBlock(msg.Content)))
			}
		}
	}

	// Build message creation params
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(p.config.Model),
		Messages:  conversationMessages,
		MaxTokens: p.config.MaxTokens,
	}

	// Add system prompts if present
	if len(systemPrompts) > 0 {
		systemText := ""
		for i, sp := range systemPrompts {
			if i > 0 {
				systemText += "\n"
			}
			systemText += sp
		}
		params.System = []anthropic.TextBlockParam{
			{Text: systemText},
		}
	}

	// Add temperature if set
	if p.config.Temperature > 0 {
		params.Temperature = param.NewOpt(p.config.Temperature)
	}

	// Add tools if provided
	if len(tools) > 0 {
		claudeTools := make([]anthropic.ToolUnionParam, 0, len(tools))
		for _, tool := range tools {
			toolJSON, err := json.Marshal(tool)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal tool: %w", err)
			}

			var toolParam anthropic.ToolParam
			if err := json.Unmarshal(toolJSON, &toolParam); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tool param: %w", err)
			}

			// Wrap ToolParam into ToolUnionParam
			unionParam := anthropic.ToolUnionParam{
				OfTool: &toolParam,
			}
			claudeTools = append(claudeTools, unionParam)
		}
		params.Tools = claudeTools
	}

	// Call Claude API
	apiMessage, err := p.client.Messages.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("Claude API error: %w", err)
	}

	// Extract text and tool uses from content blocks
	var responseText string
	toolCalls := make([]message.ToolCall, 0)

	for _, content := range apiMessage.Content {
		// Check type field to determine content type
		switch content.Type {
		case "text":
			responseText = content.Text
		case "tool_use":
			var args map[string]any
			if err := json.Unmarshal(content.Input, &args); err != nil {
				return nil, fmt.Errorf("failed to parse tool input: %w", err)
			}

			toolCalls = append(toolCalls, message.ToolCall{
				ID:   content.ID,
				Name: content.Name,
				Args: args,
			})
		}
	}

	// Create response message
	responseMsg := message.NewMessage(message.RoleAssistant, responseText)
	if len(toolCalls) > 0 {
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
	// Separate system messages from conversation
	var systemPrompts []string
	conversationMessages := make([]anthropic.MessageParam, 0, len(messages))

	for _, msg := range messages {
		if msg.Role == message.RoleSystem {
			systemPrompts = append(systemPrompts, msg.Content)
		} else {
			switch msg.Role {
			case message.RoleUser:
				conversationMessages = append(conversationMessages,
					anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content)))
			case message.RoleAssistant:
				conversationMessages = append(conversationMessages,
					anthropic.NewAssistantMessage(anthropic.NewTextBlock(msg.Content)))
			}
		}
	}

	// Build message creation params (no Stream field in params)
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(p.config.Model),
		Messages:  conversationMessages,
		MaxTokens: p.config.MaxTokens,
	}

	// Add system prompts if present
	if len(systemPrompts) > 0 {
		systemText := ""
		for i, sp := range systemPrompts {
			if i > 0 {
				systemText += "\n"
			}
			systemText += sp
		}
		params.System = []anthropic.TextBlockParam{
			{Text: systemText},
		}
	}

	// Add temperature if set
	if p.config.Temperature > 0 {
		params.Temperature = param.NewOpt(p.config.Temperature)
	}

	// Add tools if provided
	if len(tools) > 0 {
		claudeTools := make([]anthropic.ToolUnionParam, 0, len(tools))
		for _, tool := range tools {
			toolJSON, err := json.Marshal(tool)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal tool: %w", err)
			}

			var toolParam anthropic.ToolParam
			if err := json.Unmarshal(toolJSON, &toolParam); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tool param: %w", err)
			}

			// Wrap ToolParam into ToolUnionParam
			unionParam := anthropic.ToolUnionParam{
				OfTool: &toolParam,
			}
			claudeTools = append(claudeTools, unionParam)
		}
		params.Tools = claudeTools
	}

	// Create streaming client
	stream := p.client.Messages.NewStreaming(ctx, params)
	defer stream.Close()

	var responseText string
	var toolCalls []message.ToolCall
	var finalMessage *anthropic.Message

	// Process the stream
	for stream.Next() {
		event := stream.Current()

		// Handle different event types
		switch event.Type {
		case "content_block_delta":
			// Text token delta
			contentDelta := event.AsContentBlockDelta()
			if contentDelta.Delta.Type == "text_delta" {
				if contentDelta.Delta.Text != "" {
					responseText += contentDelta.Delta.Text
					// Call the callback with the text
					if err := callback(contentDelta.Delta.Text); err != nil {
						stream.Close()
						return nil, fmt.Errorf("callback error: %w", err)
					}
				}
			}
		case "message_start":
			// Message started - capture initial message structure
			msgStart := event.AsMessageStart()
			finalMessage = &msgStart.Message
		case "message_stop":
			// Message stopped - end of stream
		}
	}

	if err := stream.Err(); err != nil {
		return nil, fmt.Errorf("Claude streaming error: %w", err)
	}

	// Extract tool uses from final message content
	if finalMessage != nil {
		for _, content := range finalMessage.Content {
			if content.Type == "tool_use" {
				var args map[string]any
				if err := json.Unmarshal(content.Input, &args); err == nil {
					toolCalls = append(toolCalls, message.ToolCall{
						ID:   content.ID,
						Name: content.Name,
						Args: args,
					})
				}
			}
		}
	}

	// Create response message
	responseMsg := message.NewMessage(message.RoleAssistant, responseText)
	if len(toolCalls) > 0 {
		responseMsg.ToolCalls = toolCalls
	}

	return responseMsg, nil
}
