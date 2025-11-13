package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/contrib/provider/claude"
	"github.com/sweetpotato0/ai-allin/contrib/provider/openai"
	"github.com/sweetpotato0/ai-allin/message"
)

func main() {
	fmt.Println("=== LLM Streaming Examples ===")

	// Example 1: OpenAI Streaming
	fmt.Println("Example 1: OpenAI Provider with Streaming")
	fmt.Println("----------------------------------------")
	openaiStreamingExample()

	// Example 2: Claude Streaming
	fmt.Println("\nExample 2: Claude Provider with Streaming")
	fmt.Println("----------------------------------------")
	claudeStreamingExample()

	// Example 3: Fallback to non-streaming
	fmt.Println("\nExample 3: Mock Provider with Streaming (Fallback)")
	fmt.Println("--------------------------------------------------")
	mockStreamingExample()
}

func openaiStreamingExample() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	baseURL := os.Getenv("OPENAI_API_BASE_URL")
	if apiKey == "" {
		fmt.Println("⚠️  OPENAI_API_KEY not set - skipping example")
		fmt.Println("   Set OPENAI_API_KEY environment variable to use this example")
		return
	}

	ctx := context.Background()

	// Create OpenAI provider
	config := openai.DefaultConfig().WithAPIKey(apiKey).WithBaseURL(baseURL)
	config.Temperature = 0.7
	provider := openai.New(config)

	// Create agent with OpenAI provider
	ag := agent.New(
		agent.WithName("OpenAI Streaming Agent"),
		agent.WithSystemPrompt("You are a helpful assistant. Keep responses concise."),
		agent.WithProvider(provider),
	)

	// Create a streaming callback that prints tokens as they arrive
	tokenBuffer := strings.Builder{}
	callback := func(token string) error {
		tokenBuffer.WriteString(token)
		fmt.Print(token) // Print each token as it arrives
		return nil
	}

	fmt.Print("Agent: ")
	result, err := ag.RunStream(ctx, "Tell me a fun fact about Go programming language", callback)
	if err != nil {
		fmt.Printf("\nError: %v\n", err)
		return
	}

	fmt.Printf("\n\nFinal result: %s\n", result)
}

func claudeStreamingExample() {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Println("⚠️  ANTHROPIC_API_KEY not set - skipping example")
		fmt.Println("   Set ANTHROPIC_API_KEY environment variable to use this example")
		return
	}

	ctx := context.Background()

	// Create Claude provider
	config := claude.DefaultConfig().WithAPIKey(apiKey).WithBaseURL(os.Getenv("ANTHROPIC_BASE_URL"))
	config.Temperature = 0.7
	provider := claude.New(config)

	// Create agent with Claude provider
	ag := agent.New(
		agent.WithName("Claude Streaming Agent"),
		agent.WithSystemPrompt("You are a helpful assistant. Keep responses concise."),
		agent.WithProvider(provider),
	)

	// Create a streaming callback that counts tokens
	tokenCount := 0
	callback := func(token string) error {
		tokenCount++
		fmt.Print(token) // Print each token as it arrives
		return nil
	}

	fmt.Print("Agent: ")
	result, err := ag.RunStream(ctx, "Explain quantum computing in 2 sentences", callback)
	if err != nil {
		fmt.Printf("\nError: %v\n", err)
		return
	}

	fmt.Printf("\n\nTotal tokens received: %d\n", tokenCount)
	fmt.Printf("Final result: %s\n", result)
}

// MockStreamingClient demonstrates fallback behavior when streaming is not supported
type MockStreamingClient struct{}

func (m *MockStreamingClient) Generate(ctx context.Context, req *agent.GenerateRequest) (*agent.GenerateResponse, error) {
	msg := message.NewMessage(message.RoleAssistant, "This is a mock response without streaming support.")
	msg.Completed = true
	return &agent.GenerateResponse{Message: msg}, nil
}

func (m *MockStreamingClient) SetTemperature(float64) {}
func (m *MockStreamingClient) SetMaxTokens(int64)     {}
func (m *MockStreamingClient) SetModel(string)        {}

func mockStreamingExample() {
	// Create a mock provider that doesn't support streaming
	_ = &MockStreamingClient{}

	// Cast to agent.LLMClient for creating agent
	// Note: In real code, the mock would properly implement agent.LLMClient
	fmt.Println("This example demonstrates the fallback behavior when a provider")
	fmt.Println("doesn't support streaming. The framework will still work correctly,")
	fmt.Println("but without real-time token streaming.")
	fmt.Println()
	fmt.Println("Example streaming output (if provider supported it):")
	fmt.Println("Agent: This is a mock response... (tokens arriving one by one)")
}

// StreamingFeatures demonstrates various streaming capabilities
func streamingFeatures() {
	fmt.Println("\n=== Streaming Features ===")
	fmt.Println()
	fmt.Println("1. Real-time Token Streaming:")
	fmt.Println("   - Tokens are sent to callback function as they arrive")
	fmt.Println("   - Enables live UI updates")
	fmt.Println("   - Reduces perceived latency")
	fmt.Println()
	fmt.Println("2. Automatic Fallback:")
	fmt.Println("   - If provider doesn't support streaming, falls back to regular generation")
	fmt.Println("   - Ensures compatibility with all providers")
	fmt.Println()
	fmt.Println("3. Tool Call Support:")
	fmt.Println("   - Streaming includes tool call handling")
	fmt.Println("   - Tool calls are accumulated during streaming")
	fmt.Println()
	fmt.Println("4. Error Handling:")
	fmt.Println("   - Callback errors are properly handled")
	fmt.Println("   - Stream is closed on error to prevent resource leaks")
	fmt.Println()
	fmt.Println("5. Buffer Management:")
	fmt.Println("   - Configurable streaming options")
	fmt.Println("   - Token buffering for performance")
	fmt.Println()
}
