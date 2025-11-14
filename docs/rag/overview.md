# Agentic RAG Overview

The `rag/agentic` package adds an opinionated, multi-agent Retrieval-Augmented Generation pipeline on top of the base agent framework. It is designed for teams that want a production friendly way to plan, retrieve, reason, and critique answers with explicit structure and observability.

## Why Agentic RAG?

- **Deterministic planning** – a planner agent decomposes user questions into auditable steps.
- **Retrieval transparency** – every plan step tracks the queries plus the documents that satisfied it.
- **High-quality writing** – a dedicated synthesizer agent turns evidence into structured answers.
- **Safety net** – an optional critic agent validates draft answers and can revise them.
- **Graph-native orchestration** – the pipeline is implemented as a `graph.Graph`, making it easy to extend with custom nodes or replace agents.

## Architecture

```
┌────────┐   ┌────────────┐   ┌──────────────┐   ┌─────────┐   ┌────────────┐
│ Start  │-->| Planner    │-->| Researcher   │-->| Writer  │-->| Critic*    │
└────────┘   └────────────┘   └──────────────┘   └─────────┘   └────┬───────┘
                                                              run?  │skip
                                                                    v
                                                                 ┌──────┐
                                                                 │ End  │
                                                                 └──────┘
```

1. **Planner Agent**: builds a JSON plan with ordered steps and evidence requirements.
2. **Researcher Agent**: rewrites each step into search queries and performs vector retrieval via the injected `vector.VectorStore` + `vector.Embedder`.
3. **Writer Agent**: receives the plan plus retrieved evidence to craft a draft answer that cites documents.
4. **Critic Agent** (optional): validates the draft, lists issues, and can produce an improved final answer.

Each agent is backed by its own `agent.LLMClient`. Missing clients automatically fall back to `Clients.Default`, so you can power every agent with a single provider or mix models per task.

## Four-Stage Flow

The repository now exposes first-class packages that mirror the canonical RAG lifecycle:

1. **Data Preparation (`rag/document`, `rag/chunking`)**
   Represent raw sources as `document.Document`, then split them into manageable `document.Chunk` slices via `chunking.SimpleChunker` or a custom `chunking.Chunker`.

2. **Index Construction (`rag/embedder`, `rag/retriever`)**
   Feed each chunk through an `embedder.Embedder` (e.g., `embedder.NewVectorAdapter` wrapping any `vector.Embedder`) and persist vectors using `retriever.IndexDocuments`, which writes to the configured `vector.VectorStore`.

3. **Query + Retrieval (`rag/retriever`, `rag/reranker`)**
   When questions arrive, `retriever.Search` embeds the query, performs similarity search, and optionally refines results with a `reranker.Reranker` such as `reranker.NewCosineReranker`.

4. **Generation Integration (`rag/agentic`)**
   The Agentic pipeline consumes the reranked evidence, routes it through planner/researcher/writer/critic agents, and produces auditable responses.

Because each stage lives in its own package you can replace any component (custom chunker, hybrid retriever, cross-encoder reranker, etc.) without touching the rest of the flow.

## Quick Start

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/sweetpotato0/ai-allin/contrib/provider/openai"
	"github.com/sweetpotato0/ai-allin/rag/agentic"
	vectorstore "github.com/sweetpotato0/ai-allin/vector/store"
)

func main() {
	ctx := context.Background()
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY is required")
	}

	llm := openai.New(openai.DefaultConfig(apiKey))
	store := vectorstore.NewInMemoryVectorStore()
	embedder := newKeywordEmbedder() // see examples/rag/agentic for a placeholder implementation

	pipeline, err := agentic.NewPipeline(
		agentic.Clients{Default: llm}, // planner, writer, critic reuse the same provider
		embedder,
		store,
		agentic.WithTopK(3),
	)
	if err != nil {
		log.Fatalf("build pipeline: %v", err)
	}

	err = pipeline.IndexDocuments(ctx,
		agentic.Document{ID: "shipping", Title: "Shipping Policy", Content: "..."},
		agentic.Document{ID: "returns", Title: "Return Policy", Content: "..."},
	)
	if err != nil {
		log.Fatalf("index docs: %v", err)
	}

	response, err := pipeline.Run(ctx, "Summarize the shipping timeline and return policy.")
	if err != nil {
		log.Fatalf("pipeline run: %v", err)
	}

	log.Println("Plan steps:", len(response.Plan.Steps))
	log.Println("Draft:", response.DraftAnswer)
	log.Println("Final:", response.FinalAnswer)
}
```

## Customisation

- **Document ingestion** – use `IndexDocuments`, `ClearDocuments`, and `CountDocuments` to control the knowledge base. Documents can carry arbitrary metadata for downstream auditing.
- **Retrieval depth** – `agentic.WithTopK(k)` / `agentic.WithRerankTopK(k)` control search fan-out and reranker cutoffs.
- **Chunking & reranking** – swap in `agentic.WithChunker(...)` or `agentic.WithReranker(...)` to control how data is prepared and scored.
- **Bring your own retriever** – inject any retrieval implementation (hybrid search, external service, etc.) via `agentic.WithRetriever(...)`.
- **Retrieval presets** – call `agentic.WithRetrievalPreset(agentic.RetrievalPresetSimple|Balanced|Hybrid)` to flip multiple tuning knobs at once instead of setting every field manually.
- **Prompts** – override planner/query/writer/critic prompts with `WithPlannerPrompt`, `WithQueryPrompt`, `WithSynthesisPrompt`, and `WithCriticPrompt`.
- **Query agent** – use `WithQueryRetries`, `WithQueryMaxResults`, and `WithQueryCaching` to control how the researcher LLM generates and caches search terms.
- **Answer safety** – demand supporting evidence with `WithMinEvidenceCount(n)` and customise the fallback `WithNoAnswerMessage(...)` so the writer refuses to answer when nothing relevant was found.
- **Critic agent** – disable it via `WithCritic(false)` or supply a different LLM client through `Clients.Critic`.
- **Graph extensions** – the underlying `graph.Graph` is stored on the pipeline; you can fork the package or wrap the pipeline to inject extra nodes (tool calls, structured logging, telemetry, etc.).

### Production-grade components

The `contrib/` tree now ships ready-to-use upgrades:

- `contrib/chunking/markdown` keeps headings with their body text and tags section metadata, while `contrib/chunking/token` enforces token-aware windows compatible with LLM limits.
- `contrib/reranker/mmr` removes duplicate evidence via Max Marginal Relevance, and `contrib/reranker/cohere` calls Cohere’s hosted ReRank API with automatic local fallback.
- `contrib/retrieval/hybrid` merges semantic vectors with a lightweight BM25 index so lexical matches (dates, identifiers) survive, and can be injected via `agentic.WithRetriever`.
- `examples/rag/production` demonstrates wiring these pieces together; point it at real LLM/embedding providers for a production-like stack.

## Observability

`pipeline.Run` returns a rich `agentic.Response`:

- `Plan`: the planner's structured breakdown.
- `Evidence`: per-step document matches with scores and summaries.
- `DraftAnswer` & `FinalAnswer`: before/after critic review.
- `Critic`: verdict, issues, and revised answer if applicable.

Persist this struct or attach it to your analytics pipeline to understand how each answer was produced. The explicit plan/evidence trail makes it easier to debug regressions and to plug in human reviewers when needed.
