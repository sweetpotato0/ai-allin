package runtime

import (
	"context"
	"testing"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/message"
)

func TestAgentExecutorExecute(t *testing.T) {
	exec := NewAgentExecutor(newTestAgent())

	req := &Request{
		SessionID: "sess-1",
		Input:     "ping",
		History: []*message.Message{
			message.NewMessage(message.RoleSystem, "system override"),
			message.NewMessage(message.RoleUser, "previous"),
		},
	}

	result, err := exec.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("executor failed: %v", err)
	}

	if result.SessionID != req.SessionID {
		t.Fatalf("expected session id %s, got %s", req.SessionID, result.SessionID)
	}

	if result.Output == "" {
		t.Fatalf("expected non-empty output")
	}

	if len(result.Messages) == 0 {
		t.Fatalf("expected messages to be captured")
	}
}

func TestAgentExecutorNilRequest(t *testing.T) {
	exec := NewAgentExecutor(newTestAgent())
	if _, err := exec.Execute(context.Background(), nil); err == nil {
		t.Fatalf("expected error for nil request")
	}
}

func TestAgentExecutorEmptyInput(t *testing.T) {
	exec := NewAgentExecutor(newTestAgent())
	_, err := exec.Execute(context.Background(), &Request{Input: ""})
	if err == nil {
		t.Fatalf("expected error for empty input")
	}
}

func newTestAgent() *agent.Agent {
	llm := &mockLLM{}
	return agent.New(
		agent.WithSystemPrompt("You are a test agent."),
		agent.WithProvider(llm),
		agent.WithMaxIterations(1),
	)
}

type mockLLM struct{}

func (m *mockLLM) Generate(ctx context.Context, msgs []*message.Message, tools []map[string]any) (*message.Message, error) {
	last := ""
	if len(msgs) > 0 {
		last = msgs[len(msgs)-1].Content
	}
	return message.NewMessage(message.RoleAssistant, "echo:"+last), nil
}

func (m *mockLLM) SetTemperature(float64) {}
func (m *mockLLM) SetMaxTokens(int64)     {}
func (m *mockLLM) SetModel(string)        {}
