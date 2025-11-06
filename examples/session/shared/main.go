package main

import (
	"context"
	"fmt"
	"log"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/message"
	"github.com/sweetpotato0/ai-allin/session"
)

func main() {
	ctx := context.Background()
	orchestrator := session.NewOrchestrator()
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

	run := func(name string, ag *agent.Agent, input string) {
		resp, err := orchestrator.Run(ctx, sessionID, ag, input)
		if err != nil {
			log.Fatalf("run failed: %v", err)
		}
		fmt.Printf("[%s] input: %s\n", name, input)
		fmt.Printf("[%s] response: %s\n\n", name, resp)
	}

	run("researcher", researcher, "User needs a knowledge base for customer support.")
	run("solver", solver, "Summarize previous context and suggest next steps.")
	run("researcher", researcher, "Collect missing requirements based on the solver's suggestion.")
}

type echoProvider struct {
	tag string
}

func (e *echoProvider) Generate(ctx context.Context, msgs []*message.Message, tools []map[string]interface{}) (*message.Message, error) {
	turn := len(msgs)
	last := msgs[turn-1].Content
	return message.NewMessage(message.RoleAssistant, fmt.Sprintf("%s sees turn %d: %s", e.tag, turn, last)), nil
}

func (e *echoProvider) SetTemperature(float64) {}
func (e *echoProvider) SetMaxTokens(int64)     {}
func (e *echoProvider) SetModel(string)        {}
