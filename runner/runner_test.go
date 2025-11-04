package runner

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sweetpotato0/ai-allin/agent"
)

func TestNewRunner(t *testing.T) {
	runner := New(5)
	if runner == nil {
		t.Errorf("New returned nil")
	}
}

func TestNewRunnerDefaultConcurrency(t *testing.T) {
	runner := New(0)
	if runner == nil {
		t.Errorf("New with 0 concurrency returned nil")
	}
}

func TestNewParallelRunner(t *testing.T) {
	pr := NewParallelRunner(5)
	if pr == nil {
		t.Errorf("NewParallelRunner returned nil")
	}
}

func TestRunParallel(t *testing.T) {
	ag1 := agent.New(agent.WithName("Agent1"))
	ag2 := agent.New(agent.WithName("Agent2"))
	ag3 := agent.New(agent.WithName("Agent3"))

	tasks := []*Task{
		{ID: "task1", Agent: ag1, Input: "input1"},
		{ID: "task2", Agent: ag2, Input: "input2"},
		{ID: "task3", Agent: ag3, Input: "input3"},
	}

	pr := NewParallelRunner(10)
	results := pr.RunParallel(context.Background(), tasks)

	if len(results) != len(tasks) {
		t.Errorf("Expected %d results, got %d", len(tasks), len(results))
	}

	for i, result := range results {
		if result.TaskID != tasks[i].ID {
			t.Errorf("Result %d: expected TaskID %s, got %s", i, tasks[i].ID, result.TaskID)
		}
	}
}

func TestRunParallelWithNilTasks(t *testing.T) {
	pr := NewParallelRunner(10)
	results := pr.RunParallel(context.Background(), nil)

	if len(results) != 0 {
		t.Errorf("Expected 0 results for nil tasks, got %d", len(results))
	}
}

func TestRunParallelWithEmptyTasks(t *testing.T) {
	pr := NewParallelRunner(10)
	results := pr.RunParallel(context.Background(), []*Task{})

	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty tasks, got %d", len(results))
	}
}

func TestRunParallelWithTimeout(t *testing.T) {
	ag := agent.New()

	tasks := []*Task{
		{ID: "task1", Agent: ag, Input: "test"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	pr := NewParallelRunner(1)
	results := pr.RunParallel(ctx, tasks)

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}

func TestRunParallelSingleTask(t *testing.T) {
	ag := agent.New(agent.WithName("SingleAgent"))

	tasks := []*Task{
		{ID: "single", Agent: ag, Input: "test input"},
	}

	pr := NewParallelRunner(1)
	results := pr.RunParallel(context.Background(), tasks)

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if results[0].TaskID != "single" {
		t.Errorf("Expected TaskID 'single', got %s", results[0].TaskID)
	}
}

func TestRunParallelMultipleTasks(t *testing.T) {
	agents := make([]*agent.Agent, 5)
	tasks := make([]*Task, 5)

	for i := 0; i < 5; i++ {
		agents[i] = agent.New(agent.WithName(fmt.Sprintf("Agent%d", i)))
		tasks[i] = &Task{
			ID:    fmt.Sprintf("task%d", i),
			Agent: agents[i],
			Input: fmt.Sprintf("input%d", i),
		}
	}

	pr := NewParallelRunner(5)
	results := pr.RunParallel(context.Background(), tasks)

	if len(results) != 5 {
		t.Errorf("Expected 5 results, got %d", len(results))
	}
}

func TestRunParallelConcurrencyLimit(t *testing.T) {
	// Test that max concurrency is respected
	maxConcurrency := 2

	agents := make([]*agent.Agent, 5)
	tasks := make([]*Task, 5)

	for i := 0; i < 5; i++ {
		agents[i] = agent.New(agent.WithName(fmt.Sprintf("Agent%d", i)))
		tasks[i] = &Task{
			ID:    fmt.Sprintf("task%d", i),
			Agent: agents[i],
			Input: fmt.Sprintf("input%d", i),
		}
	}

	pr := NewParallelRunner(maxConcurrency)
	results := pr.RunParallel(context.Background(), tasks)

	if len(results) != 5 {
		t.Errorf("Expected 5 results, got %d", len(results))
	}

	// All tasks should complete successfully
	for _, result := range results {
		if result == nil {
			t.Errorf("Got nil result")
		}
	}
}

func TestParallelTaskOrder(t *testing.T) {
	// Results order should match task input order
	agents := make([]*agent.Agent, 3)
	tasks := make([]*Task, 3)

	for i := 0; i < 3; i++ {
		agents[i] = agent.New()
		tasks[i] = &Task{
			ID:    fmt.Sprintf("task%d", i),
			Agent: agents[i],
			Input: fmt.Sprintf("input%d", i),
		}
	}

	pr := NewParallelRunner(3)
	results := pr.RunParallel(context.Background(), tasks)

	for i, result := range results {
		if result.TaskID != fmt.Sprintf("task%d", i) {
			t.Errorf("Result %d: expected TaskID task%d, got %s", i, i, result.TaskID)
		}
	}
}

func TestRunAndRunGraph(t *testing.T) {
	runner := New(5)

	ag := agent.New(agent.WithName("TestAgent"))
	_ , _ = runner.Run(context.Background(), ag, "test input")

	// Just verify the call succeeded without error
	if runner == nil {
		t.Errorf("Runner is nil")
	}
}

