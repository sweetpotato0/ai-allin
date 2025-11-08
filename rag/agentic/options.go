package agentic

import (
	"github.com/sweetpotato0/ai-allin/rag/chunking"
	"github.com/sweetpotato0/ai-allin/rag/reranker"
)

// Config controls behaviour of the Agentic pipeline as well as the retrieval stage.
// It intentionally groups prompt/middleware knobs and low-level retrieval parameters
// so callers can construct reproducible agents from a single struct.
type Config struct {
	Name           string // Logical name for tracing/logging
	TopK           int    // How many neighbors to pull from the vector store
	RerankTopK     int    // How many results survive reranking
	MaxPlanSteps   int    // Upper bound for planner emitted steps
	EnableCritic   bool   // Toggle critic agent execution
	GraphMaxVisits int    // Safety guard for graph execution

	PlannerPrompt   string // Custom system prompt for planner agent
	QueryPrompt     string // System prompt for researcher/query agent
	SynthesisPrompt string // System prompt for writer/synthesizer agent
	CriticPrompt    string // System prompt for critic agent

	ChunkSize    int // Desired chunk size used by default chunker
	ChunkOverlap int // Overlap between consecutive chunks

	chunker   chunking.Chunker  // Optional override for chunking strategy
	reranker  reranker.Reranker // Optional override for reranking stage
	retrieval RetrievalEngine   // Optional override for the entire retrieval engine
}

// Option customises the pipeline configuration.
type Option func(*Config)

// WithTopK overrides how many documents each plan step retrieves from the vector store
// before reranking is applied.
func WithTopK(k int) Option {
	return func(cfg *Config) {
		if k > 0 {
			cfg.TopK = k
		}
	}
}

// WithRerankTopK caps how many results survive reranking and flow into generation.
func WithRerankTopK(k int) Option {
	return func(cfg *Config) {
		if k > 0 {
			cfg.RerankTopK = k
		}
	}
}

// WithCritic enables or disables the critic agent.
func WithCritic(enabled bool) Option {
	return func(cfg *Config) {
		cfg.EnableCritic = enabled
	}
}

// WithPlannerPrompt sets the system prompt used by the planner agent.
func WithPlannerPrompt(prompt string) Option {
	return func(cfg *Config) {
		if prompt != "" {
			cfg.PlannerPrompt = prompt
		}
	}
}

// WithQueryPrompt sets the researcher/query-rewriter system prompt.
func WithQueryPrompt(prompt string) Option {
	return func(cfg *Config) {
		if prompt != "" {
			cfg.QueryPrompt = prompt
		}
	}
}

// WithSynthesisPrompt sets the writer/synthesiser system prompt.
func WithSynthesisPrompt(prompt string) Option {
	return func(cfg *Config) {
		if prompt != "" {
			cfg.SynthesisPrompt = prompt
		}
	}
}

// WithCriticPrompt sets the critic system prompt.
func WithCriticPrompt(prompt string) Option {
	return func(cfg *Config) {
		if prompt != "" {
			cfg.CriticPrompt = prompt
		}
	}
}

// WithMaxPlanSteps caps the number of steps that the planner may emit.
func WithMaxPlanSteps(max int) Option {
	return func(cfg *Config) {
		if max > 0 {
			cfg.MaxPlanSteps = max
		}
	}
}

// WithChunkSize configures chunk character length for indexing.
func WithChunkSize(size int) Option {
	return func(cfg *Config) {
		if size > 0 {
			cfg.ChunkSize = size
		}
	}
}

// WithChunkOverlap configures overlap between chunks.
func WithChunkOverlap(overlap int) Option {
	return func(cfg *Config) {
		if overlap >= 0 {
			cfg.ChunkOverlap = overlap
		}
	}
}

// WithChunker plugs in a custom chunker implementation.
func WithChunker(ch chunking.Chunker) Option {
	return func(cfg *Config) {
		if ch != nil {
			cfg.chunker = ch
		}
	}
}

// WithReranker plugs in a custom reranker implementation.
func WithReranker(r reranker.Reranker) Option {
	return func(cfg *Config) {
		if r != nil {
			cfg.reranker = r
		}
	}
}

// WithRetriever sets a fully managed retrieval engine, bypassing the built-in chunk/embed construction.
func WithRetriever(engine RetrievalEngine) Option {
	return func(cfg *Config) {
		if engine != nil {
			cfg.retrieval = engine
		}
	}
}

// WithGraphMaxVisits tweaks the safety guard for graph traversal.
func WithGraphMaxVisits(max int) Option {
	return func(cfg *Config) {
		if max > 0 {
			cfg.GraphMaxVisits = max
		}
	}
}

func defaultConfig() *Config {
	return &Config{
		Name:            "agentic-rag",
		TopK:            3,
		RerankTopK:      4,
		MaxPlanSteps:    4,
		EnableCritic:    true,
		GraphMaxVisits:  20,
		ChunkSize:       800,
		ChunkOverlap:    120,
		PlannerPrompt:   "You are a senior research planner. Break down complex user questions into at most {{max_steps}} ordered steps. Output strict JSON {\"strategy\": string, \"steps\": [{\"id\": \"step-1\", \"goal\": \"...\", \"questions\": [\"...\"], \"expected_evidence\": \"...\"}]}. Each step must be actionable and cite the signals it needs.",
		QueryPrompt:     "You are a search strategist. For the provided plan step craft up to 2 short search queries or keywords. Return JSON {\"queries\": [\"...\"]}. Keep queries specific to the step goal.",
		SynthesisPrompt: "You are a staff research writer. Using only the supplied evidence, answer the question. Cite documents using [doc-id] format. Output helpful, structured text.",
		CriticPrompt:    "You are a meticulous reviewer. Check whether the draft answer follows the plan and uses evidence. Return JSON {\"verdict\": \"approve|revise\", \"issues\": [], \"notes\": \"\", \"final_answer\": \"...\"}. If verdict=approve keep final_answer equal to the draft.",
	}
}

func applyOptions(cfg *Config, opts []Option) *Config {
	if cfg == nil {
		cfg = defaultConfig()
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(cfg)
	}
	return cfg
}
