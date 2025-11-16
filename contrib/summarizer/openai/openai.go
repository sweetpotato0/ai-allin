package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/contrib/provider/openai"
	"github.com/sweetpotato0/ai-allin/rag/document"
)

type OpenAISummarizer struct {
	agent  *agent.Agent
	tokens int
}

func NewOpenAISummarizer(apiKey string, tokens int) *OpenAISummarizer {
	return &OpenAISummarizer{
		tokens: tokens,
	}
}

// SummarizeChunks: batch summary (preserve chunk order)
func (s *OpenAISummarizer) SummarizeChunks(ctx context.Context, chunks []document.Chunk) ([]document.Summary, error) {
	// naive: call per-chunk but with concurrency + rate limit
	out := make([]document.Summary, len(chunks))
	sem := make(chan struct{}, 8) // concurrency
	errc := make(chan error, 1)

	for i := range chunks {
		sem <- struct{}{}
		go func(i int) {
			defer func() { <-sem }()
			sum, err := s.summarizeOne(ctx, chunks[i])
			if err != nil {
				select {
				case errc <- err:
				default:
				}
				return
			}
			out[i] = *sum
		}(i)
	}

	// wait all done
	for i := 0; i < cap(sem); i++ {
		sem <- struct{}{}
	}
	select {
	case e := <-errc:
		return nil, e
	default:
	}
	return out, nil
}

func (s *OpenAISummarizer) summarizeOne(ctx context.Context, c document.Chunk) (*document.Summary, error) {
	prompt := fmt.Sprintf(`Please provide a reasoning summary of the following text:
Title: %s
Section: %s
Content:
%s

Requirements:
1) Output in input language
2) Generate a concise summary and the length is approximately %d tokens
3) Extract 3-5 key points (listed by number)
4) Output in JSON format: {"summary":"...","key_points":["kp1","kp2"...]}
`, c.Metadata["title"], c.Section, c.Content, s.tokens)

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY is required to run the Agentic RAG example")
	}

	baseURL := os.Getenv("OPENAI_API_BASE_URL")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY is required to run the Agentic RAG example")
	}
	llm := openai.New(openai.DefaultConfig().WithAPIKey(apiKey).WithBaseURL(baseURL).WithModel("gpt-4o"))

	ag := agent.New(
		agent.WithName("mcp-agent"),
		agent.WithSystemPrompt("You are a helpful assistant that can call MCP tools when needed."),
		agent.WithProvider(llm),
	)

	response, err := ag.Run(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("agent run failed: %v", err)
	}

	text := response.Text()
	// parse JSON from the text (best-effort)
	var js struct {
		Summary   string   `json:"summary"`
		KeyPoints []string `json:"key_points"`
	}
	if err := json.Unmarshal([]byte(text), &js); err != nil {
		return nil, fmt.Errorf("agent return response unmarshal failed: %v", err)
	}
	return &document.Summary{
		ChunkID:   c.ID,
		Summary:   js.Summary,
		KeyPoints: js.KeyPoints,
	}, nil
}
