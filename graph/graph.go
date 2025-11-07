package graph

import (
	"context"
	"fmt"
)

// NodeType represents the type of a node in the graph
type NodeType string

const (
	NodeTypeStart     NodeType = "start"
	NodeTypeEnd       NodeType = "end"
	NodeTypeLLM       NodeType = "llm"
	NodeTypeTool      NodeType = "tool"
	NodeTypeCondition NodeType = "condition"
	NodeTypeCustom    NodeType = "custom"
)

// State represents the execution state passed between nodes
type State map[string]any

// NodeFunc is the function executed by a node
type NodeFunc func(context.Context, State) (State, error)

// ConditionFunc evaluates a condition and returns the next node name
type ConditionFunc func(context.Context, State) (string, error)

// Node represents a node in the execution graph
type Node struct {
	Name      string
	Type      NodeType
	Execute   NodeFunc
	Condition ConditionFunc     // Only for condition nodes
	Next      string            // Default next node
	NextMap   map[string]string // For condition nodes: condition result -> next node
}

// Graph represents an execution flow graph
type Graph struct {
	nodes     map[string]*Node
	startNode string
	endNode   string
	maxVisits int
}

// NewGraph creates a new graph
func NewGraph() *Graph {
	return &Graph{
		nodes:     make(map[string]*Node),
		maxVisits: 10,
	}
}

func (g *Graph) validateNode(node *Node) {
	// 校验 node 字段
	switch node.Type {
	case NodeTypeCondition:
		if node.Condition == nil {
			panic(fmt.Sprintf("condition node %s must have non-nil Condition function", node.Name))
		}
	default:
		if node.Execute == nil {
			panic(fmt.Sprintf("node %s of type %s must have non-nil Execute function", node.Name, node.Type))
		}
	}
}

// AddNode adds a node to the graph
func (g *Graph) AddNode(node *Node) {
	if node.Name == "" {
		panic("node name cannot be empty")
	}
	if _, exists := g.nodes[node.Name]; exists {
		panic(fmt.Sprintf("node %s already exists", node.Name))
	}

	g.validateNode(node)

	g.nodes[node.Name] = node

	// Auto-set start and end nodes
	if node.Type == NodeTypeStart {
		g.startNode = node.Name
	}
	if node.Type == NodeTypeEnd {
		g.endNode = node.Name
	}

}

// SetStartNode sets the start node
func (g *Graph) SetStartNode(name string) {
	if _, exists := g.nodes[name]; !exists {
		panic(fmt.Sprintf("node %s not found", name))
	}
	g.startNode = name
}

// SetEndNode sets the end node
func (g *Graph) SetEndNode(name string) {
	if _, exists := g.nodes[name]; !exists {
		panic(fmt.Sprintf("node %s not found", name))
	}
	g.endNode = name
}

// Execute runs the graph starting from the start node
func (g *Graph) Execute(ctx context.Context, initialState State) (State, error) {
	if g.startNode == "" {
		return nil, fmt.Errorf("start node not set")
	}

	currentNode := g.startNode
	state := initialState
	if state == nil {
		state = make(State)
	}

	visited := make(map[string]int) // Track visits to detect infinite loops

	for {
		// Check for infinite loop
		visited[currentNode]++
		if visited[currentNode] > g.maxVisits {
			return nil, fmt.Errorf("infinite loop detected at node %s", currentNode)
		}

		// Get current node
		node, exists := g.nodes[currentNode]
		if !exists {
			return nil, fmt.Errorf("node %s not found", currentNode)
		}

		// Check if we've reached the end
		if node.Type == NodeTypeEnd {
			return node.Execute(ctx, state)
		}

		// Execute node
		state, err := node.Execute(ctx, state)
		if err != nil {
			return nil, fmt.Errorf("error executing node %s: %w", currentNode, err)
		}

		// Determine next node
		var nextNode string
		if node.Type == NodeTypeCondition {
			// Use condition to determine next node
			result, err := node.Condition(ctx, state)
			if err != nil {
				return nil, fmt.Errorf("error evaluating condition at node %s: %w", currentNode, err)
			}
			nextNode = node.NextMap[result]
			// If nextNode is not set, use the default next node
			if nextNode == "" {
				nextNode = node.Next // Fallback to default
			}
		} else {
			nextNode = node.Next
		}

		if nextNode == "" {
			return nil, fmt.Errorf("no next node specified for node %s", currentNode)
		}

		currentNode = nextNode
	}
}

// GetNode returns a node by name
func (g *Graph) GetNode(name string) (*Node, error) {
	node, exists := g.nodes[name]
	if !exists {
		return nil, fmt.Errorf("node %s not found", name)
	}
	return node, nil
}

// SetMaxVisits sets the maximum number of visits to a node
func (g *Graph) SetMaxVisits(maxVisits int) {
	g.maxVisits = maxVisits
}

// Builder helps build graphs fluently
type Builder struct {
	graph *Graph
}

// NewBuilder creates a new graph builder
func NewBuilder() *Builder {
	return &Builder{
		graph: NewGraph(),
	}
}

// AddNode adds a node to the graph
func (b *Builder) AddNode(name string, nodeType NodeType, execute NodeFunc) *Builder {
	b.graph.AddNode(&Node{
		Name:    name,
		Type:    nodeType,
		Execute: execute,
	})
	return b
}

// AddConditionNode adds a condition node
func (b *Builder) AddConditionNode(name string, condition ConditionFunc, nextMap map[string]string) *Builder {
	b.graph.AddNode(&Node{
		Name:      name,
		Type:      NodeTypeCondition,
		Condition: condition,
		NextMap:   nextMap,
	})
	return b
}

// AddEdge connects two nodes
func (b *Builder) AddEdge(from, to string) *Builder {
	if node, exists := b.graph.nodes[from]; exists {
		node.Next = to
	}
	return b
}

// SetStart sets the start node
func (b *Builder) SetStart(name string) *Builder {
	b.graph.SetStartNode(name)
	return b
}

// SetEnd sets the end node
func (b *Builder) SetEnd(name string) *Builder {
	b.graph.SetEndNode(name)
	return b
}

// SetMaxVisits sets the maximum number of visits to a node
func (b *Builder) SetMaxVisits(maxVisits int) *Builder {
	b.graph.SetMaxVisits(maxVisits)
	return b
}

// Build returns the constructed graph
func (b *Builder) Build() *Graph {
	return b.graph
}
