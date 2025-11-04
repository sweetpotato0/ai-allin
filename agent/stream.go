package agent

import (
	"context"
	"fmt"

	"github.com/sweetpotato0/ai-allin/memory"
	"github.com/sweetpotato0/ai-allin/message"
)

// StreamCallback is called for each token received from the LLM during streaming
type StreamCallback func(token string) error

// StreamLLMClient defines the interface for LLM providers that support streaming
type StreamLLMClient interface {
	LLMClient
	// GenerateStream generates a response with token streaming
	GenerateStream(ctx context.Context, messages []*message.Message, tools []map[string]interface{}, callback StreamCallback) (*message.Message, error)
}

// RunStream executes the agent with streaming output
// It calls the callback function for each token received from the LLM
func (a *Agent) RunStream(ctx context.Context, input string, callback StreamCallback) (string, error) {
	// Check if LLM client supports streaming
	streamProvider, ok := a.llm.(StreamLLMClient)
	if !ok {
		// Fallback to regular Run if streaming not supported
		result, err := a.Run(ctx, input)
		if err != nil {
			return "", err
		}
		// Still call the callback with the complete result
		if err := callback(result); err != nil {
			return "", err
		}
		return result, nil
	}

	// Add user message
	userMsg := message.NewMessage(message.RoleUser, input)
	a.AddMessage(userMsg)

	// Search relevant memories if enabled
	if a.enableMemory && a.memory != nil {
		memories, err := a.memory.SearchMemory(ctx, input)
		if err == nil && len(memories) > 0 {
			// Add memories as context (simplified)
			memoryContext := "Relevant memories:\n"
			for _, mem := range memories {
				memoryContext += fmt.Sprintf("- %v\n", mem)
			}
			contextMsg := message.NewMessage(message.RoleSystem, memoryContext)
			a.ctx.AddMessage(contextMsg)
		}
	}

	// Execution loop with tool calls
	for i := 0; i < a.maxIterations; i++ {
		// Get tool schemas if enabled
		var toolSchemas []map[string]interface{}
		if a.enableTools {
			toolSchemas = a.tools.ToJSONSchemas()
		}

		// Call LLM with streaming
		response, err := streamProvider.GenerateStream(ctx, a.ctx.GetMessages(), toolSchemas, callback)
		if err != nil {
			return "", fmt.Errorf("LLM generation failed: %w", err)
		}

		a.AddMessage(response)

		// Check if there are tool calls
		if len(response.ToolCalls) == 0 {
			// No tool calls, return the response
			if a.enableMemory && a.memory != nil {
				// Store conversation in memory
				mem := &memory.Memory{}
				a.memory.AddMemory(ctx, mem)
			}
			return response.Content, nil
		}

		// Execute tool calls
		for _, toolCall := range response.ToolCalls {
			result, err := a.tools.Execute(ctx, toolCall.Name, toolCall.Args)
			if err != nil {
				result = fmt.Sprintf("Error executing tool %s: %v", toolCall.Name, err)
			}

			// Add tool response
			toolMsg := message.NewToolResponseMessage(toolCall.ID, result)
			a.AddMessage(toolMsg)
		}

		// Continue loop to get final response
	}

	return "", fmt.Errorf("max iterations (%d) reached", a.maxIterations)
}

// StreamingOptions holds configuration for streaming operations
type StreamingOptions struct {
	BufferSize      int           // Size of token buffer before calling callback
	Timeout         int64         // Timeout in milliseconds for streaming
	ErrorOnStopSeq  bool          // Whether to error on stop sequence
	PreserveNewline bool          // Preserve newline characters in tokens
}

// DefaultStreamingOptions returns default streaming options
func DefaultStreamingOptions() *StreamingOptions {
	return &StreamingOptions{
		BufferSize:      1,
		Timeout:         30000,  // 30 seconds
		ErrorOnStopSeq:  false,
		PreserveNewline: true,
	}
}
