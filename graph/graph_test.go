package graph

import (
	"context"
	"testing"
)

func TestNewGraph(t *testing.T) {
	g := NewGraph()
	if g == nil {
		t.Errorf("NewGraph returned nil")
	}
}

func TestAddNode(t *testing.T) {
	g := NewGraph()

	node := &Node{
		Name: "test_node",
		Type: NodeTypeCustom,
	}

	g.AddNode(node)

	// Verify node was added
	retrieved, err := g.GetNode("test_node")
	if err != nil {
		t.Errorf("Failed to retrieve added node: %v", err)
	}

	if retrieved.Name != "test_node" {
		t.Errorf("Retrieved node name mismatch")
	}
}

func TestAddNodeEmptyName(t *testing.T) {
	g := NewGraph()

	node := &Node{
		Name: "",
		Type: NodeTypeCustom,
	}

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected function to panic, but it did not")
		} else {
			if r != "node name cannot be empty" {
				t.Errorf("Expected panic value to be 'node name cannot be empty', but got %v", r)
			}
		}
	}()

	g.AddNode(node)
}

func TestAddNodeDuplicate(t *testing.T) {
	g := NewGraph()

	node1 := &Node{Name: "dup_node", Type: NodeTypeCustom}
	node2 := &Node{Name: "dup_node", Type: NodeTypeCustom}

	g.AddNode(node1)

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected function to panic, but it did not")
		} else {
			if r != "node dup_node already exists" {
				t.Errorf("Expected panic value to be 'node dup_node already exists', but got %v", r)
			}
		}
	}()
	g.AddNode(node2)
}

func TestAutoSetStartNode(t *testing.T) {
	g := NewGraph()

	startNode := &Node{
		Name: "start",
		Type: NodeTypeStart,
	}

	g.AddNode(startNode)

	if g.startNode != "start" {
		t.Errorf("Start node not automatically set")
	}
}

func TestAutoSetEndNode(t *testing.T) {
	g := NewGraph()

	endNode := &Node{
		Name: "end",
		Type: NodeTypeEnd,
	}

	g.AddNode(endNode)

	if g.endNode != "end" {
		t.Errorf("End node not automatically set")
	}
}

func TestSetStartNode(t *testing.T) {
	g := NewGraph()

	node := &Node{Name: "start_node", Type: NodeTypeCustom}
	g.AddNode(node)

	g.SetStartNode("start_node")

	if g.startNode != "start_node" {
		t.Errorf("Start node not set correctly")
	}
}

func TestSetStartNodeNotFound(t *testing.T) {
	g := NewGraph()

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected function to panic, but it did not")
		} else {
			if r != "node nonexistent not found" {
				t.Errorf("Expected panic value to be 'node nonexistent not found', but got %v", r)
			}
		}
	}()

	g.SetStartNode("nonexistent")
}

func TestSetEndNode(t *testing.T) {
	g := NewGraph()

	node := &Node{Name: "end_node", Type: NodeTypeCustom}
	g.AddNode(node)

	g.SetEndNode("end_node")

	if g.endNode != "end_node" {
		t.Errorf("End node not set correctly")
	}
}

func TestExecuteSimpleLinearGraph(t *testing.T) {
	g := NewGraph()

	// Create a simple linear graph: start -> node1 -> node2 -> end
	startNode := &Node{
		Name: "start",
		Type: NodeTypeStart,
		Execute: func(ctx context.Context, state State) (State, error) {
			state["started"] = true
			return state, nil
		},
		Next: "node1",
	}

	node1 := &Node{
		Name: "node1",
		Type: NodeTypeCustom,
		Execute: func(ctx context.Context, state State) (State, error) {
			state["step1"] = true
			return state, nil
		},
		Next: "node2",
	}

	node2 := &Node{
		Name: "node2",
		Type: NodeTypeCustom,
		Execute: func(ctx context.Context, state State) (State, error) {
			state["step2"] = true
			return state, nil
		},
		Next: "end",
	}

	endNode := &Node{
		Name: "end",
		Type: NodeTypeEnd,
	}

	g.AddNode(startNode)
	g.AddNode(node1)
	g.AddNode(node2)
	g.AddNode(endNode)

	state, err := g.Execute(context.Background(), nil)
	if err != nil {
		t.Errorf("Graph execution failed: %v", err)
	}

	// Verify state was updated correctly
	if state["started"] != true {
		t.Errorf("Start node was not executed")
	}

	if state["step1"] != true {
		t.Errorf("Node1 was not executed")
	}

	if state["step2"] != true {
		t.Errorf("Node2 was not executed")
	}
}

