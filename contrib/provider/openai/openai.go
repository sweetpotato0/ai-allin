package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
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

// WithModel set model.
func (cfg *Config) WithModel(model string) *Config {
	cfg.Model = model
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

var _ agent.LLMClient = (*Provider)(nil)

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
func (p *Provider) Generate(ctx context.Context, req *agent.GenerateRequest) (*agent.GenerateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("generate request cannot be nil")
	}
	// Convert messages to OpenAI format
	openAIMessages := make([]openai.ChatCompletionMessageParamUnion, 0, len(req.Messages))
	for _, msg := range req.Messages {
		switch msg.Role {
		case message.RoleSystem:
			openAIMessages = append(openAIMessages, openai.SystemMessage(msg.Text()))
		case message.RoleUser:
			openAIMessages = append(openAIMessages, openai.UserMessage(msg.Text()))
		case message.RoleAssistant:
			assistantMsg := openai.AssistantMessage(msg.Text())
			if len(msg.ToolCalls) > 0 {
				toolCalls, err := encodeToolCalls(msg.ToolCalls)
				if err != nil {
					return nil, fmt.Errorf("failed to encode tool calls: %w", err)
				}
				if assistantMsg.OfAssistant != nil {
					assistantMsg.OfAssistant.ToolCalls = toolCalls
				}
			}
			openAIMessages = append(openAIMessages, assistantMsg)
		case message.RoleTool:
			openAIMessages = append(openAIMessages, openai.ToolMessage(msg.Text(), msg.ToolID))
		}
	}

	// Build chat completion request
	model := p.config.Model
	if model == "" {
		model = string(openai.ChatModelGPT4oMini)
	}
	params := openai.ChatCompletionNewParams{
		Messages: openAIMessages,
		Model:    openai.ChatModel(model),
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
	if len(req.Tools) > 0 {
		openAITools := make([]openai.ChatCompletionToolUnionParam, 0, len(req.Tools))
		for _, tool := range req.Tools {
			// Convert tool schema
			toolJSON, err := json.Marshal(tool)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal tool: %w", err)
			}

			var toolParam openai.ChatCompletionToolUnionParam
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
func (p *Provider) GenerateStream(ctx context.Context, req *agent.GenerateRequest) iter.Seq2[*agent.GenerateResponse, error] {
	return func(yield func(*agent.GenerateResponse, error) bool) {
		if req == nil {
			yield(nil, fmt.Errorf("stream request cannot be nil"))
			return
		}

		openAIMessages := make([]openai.ChatCompletionMessageParamUnion, 0, len(req.Messages))
		for _, msg := range req.Messages {
			switch msg.Role {
			case message.RoleSystem:
				openAIMessages = append(openAIMessages, openai.SystemMessage(msg.Text()))
			case message.RoleUser:
				openAIMessages = append(openAIMessages, openai.UserMessage(msg.Text()))
			case message.RoleAssistant:
				assistantMsg := openai.AssistantMessage(msg.Text())
				if len(msg.ToolCalls) > 0 {
					toolCalls, err := encodeToolCalls(msg.ToolCalls)
					if err != nil {
						yield(nil, fmt.Errorf("failed to encode tool calls: %w", err))
						return
					}
					if assistantMsg.OfAssistant != nil {
						assistantMsg.OfAssistant.ToolCalls = toolCalls
					}
				}
				openAIMessages = append(openAIMessages, assistantMsg)
			case message.RoleTool:
				openAIMessages = append(openAIMessages, openai.ToolMessage(msg.Text(), msg.ToolID))
			}
		}

		model := p.config.Model
		if model == "" {
			model = string(openai.ChatModelGPT4oMini)
		}
		params := openai.ChatCompletionNewParams{
			Messages: openAIMessages,
			Model:    openai.ChatModel(model),
		}

		if p.config.Temperature > 0 {
			params.Temperature = param.NewOpt(p.config.Temperature)
		}

		if p.config.MaxTokens > 0 {
			params.MaxCompletionTokens = param.NewOpt(p.config.MaxTokens)
		}

		if len(req.Tools) > 0 {
			openAITools := make([]openai.ChatCompletionToolUnionParam, 0, len(req.Tools))
			for _, tool := range req.Tools {
				toolJSON, err := json.Marshal(tool)
				if err != nil {
					yield(nil, fmt.Errorf("failed to marshal tool: %w", err))
					return
				}

				var toolParam openai.ChatCompletionToolUnionParam
				if err := json.Unmarshal(toolJSON, &toolParam); err != nil {
					yield(nil, fmt.Errorf("failed to unmarshal tool param: %w", err))
					return
				}

				openAITools = append(openAITools, toolParam)
			}
			params.Tools = openAITools
		}

		stream := p.client.Chat.Completions.NewStreaming(ctx, params)
		defer stream.Close()

		acc := openai.ChatCompletionAccumulator{}
		for stream.Next() {
			event := stream.Current()
			if len(event.Choices) == 0 {
				continue
			}

			acc.AddChunk(event)

			choice := event.Choices[0]
			response := &agent.GenerateResponse{
				Message: message.NewEmptyMessage(message.RoleAssistant),
			}
			if choice.Delta.Content != "" {
				response.Message.SetText(choice.Delta.Content)
			}

			if choice.FinishReason != "" {
				response.Message.FinishReason = choice.FinishReason
			}

			if !yield(response, nil) {
				return
			}
		}

		if err := stream.Err(); err != nil {
			yield(nil, fmt.Errorf("OpenAI streaming error: %w", err))
			return
		}

		finalMsg := &agent.GenerateResponse{
			Message: message.NewEmptyMessage(message.RoleAssistant),
		}
		tcs := acc.Choices[0].Message.ToolCalls
		finalMsg.Message.ToolCalls = make([]message.ToolCall, len(tcs))
		finalMsg.Message.Completed = true

		for i, call := range tcs {
			var args map[string]any
			if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
				yield(nil, fmt.Errorf("failed to parse tool arguments: %w", err))
				return
			}
			finalMsg.Message.ToolCalls[i] = message.ToolCall{
				ID:   call.ID,
				Name: call.Function.Name,
				Args: args,
			}
		}
		yield(finalMsg, nil)
	}
}

func encodeToolCalls(calls []message.ToolCall) ([]openai.ChatCompletionMessageToolCallUnionParam, error) {
	if len(calls) == 0 {
		return nil, nil
	}
	params := make([]openai.ChatCompletionMessageToolCallUnionParam, 0, len(calls))
	for _, tc := range calls {
		args := tc.Args
		if args == nil {
			args = make(map[string]any)
		}
		raw, err := json.Marshal(args)
		if err != nil {
			return nil, err
		}
		params = append(params, openai.ChatCompletionMessageToolCallUnionParam{
			OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
				ID: tc.ID,
				Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
					Name:      tc.Name,
					Arguments: string(raw),
				},
			},
		})
	}
	return params, nil
}
