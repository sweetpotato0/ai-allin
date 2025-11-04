package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/message"
	"github.com/sweetpotato0/ai-allin/middleware"
)

// MockLLMClient for demonstration
type MockLLMClient struct{}

func (m *MockLLMClient) Generate(ctx context.Context, messages []*message.Message, tools []map[string]interface{}) (*message.Message, error) {
	return message.NewMessage(message.RoleAssistant, "This is a mock response from the LLM"), nil
}

func main() {
	fmt.Println("=== Middleware Examples ===\n")

	// Example 1: Basic middleware usage
	fmt.Println("Example 1: Basic Middleware Chain")
	fmt.Println("----------------------------------")
	basicMiddlewareExample()

	// Example 2: Input validation middleware
	fmt.Println("\nExample 2: Input Validation Middleware")
	fmt.Println("---------------------------------------")
	inputValidationExample()

	// Example 3: Request/Response logging
	fmt.Println("\nExample 3: Request/Response Logging")
	fmt.Println("------------------------------------")
	loggingExample()

	// Example 4: Error handling middleware
	fmt.Println("\nExample 4: Error Handling Middleware")
	fmt.Println("-------------------------------------")
	errorHandlingExample()

	// Example 5: Context enrichment
	fmt.Println("\nExample 5: Context Enrichment")
	fmt.Println("-------------------------------")
	contextEnrichmentExample()
}

func basicMiddlewareExample() {
	llm := &MockLLMClient{}

	// Create request logger middleware
	requestLogger := middleware.NewRequestLogger(func(msg string) {
		fmt.Println(msg)
	})

	// Create response logger middleware
	responseLogger := middleware.NewResponseLogger(func(msg string) {
		fmt.Println(msg)
	})

	// Create agent with middlewares
	ag := agent.New(
		agent.WithProvider(llm),
		agent.WithSystemPrompt("You are a helpful assistant"),
		agent.WithMiddleware(requestLogger),
		agent.WithMiddleware(responseLogger),
	)

	ctx := context.Background()
	result, err := ag.Run(ctx, "Hello, how are you?")
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Final response: %s\n", result)
}

func inputValidationExample() {
	llm := &MockLLMClient{}

	// Create input validator
	inputValidator := middleware.NewInputValidator(func(input string) error {
		if len(input) == 0 {
			return middleware.ErrInvalidInput
		}
		if len(input) > 1000 {
			return fmt.Errorf("input too long: maximum 1000 characters")
		}
		return nil
	})

	// Create agent with input validation
	ag := agent.New(
		agent.WithProvider(llm),
		agent.WithMiddleware(inputValidator),
	)

	ctx := context.Background()

	// Valid input
	fmt.Println("Testing valid input...")
	result, err := ag.Run(ctx, "Hello!")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Success: %s\n", result)
	}

	// Invalid input (empty)
	fmt.Println("\nTesting invalid input (empty)...")
	result, err = ag.Run(ctx, "")
	if err != nil {
		fmt.Printf("Validation error caught: %v\n", err)
	}

	// Invalid input (too long)
	fmt.Println("\nTesting invalid input (too long)...")
	longInput := strings.Repeat("x", 1001)
	result, err = ag.Run(ctx, longInput)
	if err != nil {
		fmt.Printf("Validation error caught: %v\n", err)
	}
}

func loggingExample() {
	llm := &MockLLMClient{}

	// Create comprehensive logging
	requestLog := make([]string, 0)
	responseLog := make([]string, 0)

	requestLogger := middleware.NewRequestLogger(func(msg string) {
		requestLog = append(requestLog, msg)
		fmt.Println(msg)
	})

	responseLogger := middleware.NewResponseLogger(func(msg string) {
		responseLog = append(responseLog, msg)
		fmt.Println(msg)
	})

	ag := agent.New(
		agent.WithProvider(llm),
		agent.WithMiddleware(requestLogger),
		agent.WithMiddleware(responseLogger),
	)

	ctx := context.Background()
	result, err := ag.Run(ctx, "What is 2+2?")
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\nRequest logs: %v\n", len(requestLog))
	fmt.Printf("Response logs: %v\n", len(responseLog))
	fmt.Printf("Result: %s\n", result)
}

func errorHandlingExample() {
	llm := &MockLLMClient{}

	// Create error handler
	errorHandler := middleware.NewErrorHandler(func(err error) error {
		fmt.Printf("[ErrorHandler] Caught error: %v\n", err)
		// Could transform or recover from error here
		return nil // Continue processing
	})

	// Create input validator that will fail
	inputValidator := middleware.NewInputValidator(func(input string) error {
		if input == "error" {
			return fmt.Errorf("test error")
		}
		return nil
	})

	ag := agent.New(
		agent.WithProvider(llm),
		agent.WithMiddleware(errorHandler),
		agent.WithMiddleware(inputValidator),
	)

	ctx := context.Background()

	fmt.Println("Testing error recovery...")
	result, err := ag.Run(ctx, "error")
	if err != nil {
		fmt.Printf("Error handled: %v\n", err)
	} else {
		fmt.Printf("Result: %s\n", result)
	}
}

func contextEnrichmentExample() {
	llm := &MockLLMClient{}

	// Create context enricher
	enricher := middleware.NewContextEnricher(func(ctx *middleware.Context) error {
		// Add custom metadata
		ctx.Metadata["request_id"] = "req-12345"
		ctx.Metadata["timestamp"] = "2024-01-01T00:00:00Z"
		ctx.Metadata["user_id"] = "user-42"

		fmt.Printf("Context enriched with metadata: %v\n", ctx.Metadata)
		return nil
	})

	ag := agent.New(
		agent.WithProvider(llm),
		agent.WithMiddleware(enricher),
	)

	ctx := context.Background()
	result, err := ag.Run(ctx, "Hello with context!")
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Result: %s\n", result)
}
