package gemini

import (
	"context"
	"fmt"
	"sync"

	"iter"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/message"
)

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

var (
	_ agent.StreamLLMClient = (*Provider)(nil)
)

// Provider implements the LLMClient interface for Google Gemini
type Provider struct {
	config *Config

	mu     sync.Mutex
	client *genai.Client
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
	}
}

// Generate implements agent.LLMClient interface
func (p *Provider) Generate(ctx context.Context, req *agent.GenerateRequest) (*agent.GenerateResponse, error) {
	if p.config.APIKey == "" {
		return nil, fmt.Errorf("Gemini API key not configured")
	}
	if req == nil {
		return nil, fmt.Errorf("generate request cannot be nil")
	}

	model, err := p.ensureModel(ctx)
	if err != nil {
		return nil, err
	}

	contents := toGeminiContents(req.Messages)
	if len(contents) == 0 {
		return nil, fmt.Errorf("no messages provided")
	}

	session := model.StartChat()
	if len(contents) > 1 {
		session.History = append(session.History, contents[:len(contents)-1]...)
	}
	last := contents[len(contents)-1]
	if len(last.Parts) == 0 {
		return nil, fmt.Errorf("last message has no content")
	}

	resp, err := session.SendMessage(ctx, last.Parts...)
	if err != nil {
		return nil, fmt.Errorf("Gemini generate call failed: %w", err)
	}
	return convertResponse(resp)
}

// GenerateStream implements agent.StreamLLMClient for Gemini.
func (p *Provider) GenerateStream(ctx context.Context, req *agent.GenerateRequest) iter.Seq2[*agent.GenerateResponse, error] {
	return func(yield func(*agent.GenerateResponse, error) bool) {
		if p.config.APIKey == "" {
			yield(nil, fmt.Errorf("Gemini API key not configured"))
			return
		}
		if req == nil {
			yield(nil, fmt.Errorf("stream request cannot be nil"))
			return
		}

		model, err := p.ensureModel(ctx)
		if err != nil {
			yield(nil, err)
			return
		}

		contents := toGeminiContents(req.Messages)
		if len(contents) == 0 {
			yield(nil, fmt.Errorf("no messages provided"))
			return
		}

		session := model.StartChat()
		if len(contents) > 1 {
			session.History = append(session.History, contents[:len(contents)-1]...)
		}
		last := contents[len(contents)-1]
		if len(last.Parts) == 0 {
			yield(nil, fmt.Errorf("last message has no content"))
			return
		}

		stream := session.SendMessageStream(ctx, last.Parts...)
		for {
			resp, err := stream.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				yield(nil, fmt.Errorf("Gemini streaming error: %w", err))
				return
			}
			chunk, ok := chunkResponse(resp)
			if !ok {
				continue
			}
			if !yield(chunk, nil) {
				return
			}
		}

		final := stream.MergedResponse()
		if final == nil {
			yield(nil, fmt.Errorf("Gemini streaming ended without final response"))
			return
		}
		genResp, err := convertResponse(final)
		if err != nil {
			yield(nil, err)
			return
		}
		yield(genResp, nil)
	}
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

func (p *Provider) ensureModel(ctx context.Context) (*genai.GenerativeModel, error) {
	client, err := p.ensureClient(ctx)
	if err != nil {
		return nil, err
	}

	model := client.GenerativeModel(p.config.Model)
	if p.config.MaxTokens > 0 {
		mt := int32(p.config.MaxTokens)
		model.GenerationConfig.MaxOutputTokens = &mt
	}
	if p.config.Temperature > 0 {
		temp := p.config.Temperature
		model.GenerationConfig.Temperature = &temp
	}
	return model, nil
}

func (p *Provider) ensureClient(ctx context.Context) (*genai.Client, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.client != nil {
		return p.client, nil
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(p.config.APIKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	p.client = client
	return p.client, nil
}

func toGeminiContents(msgs []*message.Message) []*genai.Content {
	contents := make([]*genai.Content, 0, len(msgs))
	for _, msg := range msgs {
		if msg == nil || len(msg.Content.Parts) == 0 {
			continue
		}

		parts := make([]genai.Part, 0, len(msg.Content.Parts))
		for _, part := range msg.Content.Parts {
			if part.Text == "" {
				continue
			}
			parts = append(parts, genai.Text(part.Text))
		}
		if len(parts) == 0 {
			continue
		}

		contents = append(contents, &genai.Content{
			Role:  mapRole(msg.Role),
			Parts: parts,
		})
	}
	return contents
}

func mapRole(role message.Role) string {
	switch role {
	case message.RoleAssistant:
		return "model"
	default:
		return "user"
	}
}

func convertResponse(resp *genai.GenerateContentResponse) (*agent.GenerateResponse, error) {
	cand := firstCandidate(resp)
	if cand == nil {
		return nil, fmt.Errorf("empty response from Gemini")
	}

	msg := message.NewEmptyMessage(message.RoleAssistant)
	msg.FinishReason = cand.FinishReason.String()
	appendCandidateParts(msg, cand)
	if len(msg.Content.Parts) == 0 && len(msg.ToolCalls) == 0 {
		return nil, fmt.Errorf("Gemini candidate missing content")
	}
	if cand.FinishReason != genai.FinishReasonUnspecified {
		msg.Completed = true
	}

	return &agent.GenerateResponse{Message: msg}, nil
}

func chunkResponse(resp *genai.GenerateContentResponse) (*agent.GenerateResponse, bool) {
	cand := firstCandidate(resp)
	if cand == nil || (cand.Content == nil && cand.FinishReason == genai.FinishReasonUnspecified) {
		return nil, false
	}
	msg := message.NewEmptyMessage(message.RoleAssistant)
	msg.FinishReason = cand.FinishReason.String()
	appendCandidateParts(msg, cand)
	if len(msg.Content.Parts) == 0 && len(msg.ToolCalls) == 0 {
		return nil, false
	}
	return &agent.GenerateResponse{Message: msg}, true
}

func appendCandidateParts(msg *message.Message, cand *genai.Candidate) {
	if cand == nil || cand.Content == nil {
		return
	}
	for _, part := range cand.Content.Parts {
		switch p := part.(type) {
		case genai.Text:
			msg.Content.Parts = append(msg.Content.Parts, message.Part{Text: string(p)})
		case genai.FunctionCall:
			index := len(msg.ToolCalls) + 1
			msg.ToolCalls = append(msg.ToolCalls, message.ToolCall{
				ID:   fmt.Sprintf("gemini-call-%d", index),
				Name: p.Name,
				Args: p.Args,
			})
		}
	}
}

func firstCandidate(resp *genai.GenerateContentResponse) *genai.Candidate {
	if resp == nil || len(resp.Candidates) == 0 {
		return nil
	}
	return resp.Candidates[0]
}
