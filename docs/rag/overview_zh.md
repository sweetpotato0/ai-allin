# Agentic RAG 概览

`rag/agentic` 在基础 Agent 框架之上提供一套主见鲜明的多智能体检索增强生成（RAG）流水线，专为希望在生产环境中获得可观测、可审计的规划 / 检索 / 推理 / 评审体验的团队设计。

## 为什么选择 Agentic RAG？

- **确定性的规划**：规划智能体会将用户问题拆分为可审计的步骤。
- **检索透明度**：每一步都记录触发的查询以及命中的文档。
- **高质量写作**：专用的写作智能体会把证据组织成结构化回答。
- **安全兜底**：可选的审稿智能体会校验草稿并在需要时修改答案。
- **原生图式编排**：流水线基于 `graph.Graph` 实现，便于扩展自定义节点或替换任意智能体。

## 架构

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

1. **Planner**：输出带步骤与证据需求的 JSON 计划。
2. **Researcher**：把每个步骤转成检索查询，并通过注入的 `vector.VectorStore` + `vector.Embedder` 执行向量检索。
3. **Writer**：接收计划与证据，生成引用文档的草稿回答。
4. **Critic（可选）**：复核草稿、列出问题，并可给出改进后的终稿。

每个智能体都可以绑定不同的 `agent.LLMClient`。若某个角色未显式提供客户端，会自动回退到 `Clients.Default`，方便按需混用模型。

## 四阶段流程

仓库暴露了与经典 RAG 生命周期相对应的一等包：

1. **数据准备（`rag/document`, `rag/chunking`）**
   使用 `document.Document` 表示原始资料，再通过 `chunking.SimpleChunker` 或自定义 `chunking.Chunker` 切分为 `document.Chunk`。

2. **索引构建（`rag/embedder`, `rag/retriever`）**
   将每个 chunk 送入 `embedder.Embedder`（例如 `embedder.NewVectorAdapter` 包裹任何 `vector.Embedder`），再通过 `retriever.IndexDocuments` 写入指定的 `vector.VectorStore`。

3. **查询与检索（`rag/retriever`, `rag/reranker`）**
   用户提问时，`retriever.Search` 会生成查询向量、执行相似度搜索，并可借助 `reranker.Reranker`（如 `reranker.NewCosineReranker`）对结果重排。

4. **生成集成（`rag/agentic`）**
   Agentic 流水线消费重排后的证据，依次调度 planner / researcher / writer / critic，产出可追溯的回答。

由于各阶段都封装在独立包中，你可以随时替换 chunker、retriever、reranker 等实现，而无需改动其它流程。

## 快速开始

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
	embedder := newKeywordEmbedder() // 演示用嵌入器，参阅 examples/rag/agentic

	pipeline, err := agentic.NewPipeline(
		agentic.Clients{Default: llm}, // planner / writer / critic 共用同一提供商
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

## 自定义指南

- **文档入库**：通过 `IndexDocuments` / `ClearDocuments` / `CountDocuments` 管理知识库；可在 `Document.Metadata` 中挂载任意元数据。
- **检索深度**：使用 `agentic.WithTopK(k)` / `agentic.WithRerankTopK(k)` 控制召回与重排的宽度。
- **切片与重排**：可注入 `agentic.WithChunker(...)` 或 `agentic.WithReranker(...)` 调整切片策略与重排算法。
- **自带检索器**：若已有自研搜索服务，可借助 `agentic.WithRetriever(...)` 直接注入，跳过默认的 chunk/embed 流程。
- **提示词**：用 `WithPlannerPrompt` / `WithQueryPrompt` / `WithSynthesisPrompt` / `WithCriticPrompt` 覆盖各角色的系统提示。
- **审稿智能体**：通过 `WithCritic(false)` 关闭，或给 `Clients.Critic` 指定不同模型。
- **图扩展**：底层 `graph.Graph` 可随意扩展节点，用于插入工具调用、链路追踪、遥测等逻辑。

## 可观测性

`pipeline.Run` 会返回结构化的 `agentic.Response`：

- `Plan`：规划结果及其步骤。
- `Evidence`：每个步骤匹配到的文档 / chunk 及其得分与摘要。
- `DraftAnswer` / `FinalAnswer`：写作输出与审稿后的终稿。
- `Critic`：审稿结论、问题列表以及（可选）修改后的回答。

你可以把这些数据写入观测系统或分析流水线，快速定位回答生成过程中的问题，并在需要时插入人工审核。
