package main

import (
	"context"
	"fmt"
	"log"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/message"
	"github.com/sweetpotato0/ai-allin/middleware"
	"github.com/sweetpotato0/ai-allin/middleware/enricher"
	"github.com/sweetpotato0/ai-allin/middleware/errorhandler"
	"github.com/sweetpotato0/ai-allin/middleware/limiter"
	"github.com/sweetpotato0/ai-allin/middleware/logger"
	"github.com/sweetpotato0/ai-allin/middleware/validator"
)

// MockLLMClient for demonstration
type MockLLMClient struct{}

func (m *MockLLMClient) Generate(ctx context.Context, messages []*message.Message, tools []map[string]interface{}) (*message.Message, error) {
	return message.NewMessage(message.RoleAssistant, "This is a mock response from the LLM"), nil
}

func main() {
	fmt.Println("=== Middleware Examples with Separate Packages ===\n")

	// Example 1: Logger middleware
	fmt.Println("Example 1: Logger Middleware")
	fmt.Println("-----------------------------")
	loggerExample()

	// Example 2: Input validation middleware
	fmt.Println("\nExample 2: Input Validation Middleware")
	fmt.Println("---------------------------------------")
	validationExample()

	// Example 3: Error handling middleware
	fmt.Println("\nExample 3: Error Handling Middleware")
	fmt.Println("-------------------------------------")
	errorHandlingExample()

	// Example 4: Context enrichment middleware
	fmt.Println("\nExample 4: Context Enrichment")
	fmt.Println("------------------------------")
	enrichmentExample()

	// Example 5: Rate limiting middleware
	fmt.Println("\nExample 5: Rate Limiting")
	fmt.Println("------------------------")
	rateLimitingExample()

	// Example 6: Combining multiple middlewares
	fmt.Println("\nExample 6: Combined Middlewares")
	fmt.Println("--------------------------------")
	combinedMiddlewaresExample()
}

func loggerExample() {
	llm := &MockLLMClient{}

	// Create logging middlewares
	requestLogger := logger.NewRequestLogger(func(msg string) {
		fmt.Println(msg)
	})

	responseLogger := logger.NewResponseLogger(func(msg string) {
		fmt.Println(msg)
	})

	// Create agent with logging middleware
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

func validationExample() {
	llm := &MockLLMClient{}

	// Create input validator
	inputValidator := validator.NewInputValidator(func(input string) error {
		if len(input) == 0 {
			return fmt.Errorf("input cannot be empty")
		}
		if len(input) > 1000 {
			return fmt.Errorf("input too long: maximum 1000 characters")
		}
		return nil
	})

	// Create response filter
	responseFilter := validator.NewResponseFilter(func(msg *message.Message) error {
		if len(msg.Content) > 500 {
			msg.Content = msg.Content[:500] + "... (truncated)"
		}
		return nil
	})

	// Create agent with validation
	ag := agent.New(
		agent.WithProvider(llm),
		agent.WithMiddleware(inputValidator),
		agent.WithMiddleware(responseFilter),
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
}

func errorHandlingExample() {
	llm := &MockLLMClient{}

	// Create error handler that logs and recovers
	errHandler := errorhandler.NewErrorHandler(func(err error) error {
		fmt.Printf("[ErrorHandler] Caught error: %v\n", err)
		return nil // Continue processing
	})

	// Create validator that will fail
	inputValidator := validator.NewInputValidator(func(input string) error {
		if input == "error" {
			return fmt.Errorf("intentional error for testing")
		}
		return nil
	})

	ag := agent.New(
		agent.WithProvider(llm),
		agent.WithMiddleware(errHandler),
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

func enrichmentExample() {
	llm := &MockLLMClient{}

	// Create context enricher
	ctxEnricher := enricher.NewContextEnricher(func(ctx *middleware.Context) error {
		// Add custom metadata
		ctx.Metadata["request_id"] = "req-12345"
		ctx.Metadata["user_id"] = "user-42"
		ctx.Metadata["timestamp"] = "2024-01-01T00:00:00Z"

		fmt.Printf("Context enriched with metadata: %v\n", ctx.Metadata)
		return nil
	})

	ag := agent.New(
		agent.WithProvider(llm),
		agent.WithMiddleware(ctxEnricher),
	)

	ctx := context.Background()
	result, err := ag.Run(ctx, "Hello with context!")
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Result: %s\n", result)
}

func rateLimitingExample() {
	llm := &MockLLMClient{}

	// Create rate limiter (max 2 requests)
	rateLimiter := limiter.NewRateLimiter(2)

	ag := agent.New(
		agent.WithProvider(llm),
		agent.WithMiddleware(rateLimiter),
	)

	ctx := context.Background()

	fmt.Println("Making requests with rate limit (max 2)...")

	for i := 1; i <= 3; i++ {
		fmt.Printf("Request %d: ", i)
		_, err := ag.Run(ctx, fmt.Sprintf("Request number %d", i))
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("Success\n")
		}
	}

	fmt.Printf("Total requests made: %d\n", rateLimiter.GetCounter())

	// Reset and try again
	fmt.Println("\nAfter reset...")
	rateLimiter.Reset()

	result, err := ag.Run(ctx, "New request after reset")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Success: %s\n", result)
	}
}

func combinedMiddlewaresExample() {
	llm := &MockLLMClient{}

	// Combine multiple middlewares for comprehensive processing
	requestLogger := logger.NewRequestLogger(func(msg string) {
		fmt.Println(msg)
	})

	inputValidator := validator.NewInputValidator(func(input string) error {
		if len(input) == 0 {
			return fmt.Errorf("input cannot be empty")
		}
		if len(input) > 500 {
			return fmt.Errorf("input too long")
		}
		return nil
	})

	responseLogger := logger.NewResponseLogger(func(msg string) {
		fmt.Println(msg)
	})

	ctxEnricher := enricher.NewContextEnricher(func(ctx *middleware.Context) error {
		ctx.Metadata["processed_at"] = "2024-01-01"
		return nil
	})

	// Create agent with all middlewares
	ag := agent.New(
		agent.WithProvider(llm),
		agent.WithMiddleware(requestLogger),
		agent.WithMiddleware(inputValidator),
		agent.WithMiddleware(ctxEnricher),
		agent.WithMiddleware(responseLogger),
	)

	ctx := context.Background()
	result, err := ag.Run(ctx, "Complex request with multiple middlewares")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Final result: %s\n", result)
}
