package agentic

import (
	"strings"

	"github.com/sweetpotato0/ai-allin/rag/chunking"
	"github.com/sweetpotato0/ai-allin/rag/reranker"
)

// Config controls behaviour of the Agentic pipeline as well as the retrieval stage.
// It intentionally groups prompt/middleware knobs and low-level retrieval parameters
// so callers can construct reproducible agents from a single struct.
type Config struct {
	Name                string // Logical name for tracing/logging
	TopK                int    // How many neighbors to pull from the vector store
	RerankTopK          int    // How many results survive reranking
	MaxPlanSteps        int    // Upper bound for planner emitted steps
	EnableCritic        bool   // Toggle critic agent execution
	GraphMaxVisits      int    // Safety guard for graph execution
	MinEvidenceCount    int    // Minimum evidence items required before synthesis runs
	MinSearchScore      float32
	EnableHybridSearch  bool
	HybridTopK          int
	TitleScorePenalty   float32
	NormalizeEmbeddings bool

	PlannerPrompt   string // Custom system prompt for planner agent
	QueryPrompt     string // System prompt for researcher/query agent
	SynthesisPrompt string // System prompt for writer/synthesizer agent
	CriticPrompt    string // System prompt for critic agent
	NoAnswerMessage string // Message emitted when evidence is insufficient

	QueryLLMRetries int // How many times the researcher retries invalid LLM output
	QueryMaxResults int // Upper bound on emitted queries per plan step

	ChunkSize      int    // Desired chunk size used by default chunker
	ChunkOverlap   int    // Overlap between consecutive chunks
	ChunkMinSize   int    // Merge short chunks until reaching this size
	ChunkSeparator string // Custom separator passed to chunker

	chunker   chunking.Chunker  // Optional override for chunking strategy
	reranker  reranker.Reranker // Optional override for reranking stage
	retrieval RetrievalEngine   // Optional override for the entire retrieval engine
}

// RetrievalPreset bundles commonly used retrieval settings.
type RetrievalPreset string

const (
	// RetrievalPresetSimple favours speed and disables hybrid search.
	RetrievalPresetSimple RetrievalPreset = "simple"
	// RetrievalPresetBalanced provides a middle ground between recall and latency.
	RetrievalPresetBalanced RetrievalPreset = "balanced"
	// RetrievalPresetHybrid maximises recall by enabling hybrid search and vector normalisation.
	RetrievalPresetHybrid RetrievalPreset = "hybrid"
)

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

