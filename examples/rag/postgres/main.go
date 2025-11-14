package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	openaisdk "github.com/openai/openai-go/v3"
	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/contrib/chunking/markdown"
	openai_embedder "github.com/sweetpotato0/ai-allin/contrib/embedder/openai"
	"github.com/sweetpotato0/ai-allin/contrib/provider/openai"
	pgvector "github.com/sweetpotato0/ai-allin/contrib/vector/pg"
	"github.com/sweetpotato0/ai-allin/rag/agentic"
)

const (
	defaultQuestion     = "Give me a concise overview of the repository architecture and how the MCP integration works."
	defaultDocsDir      = "testdata/cases"
	defaultPGTable      = "repo_docs"
	embeddingModel      = openaisdk.EmbeddingModelTextEmbedding3Small
	embeddingDimensions = 1536
	defaultChatModel    = "gpt-4o-mini"
	envOpenAIKey        = "OPENAI_API_KEY"
	envOpenAIBaseURL    = "OPENAI_API_BASE_URL"
	envPGHost           = "PGVECTOR_HOST"
	envPGPort           = "PGVECTOR_PORT"
	envPGUser           = "PGVECTOR_USER"
	envPGPassword       = "PGVECTOR_PASSWORD"
	envPGDatabase       = "PGVECTOR_DATABASE"
	envPGSSLMode        = "PGVECTOR_SSLMODE"
	envPGTable          = "PGVECTOR_TABLE"
)

func main() {
	var (
		question = flag.String("question", defaultQuestion, "Question to ask the RAG pipeline")
		docsDir  = flag.String("docs", defaultDocsDir, "Directory containing markdown docs to index")
		reindex  = flag.Bool("reindex", false, "Force re-indexing even if documents already exist")
		table    = flag.String("table", envOr(envPGTable, defaultPGTable), "Postgres table name for vectors")
	)
	flag.Parse()

	apiKey := os.Getenv(envOpenAIKey)
	if strings.TrimSpace(apiKey) == "" {
		log.Fatalf("%s is required", envOpenAIKey)
	}

	ctx := context.Background()

	embedder := openai_embedder.New(apiKey, os.Getenv(envOpenAIBaseURL), embeddingModel, embeddingDimensions)
	pgStore, err := newPGStore(*table, embedder.Dimension())
	if err != nil {
		log.Fatalf("connect Postgres: %v", err)
	}
	defer pgStore.Close()

	llm := buildChatClient(apiKey, os.Getenv(envOpenAIBaseURL), defaultChatModel)

	pipeline, err := agentic.NewPipeline(
		agentic.Clients{Default: llm},
		embedder,
		pgStore,
		agentic.WithTopK(1),
		agentic.WithChunker(markdown.New()),
		agentic.WithHybridSearch(true),
	)
	if err != nil {
		log.Fatalf("build pipeline: %v", err)
	}

	documents, err := collectDocuments(*docsDir)
	if err != nil {
		log.Fatalf("collect docs: %v", err)
	}

	needsIndex := *reindex
	if !needsIndex {
		count, err := pipeline.CountDocuments(ctx)
		if err != nil {
			log.Fatalf("count documents: %v", err)
		}
		needsIndex = count == 0 || count < len(documents)
	}

	if needsIndex {
		log.Printf("Indexing %d documents into table %s ...", len(documents), *table)
		if err := pipeline.ClearDocuments(ctx); err != nil {
			log.Fatalf("clear documents: %v", err)
		}
		if err := pipeline.IndexDocuments(ctx, documents...); err != nil {
			log.Fatalf("index documents: %v", err)
		}
	} else {
		log.Printf("Skipped indexing (%d documents already present). Use -reindex to force.", len(documents))
	}

	log.Printf("Running Agentic RAG pipeline for question: %q", *question)
	start := time.Now()
	response, err := pipeline.Run(ctx, *question)
	if err != nil {
		log.Fatalf("pipeline run failed: %v", err)
	}
	log.Printf("Pipeline completed in %s\n", time.Since(start))

	printResponse(response)
}

func buildChatClient(apiKey, baseURL, model string) agent.LLMClient {
	cfg := openai.DefaultConfig().WithAPIKey(apiKey).WithBaseURL(baseURL).WithModel(model)
	return openai.New(cfg)
}

func newPGStore(table string, dimension int) (*pgvector.PGVectorStore, error) {
	cfg := pgvector.DefaultPGVectorConfig()
	cfg.TableName = table
	cfg.Dimension = dimension
	cfg.Host = envOr(envPGHost, cfg.Host)
	cfg.User = envOr(envPGUser, cfg.User)
	cfg.Password = envOr(envPGPassword, cfg.Password)
	cfg.DBName = envOr(envPGDatabase, cfg.DBName)
	cfg.SSLMode = envOr(envPGSSLMode, cfg.SSLMode)
	cfg.IndexType = "HNSW"

	if port := envOr(envPGPort, ""); port != "" {
		fmt.Sscanf(port, "%d", &cfg.Port)
	}

	return pgvector.NewPGVectorStore(cfg)
}

func envOr(key, fallback string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}
	return fallback
}

func printResponse(resp *agentic.Response) {
	if resp == nil {
		return
	}

	fmt.Printf("\nQuestion: %s\n", resp.Question)
	if resp.Plan != nil {
		fmt.Printf("\nPlan (%d steps):\n", len(resp.Plan.Steps))
		for _, step := range resp.Plan.Steps {
			fmt.Printf(" - %s: %s\n", step.ID, step.Goal)
		}
	}

	if len(resp.Evidence) > 0 {
		fmt.Printf("\nEvidence (%d chunks):\n", len(resp.Evidence))
		for _, ev := range resp.Evidence {
			fmt.Printf(" • Step %s matched %s (score %.2f)\n", ev.StepID, ev.Document.Title, ev.Score)
		}
	}

	fmt.Printf("\nDraft answer:\n%s\n", resp.DraftAnswer)
	fmt.Printf("\nFinal answer:\n%s\n", resp.FinalAnswer)
}