func TestExecuteWithCondition(t *testing.T) {
	g := NewGraph()

	startNode := &Node{
		Name: "start",
		Type: NodeTypeStart,
		Execute: func(ctx context.Context, state State) (State, error) {
			state["value"] = 5
			return state, nil
		},
		Next: "decision",
	}

	decisionNode := &Node{
		Name: "decision",
		Type: NodeTypeCondition,
		Condition: func(ctx context.Context, state State) (string, error) {
			val := state["value"].(int)
			if val > 10 {
				return "high", nil
			}
			return "low", nil
		},
		NextMap: map[string]string{
			"high": "node_high",
			"low":  "node_low",
		},
	}

	nodeHigh := &Node{
		Name: "node_high",
		Type: NodeTypeCustom,
		Execute: func(ctx context.Context, state State) (State, error) {
			state["branch"] = "high"
			return state, nil
		},
		Next: "end",
	}

	nodeLow := &Node{
		Name: "node_low",
		Type: NodeTypeCustom,
		Execute: func(ctx context.Context, state State) (State, error) {
			state["branch"] = "low"
			return state, nil
		},
		Next: "end",
	}

	endNode := &Node{
		Name: "end",
		Type: NodeTypeEnd,
	}

	g.AddNode(startNode)
	g.AddNode(decisionNode)
	g.AddNode(nodeHigh)
	g.AddNode(nodeLow)
	g.AddNode(endNode)

	state, err := g.Execute(context.Background(), nil)
	if err != nil {
		t.Errorf("Graph execution failed: %v", err)
	}

	// Should take "low" branch
	if state["branch"] != "low" {
		t.Errorf("Expected low branch, got %v", state["branch"])
	}
}

func TestExecuteNoStartNode(t *testing.T) {
	g := NewGraph()

	node := &Node{Name: "node", Type: NodeTypeCustom}
	g.AddNode(node)

	_, err := g.Execute(context.Background(), nil)
	if err == nil {
		t.Errorf("Expected error when executing graph without start node")
	}
}

func TestExecuteNodeNotFound(t *testing.T) {
	g := NewGraph()

	startNode := &Node{
		Name: "start",
		Type: NodeTypeStart,
		Next: "nonexistent",
	}

	g.AddNode(startNode)

	_, err := g.Execute(context.Background(), nil)
	if err == nil {
		t.Errorf("Expected error when executing with non-existent next node")
	}
}

func TestExecuteInfiniteLoop(t *testing.T) {
	g := NewGraph()

	// Create a loop: start -> node1 -> start
	startNode := &Node{
		Name: "start",
		Type: NodeTypeStart,
		Execute: func(ctx context.Context, state State) (State, error) {
			return state, nil
		},
		Next: "node1",
	}

	node1 := &Node{
		Name: "node1",
		Type: NodeTypeCustom,
		Execute: func(ctx context.Context, state State) (State, error) {
			return state, nil
		},
		Next: "start",
	}

	g.AddNode(startNode)
	g.AddNode(node1)

	_, err := g.Execute(context.Background(), nil)
	if err == nil {
		t.Errorf("Expected error for infinite loop")
	}
}

func TestExecuteWithInitialState(t *testing.T) {
	g := NewGraph()

	node := &Node{
		Name: "start",
		Type: NodeTypeStart,
		Execute: func(ctx context.Context, state State) (State, error) {
			state["processed"] = true
			return state, nil
		},
		Next: "end",
	}

	endNode := &Node{
		Name: "end",
		Type: NodeTypeEnd,
	}

	g.AddNode(node)
	g.AddNode(endNode)

	initialState := State{"initial": "value"}
	state, err := g.Execute(context.Background(), initialState)
	if err != nil {
		t.Errorf("Execution failed: %v", err)
	}

	if state["initial"] != "value" {
		t.Errorf("Initial state not preserved")
	}

	if state["processed"] != true {
		t.Errorf("State not updated by node")
	}
}

func TestNewBuilder(t *testing.T) {
	builder := NewBuilder()
	if builder == nil {
		t.Errorf("NewBuilder returned nil")
	}
	if builder.graph == nil {
		t.Errorf("Builder graph is nil")
	}
}

func TestBuilderAddNode(t *testing.T) {
	builder := NewBuilder()

	builder.AddNode("test", NodeTypeCustom, func(ctx context.Context, state State) (State, error) {
		return state, nil
	})

	node, err := builder.graph.GetNode("test")
	if err != nil {
		t.Errorf("Failed to get added node: %v", err)
	}

	if node.Name != "test" {
		t.Errorf("Node name mismatch")
	}
}

func TestBuilderAddConditionNode(t *testing.T) {
	builder := NewBuilder()

	builder.AddConditionNode("condition", func(ctx context.Context, state State) (string, error) {
		return "result", nil
	}, map[string]string{"result": "next"})

	node, err := builder.graph.GetNode("condition")
	if err != nil {
		t.Errorf("Failed to get condition node: %v", err)
	}

	if node.Type != NodeTypeCondition {
		t.Errorf("Node type should be condition")
	}
}

func TestGetNodeNotFound(t *testing.T) {
	g := NewGraph()

	_, err := g.GetNode("nonexistent")
	if err == nil {
		t.Errorf("Expected error when getting non-existent node")
	}
}
