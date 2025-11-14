package runner

import (
	"context"
	"fmt"
	"sync"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/graph"
)

// Runner executes agents and workflows
type Runner interface {
	// Run executes an agent with the given input
	Run(ctx context.Context, ag *agent.Agent, input string) (string, error)

	// RunGraph executes a graph workflow
	RunGraph(ctx context.Context, g *graph.Graph, initialState graph.State) (graph.State, error)
}

// runner is the default implementation of Runner
type runner struct {
	maxConcurrency int
	semaphore      chan struct{}
}

// New creates a new runner
func New(maxConcurrency int) Runner {
	if maxConcurrency <= 0 {
		maxConcurrency = 10 // Default concurrency
	}
	return &runner{
		maxConcurrency: maxConcurrency,
		semaphore:      make(chan struct{}, maxConcurrency),
	}
}

// Run executes an agent with the given input
func (r *runner) Run(ctx context.Context, ag *agent.Agent, input string) (string, error) {
	// Acquire semaphore
	select {
	case r.semaphore <- struct{}{}:
		defer func() { <-r.semaphore }()
	case <-ctx.Done():
		return "", ctx.Err()
	}

	msg, err := ag.Run(ctx, input)
	if err != nil {
		return "", err
	}

	return msg.Text(), nil
}

// RunGraph executes a graph workflow
func (r *runner) RunGraph(ctx context.Context, g *graph.Graph, initialState graph.State) (graph.State, error) {
	// Acquire semaphore
	select {
	case r.semaphore <- struct{}{}:
		defer func() { <-r.semaphore }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return g.Execute(ctx, initialState)
}

// ParallelRunner executes multiple agents in parallel
type ParallelRunner struct {
	runner Runner
}

// NewParallelRunner creates a new parallel runner
func NewParallelRunner(maxConcurrency int) *ParallelRunner {
	return &ParallelRunner{
		runner: New(maxConcurrency),
	}
}

// Task represents a task to be executed
type Task struct {
	ID    string
	Agent *agent.Agent
	Input string
}

// Result represents the result of a task execution
type Result struct {
	TaskID string
	Output string
	Error  error
}

// RunParallel executes multiple tasks in parallel
func (pr *ParallelRunner) RunParallel(ctx context.Context, tasks []*Task) []*Result {
	results := make([]*Result, len(tasks))
	var wg sync.WaitGroup

	for i, task := range tasks {
		wg.Add(1)
		go func(index int, t *Task) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					results[index] = &Result{
						TaskID: t.ID,
						Output: "",
						Error:  fmt.Errorf("panic in task %s: %v", t.ID, r),
					}
				}
			}()

			output, err := pr.runner.Run(ctx, t.Agent, t.Input)
			results[index] = &Result{
				TaskID: t.ID,
				Output: output,
				Error:  err,
			}
		}(i, task)
	}

	wg.Wait()
	return results
}

// SequentialRunner executes agents sequentially
type SequentialRunner struct {
	runner Runner
}

// NewSequentialRunner creates a new sequential runner
func NewSequentialRunner() *SequentialRunner {
	return &SequentialRunner{
		runner: New(1), // Single concurrency for sequential execution
	}
}

// RunSequential executes tasks sequentially, passing output to the next task
func (sr *SequentialRunner) RunSequential(ctx context.Context, tasks []*Task) (*Result, error) {
	var lastOutput string

	for _, task := range tasks {
		// Use previous output as input for current task (if not the first task)
		input := task.Input
		if lastOutput != "" {
			input = lastOutput
		}

		output, err := sr.runner.Run(ctx, task.Agent, input)
		if err != nil {
			return &Result{
				TaskID: task.ID,
				Output: output,
				Error:  err,
			}, err
		}

		lastOutput = output
	}

	return &Result{
		TaskID: tasks[len(tasks)-1].ID,
		Output: lastOutput,
		Error:  nil,
	}, nil
}

// ConditionalRunner executes agents based on conditions
type ConditionalRunner struct {
	runner Runner
}

// NewConditionalRunner creates a new conditional runner
func NewConditionalRunner() *ConditionalRunner {
	return &ConditionalRunner{
		runner: New(1),
	}
}

// ConditionFunc evaluates whether a task should be executed
type ConditionFunc func(ctx context.Context, previousResult *Result) (bool, error)

// ConditionalTask represents a task with a condition
type ConditionalTask struct {
	Task      *Task
	Condition ConditionFunc
}

// RunConditional executes tasks based on conditions
func (cr *ConditionalRunner) RunConditional(ctx context.Context, tasks []*ConditionalTask) ([]*Result, error) {
	results := make([]*Result, 0, len(tasks))
	var lastResult *Result

	for _, ctask := range tasks {
		// Evaluate condition
		shouldRun := true
		if ctask.Condition != nil {
			var err error
			shouldRun, err = ctask.Condition(ctx, lastResult)
			if err != nil {
				return results, fmt.Errorf("condition evaluation failed: %w", err)
			}
		}

		if !shouldRun {
			continue
		}

		// Execute task
		output, err := cr.runner.Run(ctx, ctask.Task.Agent, ctask.Task.Input)
		result := &Result{
			TaskID: ctask.Task.ID,
			Output: output,
			Error:  err,
		}
		results = append(results, result)
		lastResult = result

		if err != nil {
			return results, err
		}
	}

	return results, nil
}
