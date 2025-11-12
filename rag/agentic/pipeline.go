package agentic

import (
	"context"
	"fmt"
	"strings"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/graph"
	"github.com/sweetpotato0/ai-allin/rag/document"
	"github.com/sweetpotato0/ai-allin/vector"
)

const ragStateKey = "__agentic_rag_state"

// Clients groups the LLM clients used by the different pipeline agents.
type Clients struct {
	Default    agent.LLMClient
	Planner    agent.LLMClient
	Researcher agent.LLMClient
	Writer     agent.LLMClient
	Critic     agent.LLMClient
}

// Pipeline wires the multi-agent RAG workflow together.
// Internally it manages three stages:
//  1. planning/query generation
//  2. retrieval (delegated to RetrievalEngine)
//  3. synthesis + optional critique
//
// Each stage only depends on the data provided by the previous one which keeps the
// execution graph easy to reason about.
type Pipeline struct {
	cfg        *Config
	planner    *planner
	researcher *researcher
	writer     *synthesizer
	critic     *critic
	retrieval  RetrievalEngine
	graph      *graph.Graph
}

type pipelineState struct {
	Question string          // Original user question
	Plan     *Plan           // Plan produced by planner node
	Evidence []Evidence      // Collected evidence per step
	Draft    string          // Writer response before critique
	Critic   *CriticFeedback // Optional critic verdict
}

// NewPipeline creates a fully wired Agentic RAG pipeline.
func NewPipeline(clients Clients, embedder vector.Embedder, store vector.VectorStore, opts ...Option) (*Pipeline, error) {
	cfg := applyOptions(nil, opts)

	plannerLLM := pickClient(clients.Planner, clients.Default)
	writerLLM := pickClient(clients.Writer, clients.Default)
	if plannerLLM == nil {
		return nil, fmt.Errorf("planner client is required")
	}
	if writerLLM == nil {
		return nil, fmt.Errorf("writer client is required")
	}

	engine := cfg.retrieval
	if engine == nil {
		var err error
		engine, err = newDefaultRetrievalEngine(store, embedder, cfg)
		if err != nil {
			return nil, err
		}
	}

	p := &Pipeline{
		cfg:        cfg,
		planner:    newPlanner(plannerLLM, cfg),
		researcher: newResearcher(pickClient(clients.Researcher, clients.Default), cfg),
		writer:     newSynthesizer(writerLLM, cfg),
		critic:     nil,
		retrieval:  engine,
	}
	if cfg.EnableCritic {
		p.critic = newCritic(pickClient(clients.Critic, clients.Default), cfg)
	}

	builder := graph.NewBuilder().
		AddNode("start", graph.NodeTypeStart, p.startNode).
		AddNode("planner", graph.NodeTypeLLM, p.planNode).
		AddNode("research", graph.NodeTypeTool, p.researchNode).
		AddNode("synthesis", graph.NodeTypeLLM, p.synthesizeNode).
		AddConditionNode("critic_gate", p.criticGate, map[string]string{
			"run":  "critic",
			"skip": "end",
		}).
		AddNode("critic", graph.NodeTypeLLM, p.criticNode).
		AddNode("end", graph.NodeTypeEnd, p.endNode).
		AddEdge("start", "planner").
		AddEdge("planner", "research").
		AddEdge("research", "synthesis").
		AddEdge("synthesis", "critic_gate").
		AddEdge("critic", "end").
		SetStart("start").
		SetEnd("end")

	g := builder.Build()
	g.SetMaxVisits(cfg.GraphMaxVisits)
	p.graph = g
	return p, nil
}

func pickClient(primary, fallback agent.LLMClient) agent.LLMClient {
	if primary != nil {
		return primary
	}
	return fallback
}

// Run executes the pipeline for a new question.
func (p *Pipeline) Run(ctx context.Context, question string) (*Response, error) {
	if strings.TrimSpace(question) == "" {
		return nil, fmt.Errorf("question cannot be empty")
	}

	initial := graph.State{
		ragStateKey: &pipelineState{
			Question: strings.TrimSpace(question),
		},
	}

	finalState, err := p.graph.Execute(ctx, initial)
	if err != nil {
		return nil, err
	}

	state, err := getState(finalState)
	if err != nil {
		return nil, err
	}

	resp := &Response{
		Question:    state.Question,
		Plan:        state.Plan,
		Evidence:    state.Evidence,
		DraftAnswer: state.Draft,
		FinalAnswer: state.Draft,
		Critic:      state.Critic,
	}
	if state.Critic != nil && state.Critic.FinalAnswer != "" {
		resp.FinalAnswer = state.Critic.FinalAnswer
	}
	return resp, nil
}

