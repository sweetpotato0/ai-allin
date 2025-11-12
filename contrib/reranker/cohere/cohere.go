package cohere

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sweetpotato0/ai-allin/rag/reranker"
)

const defaultEndpoint = "https://api.cohere.com/v1/rerank"

// Client implements Cohere's ReRank API.
type Client struct {
	apiKey     string
	model      string
	topN       int
	httpClient *http.Client
	endpoint   string
	fallback   reranker.Reranker
}

// Option customises the Cohere reranker client.
type Option func(*Client)

// WithModel overrides the default Cohere model (rerank-english-v3.0).
func WithModel(model string) Option {
	return func(c *Client) {
		if model != "" {
			c.model = model
		}
	}
}

// WithTopN limits how many documents Cohere re-ranks per call.
func WithTopN(topN int) Option {
	return func(c *Client) {
		if topN > 0 {
			c.topN = topN
		}
	}
}

// WithHTTPClient swaps the HTTP client (useful for timeouts or proxies).
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		if client != nil {
			c.httpClient = client
		}
	}
}

// WithEndpoint overrides the Cohere API endpoint.
func WithEndpoint(endpoint string) Option {
	return func(c *Client) {
		if endpoint != "" {
			c.endpoint = endpoint
		}
	}
}

// WithFallback specifies the reranker used when Cohere is unavailable.
func WithFallback(r reranker.Reranker) Option {
	return func(c *Client) {
		if r != nil {
			c.fallback = r
		}
	}
}

// New creates a new Cohere-based reranker.
func New(apiKey string, opts ...Option) *Client {
	client := &Client{
		apiKey:     apiKey,
		model:      "rerank-english-v3.0",
		topN:       50,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		endpoint:   defaultEndpoint,
	}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

type rerankRequest struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopN      int      `json:"top_n,omitempty"`
}

type rerankResponse struct {
	Results []struct {
		Index int     `json:"index"`
		Score float32 `json:"score"`
	} `json:"results"`
}

// Rank implements reranker.Reranker.
func (c *Client) Rank(ctx context.Context, queryVector []float32, candidates []reranker.Candidate) ([]reranker.Result, error) {
	if len(candidates) == 0 {
		return nil, nil
	}
	query, ok := reranker.QueryFromContext(ctx)
	if !ok || strings.TrimSpace(query) == "" || c.apiKey == "" {
		return c.runFallback(ctx, queryVector, candidates, nil)
	}

	limit := len(candidates)
	if limit > c.topN {
		limit = c.topN
	}
	docTexts := make([]string, limit)
	for i := 0; i < limit; i++ {
		docTexts[i] = candidates[i].Chunk.Content
	}

	payload := rerankRequest{
		Model:     c.model,
		Query:     query,
		Documents: docTexts,
		TopN:      limit,
	}
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return c.runFallback(ctx, queryVector, candidates, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return c.runFallback(ctx, queryVector, candidates, fmt.Errorf("cohere rerank failed: status %d", resp.StatusCode))
	}

	var rr rerankResponse
	if err := json.NewDecoder(resp.Body).Decode(&rr); err != nil {
		return c.runFallback(ctx, queryVector, candidates, err)
	}

	results := make([]reranker.Result, 0, len(rr.Results))
	for _, res := range rr.Results {
		if res.Index < 0 || res.Index >= limit {
			continue
		}
		results = append(results, reranker.Result{
			Chunk: candidates[res.Index].Chunk,
			Score: res.Score,
		})
	}
	if len(results) == 0 {
		return c.runFallback(ctx, queryVector, candidates, fmt.Errorf("cohere returned no results"))
	}
	return results, nil
}

func (c *Client) runFallback(ctx context.Context, queryVector []float32, candidates []reranker.Candidate, cause error) ([]reranker.Result, error) {
	if c.fallback == nil {
		return defaultVectorSort(queryVector, candidates), cause
	}
	results, err := c.fallback.Rank(ctx, queryVector, candidates)
	if err != nil {
		return results, err
	}
	return results, cause
}

func defaultVectorSort(queryVector []float32, candidates []reranker.Candidate) []reranker.Result {
	results := make([]reranker.Result, 0, len(candidates))
	for _, cand := range candidates {
		results = append(results, reranker.Result{
			Chunk: cand.Chunk,
			Score: cand.Score,
		})
	}
	return results
}
