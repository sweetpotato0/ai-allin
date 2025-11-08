package main

import (
	"context"
	"fmt"
	"log"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/contrib/session/inmemory"
	"github.com/sweetpotato0/ai-allin/message"
	"github.com/sweetpotato0/ai-allin/session"
)

func main() {
	ctx := context.Background()

	// Create session manager with in-memory store
	mgr := session.NewManager(session.WithStore(inmemory.NewInMemoryStore()))
	sessionID := "shared-conversation"

	researcher := agent.New(
		agent.WithName("researcher"),
		agent.WithSystemPrompt("You gather context."),
		agent.WithProvider(&echoProvider{tag: "researcher"}),
	)

	solver := agent.New(
		agent.WithName("solver"),
		agent.WithSystemPrompt("You provide solutions."),
		agent.WithProvider(&echoProvider{tag: "solver"}),
	)

	// Create a shared session for multi-agent collaboration
	sharedSess, err := mgr.CreateShared(ctx, sessionID)
	if err != nil {
		log.Fatalf("Failed to create shared session: %v", err)
	}

	// Method 1: Use shared session with different agents
	fmt.Println("=== Method 1: Using Shared Session with Different Agents ===")
	runWithAgent := func(name string, ag *agent.Agent, input string) {
		resp, err := sharedSess.RunWithAgent(ctx, ag, input)
		if err != nil {
			log.Fatalf("run failed: %v", err)
		}
		fmt.Printf("[%s] input: %s\n", name, input)
		fmt.Printf("[%s] response: %s\n\n", name, resp)
	}

	runWithAgent("researcher", researcher, "User needs a knowledge base for customer support.")
	runWithAgent("solver", solver, "Summarize previous context and suggest next steps.")
	runWithAgent("researcher", researcher, "Collect missing requirements based on the solver's suggestion.")

	// Persist the shared session snapshot for analytics/persistence
	if err := mgr.Save(ctx, sharedSess); err != nil {
		log.Fatalf("Failed to save shared session: %v", err)
	}
	snap := sharedSess.Snapshot()
	fmt.Printf("Shared session snapshot captured with %d messages (last turn %s)\n", len(snap.Messages), snap.LastDuration)

	// Method 2: Create a new shared session for another conversation
	fmt.Println("\n=== Method 2: Using Another Shared Session ===")
	sharedSess2, err := mgr.CreateShared(ctx, sessionID+"-2")
	if err != nil {
		log.Fatalf("Failed to create shared session: %v", err)
	}

	runWithAgent2 := func(name string, ag *agent.Agent, input string) {
		resp, err := sharedSess2.RunWithAgent(ctx, ag, input)
		if err != nil {
			log.Fatalf("run failed: %v", err)
		}
		fmt.Printf("[%s] input: %s\n", name, input)
		fmt.Printf("[%s] response: %s\n\n", name, resp)
	}

	runWithAgent2("researcher", researcher, "What are the key requirements?")
	runWithAgent2("solver", solver, "Based on the requirements, what's the solution?")

	// Method 3: Get existing shared session and continue
	fmt.Println("\n=== Method 3: Getting Existing Shared Session ===")
	existingSess, err := mgr.Get(ctx, sessionID)
	if err != nil {
		log.Fatalf("Failed to get session: %v", err)
	}

	if sharedSess3, ok := existingSess.(*session.SharedSession); ok {
		runWithAgent3 := func(name string, ag *agent.Agent, input string) {
			resp, err := sharedSess3.RunWithAgent(ctx, ag, input)
			if err != nil {
				log.Fatalf("run failed: %v", err)
			}
			fmt.Printf("[%s] input: %s\n", name, input)
			fmt.Printf("[%s] response: %s\n\n", name, resp)
		}

		runWithAgent3("researcher", researcher, "Direct agent passing example.")
		runWithAgent3("solver", solver, "Another direct agent passing example.")
	}
}

type echoProvider struct {
	tag string
}

func (e *echoProvider) Generate(ctx context.Context, msgs []*message.Message, tools []map[string]any) (*message.Message, error) {
	turn := len(msgs)
	last := msgs[turn-1].Content
	return message.NewMessage(message.RoleAssistant, fmt.Sprintf("%s sees turn %d: %s", e.tag, turn, last)), nil
}

func (e *echoProvider) SetTemperature(float64) {}
func (e *echoProvider) SetMaxTokens(int64)     {}
func (e *echoProvider) SetModel(string)        {}
