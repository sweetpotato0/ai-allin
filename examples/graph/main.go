package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/sweetpotato0/ai-allin/graph"
)

func main() {
	fmt.Println("=== Graph Workflow Example ===")

	ctx := context.Background()

	// Example 1: Simple Linear Workflow
	fmt.Println("Example 1: Simple Linear Workflow")
	fmt.Println("-----------------------------------")

	simpleGraph := graph.NewBuilder().
		AddNode("start", graph.NodeTypeStart, func(ctx context.Context, state graph.State) (graph.State, error) {
			fmt.Println("→ Starting workflow...")
			state["count"] = 0
			return state, nil
		}).
		AddNode("increment", graph.NodeTypeCustom, func(ctx context.Context, state graph.State) (graph.State, error) {
			count := state["count"].(int)
			count++
			fmt.Printf("→ Incrementing count: %d\n", count)
			state["count"] = count
			return state, nil
		}).
		AddNode("double", graph.NodeTypeCustom, func(ctx context.Context, state graph.State) (graph.State, error) {
			count := state["count"].(int)
			count *= 2
			fmt.Printf("→ Doubling count: %d\n", count)
			state["count"] = count
			return state, nil
		}).
		AddNode("end", graph.NodeTypeEnd, func(ctx context.Context, state graph.State) (graph.State, error) {
			fmt.Printf("→ Workflow complete! Final count: %d\n", state["count"])
			return state, nil
		}).
		AddEdge("start", "increment").
		AddEdge("increment", "double").
		AddEdge("double", "end").
		SetStart("start").
		Build()

	finalState, err := simpleGraph.Execute(ctx, make(graph.State))
	if err != nil {
		log.Fatalf("Graph execution failed: %v", err)
	}
	fmt.Printf("Final state: %+v\n\n", finalState)

	// Example 2: Conditional Workflow
	fmt.Println("Example 2: Conditional Workflow")
	fmt.Println("--------------------------------")

	conditionalGraph := graph.NewBuilder().
		AddNode("start", graph.NodeTypeStart, func(ctx context.Context, state graph.State) (graph.State, error) {
			fmt.Println("→ Starting conditional workflow...")
			state["value"] = 15
			return state, nil
		}).
		AddConditionNode("check_value",
			func(ctx context.Context, state graph.State) (string, error) {
				value := state["value"].(int)
				fmt.Printf("→ Checking value: %d\n", value)
				if value > 10 {
					return "high", nil
				}
				return "low", nil
			},
			map[string]string{
				"high": "process_high",
				"low":  "process_low",
			},
		).
		AddNode("process_high", graph.NodeTypeCustom, func(ctx context.Context, state graph.State) (graph.State, error) {
			fmt.Println("→ Processing high value...")
			state["result"] = "HIGH"
			return state, nil
		}).
		AddNode("process_low", graph.NodeTypeCustom, func(ctx context.Context, state graph.State) (graph.State, error) {
			fmt.Println("→ Processing low value...")
			state["result"] = "LOW"
			return state, nil
		}).
		AddNode("end", graph.NodeTypeEnd, func(ctx context.Context, state graph.State) (graph.State, error) {
			fmt.Printf("→ Done! Result: %s\n", state["result"])
			return state, nil
		}).
		AddEdge("start", "check_value").
		AddEdge("process_high", "end").
		AddEdge("process_low", "end").
		SetStart("start").
		Build()

	finalState, err = conditionalGraph.Execute(ctx, make(graph.State))
	if err != nil {
		log.Fatalf("Graph execution failed: %v", err)
	}
	fmt.Printf("Final state: %+v\n\n", finalState)

	// Example 3: Data Processing Pipeline
	fmt.Println("Example 3: Data Processing Pipeline")
	fmt.Println("------------------------------------")

	pipelineGraph := graph.NewBuilder().
		AddNode("start", graph.NodeTypeStart, func(ctx context.Context, state graph.State) (graph.State, error) {
			fmt.Println("→ Loading data...")
			state["data"] = []string{"hello", "world", "from", "graph"}
			return state, nil
		}).
		AddNode("transform", graph.NodeTypeCustom, func(ctx context.Context, state graph.State) (graph.State, error) {
			fmt.Println("→ Transforming data...")
			data := state["data"].([]string)
			transformed := make([]string, len(data))
			for i, s := range data {
				transformed[i] = strings.ToUpper(s)
			}
			state["data"] = transformed
			return state, nil
		}).
		AddNode("filter", graph.NodeTypeCustom, func(ctx context.Context, state graph.State) (graph.State, error) {
			fmt.Println("→ Filtering data...")
			data := state["data"].([]string)
			filtered := make([]string, 0)
			for _, s := range data {
				if len(s) > 4 {
					filtered = append(filtered, s)
				}
			}
			state["data"] = filtered
			return state, nil
		}).
		AddNode("save", graph.NodeTypeCustom, func(ctx context.Context, state graph.State) (graph.State, error) {
			fmt.Println("→ Saving results...")
			data := state["data"].([]string)
			state["count"] = len(data)
			fmt.Printf("   Saved %d items: %v\n", len(data), data)
			return state, nil
		}).
		AddNode("end", graph.NodeTypeEnd, nil).
		AddEdge("start", "transform").
		AddEdge("transform", "filter").
		AddEdge("filter", "save").
		AddEdge("save", "end").
		SetStart("start").
		Build()

	finalState, err = pipelineGraph.Execute(ctx, make(graph.State))
	if err != nil {
		log.Fatalf("Graph execution failed: %v", err)
	}
	fmt.Printf("Final state: %+v\n", finalState)

	fmt.Println("\n=== Graph Features ===")
	fmt.Println("✓ Multiple node types (start, end, custom, condition)")
	fmt.Println("✓ State passing between nodes")
	fmt.Println("✓ Conditional branching")
	fmt.Println("✓ Infinite loop detection")
	fmt.Println("✓ Builder pattern for easy construction")
}
