package main

import (
	"context"
	"fmt"
	"os"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/contrib/provider/claude"
	"github.com/sweetpotato0/ai-allin/contrib/provider/cohere"
	"github.com/sweetpotato0/ai-allin/contrib/provider/gemini"
	"github.com/sweetpotato0/ai-allin/contrib/provider/groq"
	"github.com/sweetpotato0/ai-allin/contrib/provider/openai"
)

func main() {
	fmt.Println("=== Multiple LLM Providers Example ===")

	ctx := context.Background()

	// Example 1: OpenAI Provider
	fmt.Println("Example 1: OpenAI Provider")
	fmt.Println("---------------------------")
	openaiExample(ctx)

	// Example 2: Claude Provider
	fmt.Println("\nExample 2: Claude Provider")
	fmt.Println("---------------------------")
	claudeExample(ctx)

	// Example 3: Groq Provider
	fmt.Println("\nExample 3: Groq Provider")
	fmt.Println("------------------------")
	groqExample(ctx)

	// Example 4: Cohere Provider
	fmt.Println("\nExample 4: Cohere Provider")
	fmt.Println("---------------------------")
	cohereExample(ctx)

	// Example 5: Gemini Provider
	fmt.Println("\nExample 5: Gemini Provider")
	fmt.Println("---------------------------")
	geminiExample(ctx)

	// Example 6: Provider Switching
	fmt.Println("\nExample 6: Dynamic Provider Switching")
	fmt.Println("-------------------------------------")
	providerSwitchingExample(ctx)
}

func openaiExample(ctx context.Context) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("⚠️  OPENAI_API_KEY not set - skipping")
		return
	}

	config := openai.DefaultConfig()
	provider := openai.New(config)

	ag := agent.New(
		agent.WithName("OpenAI Agent"),
		agent.WithProvider(provider),
		agent.WithSystemPrompt("You are a helpful assistant."),
	)

	result, err := ag.Run(ctx, "What is 2+2?")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Response: %s\n", result.Text())
}

func claudeExample(ctx context.Context) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Println("⚠️  ANTHROPIC_API_KEY not set - skipping")
		return
	}

	config := claude.DefaultConfig().WithAPIKey(apiKey).WithBaseURL(os.Getenv("ANTHROPIC_BASE_URL"))
	provider := claude.New(config)

	ag := agent.New(
		agent.WithName("Claude Agent"),
		agent.WithProvider(provider),
		agent.WithSystemPrompt("You are a helpful assistant."),
	)

	result, err := ag.Run(ctx, "What is 2+2?")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Response: %s\n", result.Text())
}

func groqExample(ctx context.Context) {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		fmt.Println("⚠️  GROQ_API_KEY not set - skipping")
		return
	}

	config := groq.DefaultConfig(apiKey)
	provider := groq.New(config)

	ag := agent.New(
		agent.WithName("Groq Agent"),
		agent.WithProvider(provider),
		agent.WithSystemPrompt("You are a helpful assistant."),
	)

	result, err := ag.Run(ctx, "What is 2+2?")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Response: %s\n", result.Text())
}

func cohereExample(ctx context.Context) {
	apiKey := os.Getenv("COHERE_API_KEY")
	if apiKey == "" {
		fmt.Println("⚠️  COHERE_API_KEY not set - skipping")
		return
	}

	config := cohere.DefaultConfig(apiKey)
	provider := cohere.New(config)

	ag := agent.New(
		agent.WithName("Cohere Agent"),
		agent.WithProvider(provider),
		agent.WithSystemPrompt("You are a helpful assistant."),
	)

	result, err := ag.Run(ctx, "What is 2+2?")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Response: %s\n", result.Text())
}

func geminiExample(ctx context.Context) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Println("⚠️  GEMINI_API_KEY not set - skipping")
		return
	}

	config := gemini.DefaultConfig(apiKey)
	provider := gemini.New(config)

	ag := agent.New(
		agent.WithName("Gemini Agent"),
		agent.WithProvider(provider),
		agent.WithSystemPrompt("You are a helpful assistant."),
	)

	result, err := ag.Run(ctx, "What is 2+2?")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Response: %s\n", result.Text())
}

func providerSwitchingExample(ctx context.Context) {
	fmt.Println("This example demonstrates switching between providers:")
	fmt.Println("1. Create an agent with one provider")
	fmt.Println("2. Run a query")
	fmt.Println("3. Update provider configuration")
	fmt.Println("4. Run another query with updated settings")
	fmt.Println("\nFor example:")
	fmt.Println("  provider := openai.New(config)")
	fmt.Println("  ag := agent.New(agent.WithProvider(provider))")
	fmt.Println("  result1, _ := ag.Run(ctx, query)")
	fmt.Println("  provider.SetTemperature(0.9)")
	fmt.Println("  result2, _ := ag.Run(ctx, query)")
	fmt.Println("\nAll providers support SetTemperature, SetMaxTokens, and SetModel methods.")
}
