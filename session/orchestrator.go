package session

import (
	"context"

	"github.com/sweetpotato0/ai-allin/agent"
)

// Orchestrator coordinates conversations across multiple agents.
type Orchestrator struct {
	conversations *ConversationManager
}

// NewOrchestrator creates a new orchestrator with in-memory storage.
func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		conversations: NewConversationManager(),
	}
}

// Run executes the specified agent against the shared conversation.
func (o *Orchestrator) Run(ctx context.Context, sessionID string, ag *agent.Agent, input string) (string, error) {
	conv := o.conversations.GetOrCreate(sessionID)
	return conv.RunAgent(ctx, ag, input)
}

// Close terminates the conversation and releases resources.
func (o *Orchestrator) Close(sessionID string) {
	o.conversations.Delete(sessionID)
}
