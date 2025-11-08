package agentic

import "github.com/sweetpotato0/ai-allin/rag/document"

// Document re-exports the rag/document type for backwards compatibility.
type Document = document.Document

// Plan captures the multi-agent research plan emitted by the planner agent.
type Plan struct {
	Strategy string     `json:"strategy"` // High-level approach for the end-to-end answer
	Steps    []PlanStep `json:"steps"`    // Ordered tasks the pipeline should execute
}

// PlanStep enumerates one actionable task in the plan.
type PlanStep struct {
	ID                string   `json:"id"`                          // Stable identifier used in evidence records
	Goal              string   `json:"goal"`                        // Human readable objective
	Questions         []string `json:"questions,omitempty"`         // Extra clarifying questions / query hints
	ExpectedEvidence  string   `json:"expected_evidence,omitempty"` // What signal or doc types unlock the step
	DownstreamSupport string   `json:"downstream_support,omitempty"`
}

// Evidence links a retrieval result (document) to the plan step that needed it.
type Evidence struct {
	StepID   string             `json:"step_id"`            // Which plan step this chunk supports
	Query    string             `json:"query"`              // Query used to fetch the chunk
	Chunk    document.Chunk     `json:"chunk"`              // Retrieved chunk payload
	Document *document.Document `json:"document,omitempty"` // Optional parent document metadata
	Score    float32            `json:"score"`              // Similarity score after reranking
	Summary  string             `json:"summary,omitempty"`  // Short summary fed to downstream agents
}

// CriticFeedback is produced by the critic agent when the pipeline is configured
// to run quality checks.
type CriticFeedback struct {
	Verdict     string   `json:"verdict"`                // approve | revise
	Issues      []string `json:"issues,omitempty"`       // Concrete problems spotted by critic
	Notes       string   `json:"notes,omitempty"`        // Free-form explanation
	FinalAnswer string   `json:"final_answer,omitempty"` // Final answer (may equal draft)
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
