package agentic

import "github.com/sweetpotato0/ai-allin/rag/document"

// Document re-exports the rag/document type for backwards compatibility.
type Document = document.Document

// Plan captures the multi-agent research plan emitted by the planner agent.
type Plan struct {
	Strategy string     `json:"strategy"`
	Steps    []PlanStep `json:"steps"`
}

// PlanStep enumerates one actionable task in the plan.
type PlanStep struct {
	ID                string   `json:"id"`
	Goal              string   `json:"goal"`
	Questions         []string `json:"questions,omitempty"`
	ExpectedEvidence  string   `json:"expected_evidence,omitempty"`
	DownstreamSupport string   `json:"downstream_support,omitempty"`
}

// Evidence links a retrieval result (document) to the plan step that needed it.
type Evidence struct {
	StepID   string             `json:"step_id"`
	Query    string             `json:"query"`
	Chunk    document.Chunk     `json:"chunk"`
	Document *document.Document `json:"document,omitempty"`
	Score    float32            `json:"score"`
	Summary  string             `json:"summary,omitempty"`
}

// CriticFeedback is produced by the critic agent when the pipeline is configured
// to run quality checks.
type CriticFeedback struct {
	Verdict     string   `json:"verdict"`
	Issues      []string `json:"issues,omitempty"`
	Notes       string   `json:"notes,omitempty"`
	FinalAnswer string   `json:"final_answer,omitempty"`
}

// Response captures the structured pipeline result that applications consume.
type Response struct {
	Question    string          `json:"question"`
	Plan        *Plan           `json:"plan,omitempty"`
	Evidence    []Evidence      `json:"evidence,omitempty"`
	DraftAnswer string          `json:"draft_answer,omitempty"`
	FinalAnswer string          `json:"final_answer,omitempty"`
	Critic      *CriticFeedback `json:"critic,omitempty"`
}