// IndexDocuments ingests documents into the vector store.
// IndexDocuments chunks and embeds documents through the configured retrieval engine.
func (p *Pipeline) IndexDocuments(ctx context.Context, docs ...Document) error {
	casts := make([]document.Document, len(docs))
	for i, doc := range docs {
		if strings.TrimSpace(doc.Content) == "" {
			return fmt.Errorf("document content cannot be empty")
		}
		casts[i] = document.Document{
			ID:       doc.ID,
			Title:    doc.Title,
			Content:  doc.Content,
			Metadata: cloneMetadata(doc.Metadata),
		}
	}
	if len(casts) == 0 {
		return nil
	}
	return p.retrieval.IndexDocuments(ctx, casts...)
}

// ClearDocuments removes all indexed documents.
func (p *Pipeline) ClearDocuments(ctx context.Context) error {
	return p.retrieval.Clear(ctx)
}

// CountDocuments returns the number of indexed documents.
func (p *Pipeline) CountDocuments(ctx context.Context) (int, error) {
	return p.retrieval.Count(ctx)
}

func (p *Pipeline) startNode(ctx context.Context, state graph.State) (graph.State, error) {
	_, err := getState(state)
	return state, err
}

func (p *Pipeline) planNode(ctx context.Context, state graph.State) (graph.State, error) {
	st, err := getState(state)
	if err != nil {
		return state, err
	}

	plan, err := p.planner.Plan(ctx, st.Question)
	if err != nil {
		return state, err
	}
	st.Plan = plan
	return state, nil
}

func (p *Pipeline) researchNode(ctx context.Context, state graph.State) (graph.State, error) {
	st, err := getState(state)
	if err != nil {
		return state, err
	}
	if st.Plan == nil {
		return state, fmt.Errorf("plan not available for research node")
	}

	collected := make([]Evidence, 0)
	type evidenceKey struct {
		step string
		doc  string
	}
	index := make(map[evidenceKey]int)

	for _, step := range st.Plan.Steps {
		queries, err := p.researcher.buildQueries(ctx, st.Question, step)
		if err != nil {
			return state, err
		}
		for _, q := range queries {
			fmt.Printf("当前查询：%#v\n", q)
			results, err := p.retrieval.Search(ctx, q)
			if err != nil {
				return state, fmt.Errorf("vector search failed: %w", err)
			}
			for _, candidate := range results {
				doc, ok := p.retrieval.Document(candidate.Chunk.DocumentID)
				if !ok {
					continue
				}
				score := candidate.Score
				key := evidenceKey{step: step.ID, doc: doc.ID}
				if idx, ok := index[key]; ok {
					if score > collected[idx].Score {
						collected[idx].Score = score
						collected[idx].Query = q
					}
					continue
				}
				ev := Evidence{
					StepID:   step.ID,
					Query:    q,
					Document: &doc,
					Chunk:    candidate.Chunk,
					Score:    score,
					Summary:  summarizeChunk(candidate.Chunk, 320),
				}
				index[key] = len(collected)
				collected = append(collected, ev)
			}
		}
	}

	st.Evidence = collected
	return state, nil
}

func (p *Pipeline) synthesizeNode(ctx context.Context, state graph.State) (graph.State, error) {
	st, err := getState(state)
	if err != nil {
		return state, err
	}
	required := p.cfg.MinEvidenceCount
	if required < 0 {
		required = 0
	}
	if len(st.Evidence) < required {
		fallback := strings.TrimSpace(p.cfg.NoAnswerMessage)
		if fallback == "" {
			fallback = "No supporting evidence was found for this question."
		}
		st.Draft = fallback
		return state, nil
	}
	draft, err := p.writer.Compose(ctx, st.Question, st.Plan, st.Evidence)
	if err != nil {
		return state, err
	}
	st.Draft = draft
	return state, nil
}

func (p *Pipeline) criticGate(ctx context.Context, state graph.State) (string, error) {
	if !p.cfg.EnableCritic || p.critic == nil {
		return "skip", nil
	}
	return "run", nil
}

func (p *Pipeline) criticNode(ctx context.Context, state graph.State) (graph.State, error) {
	st, err := getState(state)
	if err != nil {
		return state, err
	}
	if p.critic == nil {
		return state, nil
	}
	feedback, err := p.critic.Review(ctx, st.Question, st.Draft, st.Plan, st.Evidence)
	if err != nil {
		return state, err
	}
	st.Critic = feedback
	return state, nil
}

func (p *Pipeline) endNode(ctx context.Context, state graph.State) (graph.State, error) {
	_, err := getState(state)
	return state, err
}

func getState(state graph.State) (*pipelineState, error) {
	raw, ok := state[ragStateKey]
	if !ok {
		return nil, fmt.Errorf("rag state missing in graph")
	}
	ps, ok := raw.(*pipelineState)
	if !ok {
		return nil, fmt.Errorf("invalid rag state type")
	}
	return ps, nil
}

func cloneMetadata(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func summarizeChunk(chunk document.Chunk, limit int) string {
	text := strings.TrimSpace(chunk.Content)
	if limit <= 0 || len(text) <= limit {
		return text
	}
	return text[:limit] + "..."
}
