package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/contrib/provider/openai"
)

func main() {
	fmt.Println("=== Basic Agent Example ===")

	ctx := context.Background()
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY is required to run the Agentic RAG example")
	}

	baseURL := os.Getenv("OPENAI_API_BASE_URL")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY is required to run the Agentic RAG example")
	}
	llm := openai.New(openai.DefaultConfig().WithAPIKey(apiKey).WithBaseURL(baseURL))

	// Create agent with options pattern
	ag := agent.New(
		agent.WithName("MyAssistant"),
		agent.WithSystemPrompt("You are a helpful AI assistant"),
		agent.WithProvider(llm),
		agent.WithMaxIterations(5),
		agent.WithTemperature(0.7),
	)

	// Run the agent
	result, err := ag.Run(ctx, "Hello, how are you?")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Printf("User: Hello, how are you?\n")
	fmt.Printf("Agent: %s\n", result.Text())

	// Show message history
	fmt.Println("\nMessage History:")
	for i, msg := range ag.GetMessages() {
		fmt.Printf("%d. [%s] %s\n", i+1, msg.Role, msg.Text())
	}
}
