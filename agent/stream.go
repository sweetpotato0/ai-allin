package agent

import (
	"context"
	"fmt"

	"iter"

	"github.com/sweetpotato0/ai-allin/memory"
	"github.com/sweetpotato0/ai-allin/message"
)

// StreamCallback is called for each message received from the LLM during streaming
type StreamCallback func(*message.Message) error

// StreamLLMClient defines the interface for LLM providers that support streaming
type StreamLLMClient interface {
	LLMClient
	// GenerateStream generates a response with token streaming
	GenerateStream(ctx context.Context, req *GenerateRequest) iter.Seq2[*GenerateResponse, error]
}

// RunStream executes the agent with streaming output
// It calls the callback function for each token received from the LLM
func (a *Agent) RunStream(ctx context.Context, input string, callback StreamCallback) iter.Seq2[*message.Message, error] {
	return func(yield func(*message.Message, error) bool) {
		if err := a.ensureToolProviders(ctx); err != nil {
			yield(nil, err)
			return
		}

		// Check if LLM client supports streaming
		streamProvider, ok := a.llm.(StreamLLMClient)
		if !ok {
			// Fallback to regular Run if streaming not supported
			result, err := a.Run(ctx, input)
			if err != nil {
				yield(nil, err)
				return
			}
			// Still call the callback with the complete result
			if err := callback(result); err != nil {
				yield(nil, err)
				return
			}
			yield(result, nil)
			return
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
		// Get tool schemas if enabled
		var toolSchemas []map[string]any
		if a.enableTools {
			toolSchemas = a.tools.ToJSONSchemas()
		}

		// Call LLM with streaming
		streamSeq := streamProvider.GenerateStream(ctx, &GenerateRequest{
			Messages: a.ctx.GetMessages(),
			Tools:    toolSchemas,
		})
		if streamSeq == nil {
			yield(nil, fmt.Errorf("LLM streaming returned empty sequence"))
			return
		}

		var (
			streamErr error
			finalResp *message.Message
		)

		for resp, err := range streamSeq {
			if err != nil {
				streamErr = err
				break
			}
			if resp == nil {
				continue
			}

			if callback != nil && !resp.Message.Completed {
				if err := callback(resp.Message); err != nil {
					streamErr = err
					break
				}
			}

			if streamErr != nil {
				yield(nil, streamErr)
				return
			}

			if resp.Message.Completed {
				finalResp = resp.Message
			} else {
				if !yield(resp.Message, nil) {
					return
				}
			}
		}

		if finalResp == nil {
			yield(nil, fmt.Errorf("LLM streaming ended without final response"))
			return
		}

		a.AddMessage(finalResp)

		// Check if there are tool calls
		if len(finalResp.ToolCalls) == 0 {
			// No tool calls, return the response
			if a.enableMemory && a.memory != nil {
				// Store conversation in memory
				conversationContent := fmt.Sprintf("User: %s\nAssistant: %s", input, finalResp.Text())
				mem := &memory.Memory{
					ID:       memory.GenerateMemoryID(),
					Content:  conversationContent,
					Metadata: map[string]any{"input": input, "response": finalResp.Text()},
				}
				a.memory.AddMemory(ctx, mem)
			}

			yield(finalResp, nil)
			return
		}

		// Execute tool calls
		for _, toolCall := range finalResp.ToolCalls {
			result, err := a.tools.Execute(ctx, toolCall.Name, toolCall.Args)
			if err != nil {
				result = fmt.Sprintf("Error executing tool %s: %v", toolCall.Name, err)
			}

			// Add tool response
			toolMsg := message.NewToolResponseMessage(toolCall.ID, result)
			a.AddMessage(toolMsg)
		}

		yield(finalResp, nil)
	}
}

// StreamingOptions holds configuration for streaming operations
type StreamingOptions struct {
	BufferSize      int   // Size of token buffer before calling callback
	Timeout         int64 // Timeout in milliseconds for streaming
	ErrorOnStopSeq  bool  // Whether to error on stop sequence
	PreserveNewline bool  // Preserve newline characters in tokens
}

// DefaultStreamingOptions returns default streaming options
func DefaultStreamingOptions() *StreamingOptions {
	return &StreamingOptions{
		BufferSize:      1,
		Timeout:         30000, // 30 seconds
		ErrorOnStopSeq:  false,
		PreserveNewline: true,
	}
}
