package main

import (
	"context"
	"fmt"
	"log"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/graph"
	"github.com/sweetpotato0/ai-allin/message"
	"github.com/sweetpotato0/ai-allin/runner"
	"github.com/sweetpotato0/ai-allin/session"
	"github.com/sweetpotato0/ai-allin/session/store"
	"github.com/sweetpotato0/ai-allin/tool"
)

// MockLLMClient is a mock LLM client for demonstration
type MockLLMClient struct{}

func (m *MockLLMClient) Generate(ctx context.Context, messages []*message.Message, tools []map[string]any) (*message.Message, error) {
	// Mock response
	return message.NewMessage(message.RoleAssistant, "This is a mock response from the LLM"), nil
}

func (m *MockLLMClient) SetTemperature(temp float64) {
	// Mock implementation - does nothing
}

func (m *MockLLMClient) SetMaxTokens(max int64) {
	// Mock implementation - does nothing
}

func (m *MockLLMClient) SetModel(model string) {
	// Mock implementation - does nothing
}

func main() {
	fmt.Println("=== AI Agent Framework Examples ===")

	// Example 1: Basic Agent Usage
	fmt.Println("\nExample 1: Basic Agent")
	basicAgentExample()

	// Example 2: Agent with Tools
	fmt.Println("\nExample 2: Agent with Tools")
	agentWithToolsExample()

	// Example 3: Graph Workflow
	fmt.Println("\nExample 3: Graph Workflow")
	graphWorkflowExample()

	// Example 4: Session Management
	fmt.Println("\nExample 4: Session Management")
	sessionManagementExample()

	// Example 5: Parallel Execution
	fmt.Println("\nExample 5: Parallel Execution")
	parallelExecutionExample()
}

func basicAgentExample() {
	ctx := context.Background()
	llm := &MockLLMClient{}

	// Create agent using Options pattern
	ag := agent.New(
		agent.WithName("Assistant"),
		agent.WithSystemPrompt("You are a helpful assistant."),
		agent.WithProvider(llm),
	)

	result, err := ag.Run(ctx, "Hello, how are you?")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Agent response: %s\n", result)
}

func agentWithToolsExample() {
	llm := &MockLLMClient{}

	// Create agent using Options pattern
	ag := agent.New(
		agent.WithProvider(llm),
		agent.WithTools(true),
	)

	// Register a calculator tool
	calculatorTool := &tool.Tool{
		Name:        "calculator",
		Description: "Performs basic arithmetic operations",
		Parameters: []tool.Parameter{
			{Name: "operation", Type: "string", Description: "Operation to perform (add, subtract, multiply, divide)", Required: true},
			{Name: "a", Type: "number", Description: "First number", Required: true},
			{Name: "b", Type: "number", Description: "Second number", Required: true},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			op := args["operation"].(string)
			a := args["a"].(float64)
			b := args["b"].(float64)

			var result float64
			switch op {
			case "add":
				result = a + b
			case "subtract":
				result = a - b
			case "multiply":
				result = a * b
			case "divide":
				if b == 0 {
					return "", fmt.Errorf("division by zero")
				}
				result = a / b
			default:
				return "", fmt.Errorf("unknown operation: %s", op)
			}

			return fmt.Sprintf("%.2f", result), nil
		},
	}

	if err := ag.RegisterTool(calculatorTool); err != nil {
		log.Printf("Error registering tool: %v", err)
		return
	}

	fmt.Println("Registered calculator tool successfully")
}

func graphWorkflowExample() {
	ctx := context.Background()

	// Build a simple workflow graph
	g := graph.NewBuilder().
		AddNode("start", graph.NodeTypeStart, func(ctx context.Context, state graph.State) (graph.State, error) {
			fmt.Println("Starting workflow...")
			state["step"] = 1
			return state, nil
		}).
		AddNode("process", graph.NodeTypeCustom, func(ctx context.Context, state graph.State) (graph.State, error) {
			fmt.Printf("Processing at step %v...\n", state["step"])
			state["step"] = state["step"].(int) + 1
			state["result"] = "Processed"
			return state, nil
		}).
		AddNode("end", graph.NodeTypeEnd, func(ctx context.Context, state graph.State) (graph.State, error) {
			fmt.Printf("Workflow completed with result: %s\n", state["result"])
			return state, nil
		}).
		AddEdge("start", "process").
		AddEdge("process", "end").
		SetStart("start").
		Build()

	initialState := make(graph.State)
	finalState, err := g.Execute(ctx, initialState)
	if err != nil {
		log.Printf("Error executing graph: %v", err)
		return
	}

	fmt.Printf("Final state: %v\n", finalState)
}

func sessionManagementExample() {
	ctx := context.Background()
	llm := &MockLLMClient{}

	// Create session manager with in-memory store
	mgr := session.NewManager(session.WithStore(store.NewInMemoryStore()))

	// Create multiple sessions
	for i := 1; i <= 3; i++ {
		sessionID := fmt.Sprintf("session-%d", i)
		ag := agent.New(agent.WithProvider(llm))

		sess, err := mgr.Create(ctx, sessionID, ag)
		if err != nil {
			log.Printf("Error creating session: %v", err)
			continue
		}

		// Run a query in each session
		result, err := sess.Run(ctx, fmt.Sprintf("Hello from session %d", i))
		if err != nil {
			log.Printf("Error running session: %v", err)
			continue
		}

		fmt.Printf("Session %s: %s\n", sessionID, result)

		// Persist the latest session snapshot (messages, last message, duration)
		if err := mgr.Save(ctx, sess); err != nil {
			log.Printf("Error saving session snapshot: %v", err)
		} else {
			fmt.Printf("Snapshot stored for %s (last duration: %s)\n", sessionID, sess.Snapshot().LastDuration)
		}
	}

	// List all sessions
	sessions, _ := mgr.List(ctx)
	fmt.Printf("Active sessions: %v\n", sessions)
	count, _ := mgr.Count(ctx)
	fmt.Printf("Session count: %d\n", count)
}

func parallelExecutionExample() {
	ctx := context.Background()
	llm := &MockLLMClient{}

	// Create multiple agents
	tasks := []*runner.Task{
		{
			ID:    "task-1",
			Agent: agent.New(agent.WithProvider(llm)),
			Input: "Analyze sentiment",
		},
		{
			ID:    "task-2",
			Agent: agent.New(agent.WithProvider(llm)),
			Input: "Summarize text",
		},
		{
			ID:    "task-3",
			Agent: agent.New(agent.WithProvider(llm)),
			Input: "Extract keywords",
		},
	}

	// Execute in parallel
	pr := runner.NewParallelRunner(5)
	results := pr.RunParallel(ctx, tasks)

	fmt.Println("Parallel execution results:")
	for _, result := range results {
		if result.Error != nil {
			fmt.Printf("Task %s failed: %v\n", result.TaskID, result.Error)
		} else {
			fmt.Printf("Task %s: %s\n", result.TaskID, result.Output)
		}
	}
}
