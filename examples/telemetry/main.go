package main

import (
	"context"
	"fmt"
	"log"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/message"
	"github.com/sweetpotato0/ai-allin/pkg/telemetry"
)

func main() {
	ctx := context.Background()

	shutdown, err := telemetry.Init(ctx, telemetry.Config{
		ServiceName:    "ai-allin-example",
		ServiceVersion: "v0.1.0",
		Environment:    "example",
	})
	if err != nil {
		log.Fatalf("init telemetry: %v", err)
	}
	defer shutdown(context.Background())

	ag := agent.New(
		agent.WithName("telemetry-agent"),
		agent.WithSystemPrompt("You are a helpful assistant."),
		agent.WithProvider(echoLLM{}),
	)

	resp, err := ag.Run(ctx, "ping")
	if err != nil {
		log.Fatalf("agent run failed: %v", err)
	}

	fmt.Println("assistant:", resp.Text())
}

type echoLLM struct{}

func (echoLLM) Generate(ctx context.Context, req *agent.GenerateRequest) (*agent.GenerateResponse, error) {
	var last string
	if len(req.Messages) > 0 {
		last = req.Messages[len(req.Messages)-1].Text()
	}
	msg := message.NewMessage(message.RoleAssistant, fmt.Sprintf("echo:%s", last))
	return &agent.GenerateResponse{Message: msg}, nil
}

func (echoLLM) SetTemperature(float64) {}
func (echoLLM) SetMaxTokens(int64)     {}
func (echoLLM) SetModel(string)        {}