// WithRetrievalPreset applies a predefined bundle of retrieval settings instead of toggling knobs one by one.
func WithRetrievalPreset(preset RetrievalPreset) Option {
	return func(cfg *Config) {
		switch preset {
		case RetrievalPresetSimple:
			cfg.TopK = 3
			cfg.RerankTopK = 3
			cfg.MinSearchScore = 0
			cfg.EnableHybridSearch = false
			cfg.HybridTopK = 0
			cfg.TitleScorePenalty = 0.9
			cfg.NormalizeEmbeddings = false
		case RetrievalPresetBalanced:
			cfg.TopK = 6
			cfg.RerankTopK = 4
			cfg.MinSearchScore = 0.15
			cfg.EnableHybridSearch = false
			cfg.HybridTopK = 0
			cfg.TitleScorePenalty = 0.9
			cfg.NormalizeEmbeddings = true
		case RetrievalPresetHybrid:
			cfg.TopK = 8
			cfg.RerankTopK = 6
			cfg.MinSearchScore = 0.25
			cfg.EnableHybridSearch = true
			cfg.HybridTopK = 6
			cfg.TitleScorePenalty = 0.85
			cfg.NormalizeEmbeddings = true
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

// WithMinEvidenceCount sets the minimum amount of evidence required before synthesis runs.
func WithMinEvidenceCount(count int) Option {
	return func(cfg *Config) {
		if count >= 0 {
			cfg.MinEvidenceCount = count
		}
	}
}

// WithMinSearchScore filters retrieval results below the provided score.
func WithMinSearchScore(score float32) Option {
	return func(cfg *Config) {
		if score >= 0 {
			cfg.MinSearchScore = score
		}
	}
}

// WithHybridSearch toggles the keyword fallback search.
func WithHybridSearch(enabled bool) Option {
	return func(cfg *Config) {
		cfg.EnableHybridSearch = enabled
	}
}

// WithHybridTopK caps how many fallback keyword hits to merge.
func WithHybridTopK(k int) Option {
	return func(cfg *Config) {
		if k > 0 {
			cfg.HybridTopK = k
		}
	}
}

// WithTitleScorePenalty reduces the score of title chunks to favor body text.
func WithTitleScorePenalty(p float32) Option {
	return func(cfg *Config) {
		if p > 0 && p <= 1 {
			cfg.TitleScorePenalty = p
		}
	}
}

// WithNormalizeEmbeddings enforces L2-normalisation before vector storage.
func WithNormalizeEmbeddings(enabled bool) Option {
	return func(cfg *Config) {
		cfg.NormalizeEmbeddings = enabled
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

// WithQueryRetries overrides how many times the researcher retries when the LLM output cannot be parsed.
func WithQueryRetries(retries int) Option {
	return func(cfg *Config) {
		if retries >= 0 {
			cfg.QueryLLMRetries = retries
		}
	}
}

// WithQueryMaxResults limits how many queries the researcher may emit per plan step.
func WithQueryMaxResults(max int) Option {
	return func(cfg *Config) {
		if max > 0 {
			cfg.QueryMaxResults = max
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

// WithNoAnswerMessage customises the fallback message when there is no supporting evidence.
func WithNoAnswerMessage(message string) Option {
	return func(cfg *Config) {
		if strings.TrimSpace(message) != "" {
			cfg.NoAnswerMessage = message
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

// WithChunkSeparator overrides the logical separator used by the default chunker.
func WithChunkSeparator(sep string) Option {
	return func(cfg *Config) {
		if strings.TrimSpace(sep) != "" {
			cfg.ChunkSeparator = sep
		}
	}
}

// WithChunkMinSize enforces a lower bound on chunk length before emitting.
func WithChunkMinSize(size int) Option {
	return func(cfg *Config) {
		if size > 0 {
			cfg.ChunkMinSize = size
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
	cfg := &Config{
		Name:             "agentic-rag",
		MaxPlanSteps:     4,
		EnableCritic:     true,
		GraphMaxVisits:   20,
		MinEvidenceCount: 1,
		QueryLLMRetries:  2,
		QueryMaxResults:  4,
		ChunkSize:        800,
		ChunkOverlap:     120,
		ChunkMinSize:     200,
		ChunkSeparator:   "\n\n",
		PlannerPrompt: `You are the lead planner for an agentic RAG pipeline. Break the user question into at most {{max_steps}} sequential research steps that collect the evidence needed for a final answer. Output compact JSON only matching {"strategy":"...", "steps":[{"id":"step-1","goal":"...","questions":["..."],"expected_evidence":"...","downstream_support":"..."}]}.
Planning rules:
- "strategy" is a single sentence describing the overall approach.
- Each step is actionable, scoped to one objective, and states which signals or document types to hunt for inside "expected_evidence".
- Use "questions" for concrete clarifications or keyword variants that inform search.
- Note how the step unlocks later work in "downstream_support" (leave empty if unnecessary) and keep IDs sequential (step-1, step-2, ...).
- Merge redundant tasks, never exceed {{max_steps}} steps, and mirror the user's language (Chinese input -> Chinese plan, otherwise English).`,
		QueryPrompt: `You are a multilingual search strategist assisting the researcher. Transform the provided plan step into retrieval-ready search queries.
Return strict JSON {"queries":["..."],"question":"original question"} with no prose.
Rules:
- Produce at least one and at most the requested maximum number of queries; diversify vocabulary, operators, and intent.
- Keep each query under 18 words, inject concrete entities, time ranges, file types, or domain hints drawn from the step's goal, questions, and expected evidence.
- Remove duplicates or vague boilerplate; if a clarifying probe would unblock the step, include one precise question-style query.
- Always write the queries in the same language as the user's question (Chinese stays Chinese, otherwise English).`,
		SynthesisPrompt: `You are the staff research writer for this RAG system. Using only the supplied evidence, craft a precise, citation-backed answer to the user question.
Guidelines:
1. Synthesize across documents, pointing out agreements or contradictions before concluding.
2. Attribute every factual statement with [doc-id] citations placed at the end of the supporting sentence or clause.
3. Organise the response into short sections or bullet lists when multiple themes exist, and close with a brief "Limitations / Next steps" note when coverage is partial.
4. If the evidence cannot answer the question, say so explicitly and describe what information is missing instead of guessing.
5. Respond entirely in the user's language (Chinese input -> Chinese output; otherwise English).`,
		CriticPrompt: `You are the QA critic for the agentic RAG pipeline. Verify that the draft answer follows the plan, uses the supplied evidence, and satisfies the user instructions.
Return JSON only: {"verdict":"approve|revise","issues":["..."],"notes":"...","final_answer":"..."}.
Rules:
- Approve only when the draft answers the question, covers required plan steps, and cites existing evidence without hallucinations.
- List concrete problems in "issues" (missing evidence, wrong citations, unanswered sub-questions) referencing plan step IDs or [doc-id] when helpful.
- If revision is needed, set "verdict":"revise" and provide an improved, citation-backed answer in "final_answer"; otherwise copy the draft verbatim.
- Match the language of the original question (Chinese stays Chinese, else English).`,
		NoAnswerMessage: "抱歉，我没有在知识库中找到与该问题相关的答案，请提供更多上下文或重新描述问题。",
	}
	WithRetrievalPreset(RetrievalPresetHybrid)(cfg)
	return cfg
}

func applyOptions(cfg *Config, opts []Option) *Config {
	if cfg == nil {
		cfg = defaultConfig()
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}
