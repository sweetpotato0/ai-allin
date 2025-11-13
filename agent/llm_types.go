package agent

import "github.com/sweetpotato0/ai-allin/message"

// GenerateRequest bundles inputs for a non-streaming LLM invocation.
type GenerateRequest struct {
	Messages []*message.Message
	Tools    []map[string]any
}

// GenerateResponse captures the LLM reply for non-streaming calls.
type GenerateResponse struct {
	Message *message.Message
}

// GenerateStreamRequest bundles inputs for streaming LLM invocations.
type GenerateStreamRequest struct {
	Messages []*message.Message
	Tools    []map[string]any
}

// StreamResponse returns both the accumulated assistant message and a token iterator.
// Consumers should drain Stream to receive incremental content; Message will contain
// the final accumulated result after iteration completes.
