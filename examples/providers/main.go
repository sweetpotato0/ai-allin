package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/contrib/provider/claude"
	"github.com/sweetpotato0/ai-allin/contrib/provider/openai"
)

func main() {
	fmt.Println("=== LLM Provider Examples ===\n")

	// Example 1: Using OpenAI Provider
	fmt.Println("Example 1: OpenAI Provider")
	openaiProviderExample()

	// Example 2: Using Claude Provider
	fmt.Println("\nExample 2: Claude Provider")
	claudeProviderExample()

	// Example 3: Provider Configuration
	fmt.Println("\nExample 3: Provider Configuration")
	providerConfigurationExample()
}

func openaiProviderExample() {
	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("⚠️  OPENAI_API_KEY not set - skipping example")
		fmt.Println("   Set OPENAI_API_KEY environment variable to use this example")
		return
	}

	ctx := context.Background()

	// Create OpenAI provider with default config
	openaiConfig := openai.DefaultConfig(apiKey)
	openaiProvider := openai.New(openaiConfig)

	// Create agent with OpenAI provider
	ag := agent.New(
		agent.WithName("OpenAI Assistant"),
		agent.WithSystemPrompt("You are a helpful assistant powered by OpenAI."),
		agent.WithProvider(openaiProvider),
		agent.WithTemperature(0.7),
	)

	// Run the agent
	result, err := ag.Run(ctx, "Hello, can you tell me a brief joke?")
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("OpenAI Response: %s\n", result)
}

func claudeProviderExample() {
	// Get API key from environment
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Println("⚠️  ANTHROPIC_API_KEY not set - skipping example")
		fmt.Println("   Set ANTHROPIC_API_KEY environment variable to use this example")
		return
	}

	ctx := context.Background()

	// Create Claude provider with default config
	claudeConfig := claude.DefaultConfig(apiKey)
	claudeProvider := claude.New(claudeConfig)

	// Create agent with Claude provider
	ag := agent.New(
		agent.WithName("Claude Assistant"),
		agent.WithSystemPrompt("You are a helpful assistant powered by Anthropic's Claude."),
		agent.WithProvider(claudeProvider),
		agent.WithTemperature(0.7),
	)

	// Run the agent
	result, err := ag.Run(ctx, "Hello, what is 2 + 2?")
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Claude Response: %s\n", result)
}

func providerConfigurationExample() {
	fmt.Println("OpenAI Provider Configuration:")
	fmt.Println("  - Model: gpt-4o-mini (default)")
	fmt.Println("  - MaxTokens: 2000 (configurable)")
	fmt.Println("  - Temperature: 0.7 (configurable)")
	fmt.Println("  - Usage: openai.New(config)")

	fmt.Println("\nClaude Provider Configuration:")
	fmt.Println("  - Model: claude-3-5-sonnet-20241022 (default)")
	fmt.Println("  - MaxTokens: 4096 (configurable)")
	fmt.Println("  - Temperature: 0.7 (configurable)")
	fmt.Println("  - Usage: claude.New(config)")

	fmt.Println("\nTo use these providers:")
	fmt.Println("  1. Set OPENAI_API_KEY or ANTHROPIC_API_KEY environment variable")
	fmt.Println("  2. Create a config with DefaultConfig(apiKey)")
	fmt.Println("  3. Create a provider with New(config)")
	fmt.Println("  4. Pass provider to agent with agent.WithProvider(provider)")

	fmt.Println("\nExample:")
	fmt.Println(`
  openaiConfig := openai.DefaultConfig(apiKey)
  openaiConfig.Temperature = 0.5
  openaiConfig.MaxTokens = 1000
  provider := openai.New(openaiConfig)

  ag := agent.New(
    agent.WithProvider(provider),
    agent.WithSystemPrompt("You are helpful"),
  )

  result, err := ag.Run(ctx, "Your question here")
  `)
}
