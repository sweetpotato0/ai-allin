package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"

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

// WithBaseURL set BaseURL.
func (cfg *Config) WithBaseURL(url string) *Config {
	cfg.BaseURL = url
	return cfg
}

// WithModel set model.
func (cfg *Config) WithModel(model string) *Config {
	cfg.Model = model
	return cfg
}

// WithAPIKey set api key.
func (cfg *Config) WithAPIKey(apiKey string) *Config {
	cfg.APIKey = apiKey
	return cfg
}

// DefaultConfig returns default Claude configuration
func DefaultConfig() *Config {
	return &Config{
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

var _ agent.LLMClient = (*Provider)(nil)

// Generate implements agent.LLMClient interface
func (p *Provider) Generate(ctx context.Context, req *agent.GenerateRequest) (*agent.GenerateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("generate request cannot be nil")
	}
	// Separate system messages from conversation
	var systemPrompts []string
	conversationMessages := make([]anthropic.MessageParam, 0)

	for _, msg := range req.Messages {
		if msg.Role == message.RoleSystem {
			systemPrompts = append(systemPrompts, msg.Text())
		} else {
			switch msg.Role {
			case message.RoleUser:
				conversationMessages = append(conversationMessages,
					anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Text())))
			case message.RoleAssistant:
				conversationMessages = append(conversationMessages,
					anthropic.NewAssistantMessage(anthropic.NewTextBlock(msg.Text())))
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
	if len(req.Tools) > 0 {
		claudeTools := make([]anthropic.ToolUnionParam, 0, len(req.Tools))
		for _, tool := range req.Tools {
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

	responseMsg.Completed = true
	return &agent.GenerateResponse{Message: responseMsg}, nil
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
func (p *Provider) GenerateStream(ctx context.Context, req *agent.GenerateRequest) iter.Seq2[*message.Message, error] {
	return func(yield func(*message.Message, error) bool) {
		if req == nil {
			yield(nil, fmt.Errorf("stream request cannot be nil"))
			return
		}

		var systemPrompts []string
		conversationMessages := make([]anthropic.MessageParam, 0, len(req.Messages))

		for _, msg := range req.Messages {
			if msg.Role == message.RoleSystem {
				systemPrompts = append(systemPrompts, msg.Text())
				continue
			}
			switch msg.Role {
			case message.RoleUser:
				conversationMessages = append(conversationMessages,
					anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Text())))
			case message.RoleAssistant:
				conversationMessages = append(conversationMessages,
					anthropic.NewAssistantMessage(anthropic.NewTextBlock(msg.Text())))
			}
		}

		params := anthropic.MessageNewParams{
			Model:     anthropic.Model(p.config.Model),
			Messages:  conversationMessages,
			MaxTokens: p.config.MaxTokens,
		}

		if len(systemPrompts) > 0 {
			systemText := ""
			for i, sp := range systemPrompts {
				if i > 0 {
					systemText += "\n"
				}
				systemText += sp
			}
			params.System = []anthropic.TextBlockParam{{Text: systemText}}
		}

		if p.config.Temperature > 0 {
			params.Temperature = param.NewOpt(p.config.Temperature)
		}

		if len(req.Tools) > 0 {
			claudeTools := make([]anthropic.ToolUnionParam, 0, len(req.Tools))
			for _, tool := range req.Tools {
				toolJSON, err := json.Marshal(tool)
				if err != nil {
					yield(nil, fmt.Errorf("failed to marshal tool: %w", err))
					return
				}

				var toolParam anthropic.ToolParam
				if err := json.Unmarshal(toolJSON, &toolParam); err != nil {
					yield(nil, fmt.Errorf("failed to unmarshal tool param: %w", err))
					return
				}

				claudeTools = append(claudeTools, anthropic.ToolUnionParam{OfTool: &toolParam})
			}
			params.Tools = claudeTools
		}

		stream := p.client.Messages.NewStreaming(ctx, params)
		defer stream.Close()

		finalMsg := message.NewMessage(message.RoleAssistant, "")
		var finalMessage *anthropic.Message

		for stream.Next() {
			event := stream.Current()
			switch event.Type {
			case "content_block_delta":
				contentDelta := event.AsContentBlockDelta()
				if contentDelta.Delta.Type == "text_delta" && contentDelta.Delta.Text != "" {
					finalMsg.AppendText(contentDelta.Delta.Text)
					chunk := message.NewMessage(message.RoleAssistant, contentDelta.Delta.Text)
					chunk.Completed = false
					if !yield(chunk, nil) {
						return
					}
				}
			case "message_start":
				msgStart := event.AsMessageStart()
				finalMessage = &msgStart.Message
			case "message_stop":
				// no-op
			}
		}

		if err := stream.Err(); err != nil {
			yield(nil, fmt.Errorf("Claude streaming error: %w", err))
			return
		}

		if finalMessage != nil {
			var toolCalls []message.ToolCall
			for _, content := range finalMessage.Content {
				if content.Type != "tool_use" {
					continue
				}
				var args map[string]any
				if err := json.Unmarshal(content.Input, &args); err == nil {
					toolCalls = append(toolCalls, message.ToolCall{
						ID:   content.ID,
						Name: content.Name,
						Args: args,
					})
				}
			}
			if len(toolCalls) > 0 {
				finalMsg.ToolCalls = toolCalls
			}
		}

		finalMsg.Completed = true
		yield(finalMsg, nil)
	}
}
