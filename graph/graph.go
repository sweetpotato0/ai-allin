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
	Name           string
	Type           NodeType
	Execute        NodeFunc
	Condition      ConditionFunc     // Only for condition nodes
	NextNodes      []string          // Outgoing edges (order defines default)
	NextMap        map[string]string // For condition nodes: condition result -> next node
	WaitAllParents bool              // Whether execution waits for all parents to finish
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
	// validate node

	if node.Name == "" {
		panic("node name cannot be empty")
	}

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

func (n *Node) addNext(name string) {
	n.NextNodes = append(n.NextNodes, name)
}

func (n *Node) nextList() []string {
	if n == nil {
		return nil
	}

	seen := make(map[string]struct{})
	var result []string

	for _, child := range n.NextNodes {
		if _, ok := seen[child]; ok {
			continue
		}
		seen[child] = struct{}{}
		result = append(result, child)
	}

	return result
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

// Execute runs the graph starting from the configured start node.
// Algorithm outline:
//  1. Pre-compute how many unique parents each node has (needed for fork-join semantics).
//  2. Use a queue to perform breadth-first scheduling: dequeue a node, execute it, determine
//     which children are activated, and propagate signals to them.
//  3. handleChildSignal inspects whether the current parent actually triggered a child
//     (participated) and whether the child waits for all parents before enqueuing it,
//     preventing missed or duplicated executions along conditional branches.
func (g *Graph) Execute(ctx context.Context, initialState State) (State, error) {
	if g.startNode == "" {
		return nil, fmt.Errorf("start node not set")
	}

	state := initialState
	if state == nil {
		state = make(State)
	}

	// expectedParents stores how many unique parents each node has.
	expectedParents := g.buildParentCounts()
	// completedParents counts how many parents (participating or not) already reported completion.
	completedParents := make(map[string]int)
	// parentHits counts how many parents actually produced output for a child.
	parentHits := make(map[string]int)
	// awaiting tracks whether a node is already queued to avoid duplicates.
	awaiting := make(map[string]bool)
	// queue holds nodes pending execution, starting with the start node.
	queue := []string{g.startNode}
	awaiting[g.startNode] = true
	visited := make(map[string]int)

	for len(queue) > 0 {
		currentNode := queue[0]
		queue = queue[1:]
		awaiting[currentNode] = false

		// Fetch node metadata; failure means the graph definition is inconsistent.
		node, exists := g.nodes[currentNode]
		if !exists {
			return nil, fmt.Errorf("node %s not found", currentNode)
		}

		// Detect runaway loops by counting how many times we revisit a node.
		visited[currentNode]++
		if visited[currentNode] > g.maxVisits {
			return nil, fmt.Errorf("infinite loop detected at node %s", currentNode)
		}

		// End nodes terminate execution immediately and return the final state.
		if node.Type == NodeTypeEnd {
			return node.Execute(ctx, state)
		}

		// Determine which child nodes should run next (e.g., the taken branch of a condition).
		nextNodes, err := g.resolveNextNodes(ctx, node, state)
		if err != nil {
			return nil, err
		}

		// allChildren captures every potential child; useful for notifying skipped branches.
		allChildren := g.staticChildren(node)
		triggered := make(map[string]struct{}, len(nextNodes))

		// Send participation signals to children that were actually triggered.
		for _, child := range nextNodes {
			triggered[child] = struct{}{}
			if err := g.handleChildSignal(child, true, parentHits, completedParents, expectedParents, awaiting, &queue); err != nil {
				return nil, err
			}
		}

		// Inform remaining children that this parent finished without triggering them.
		for _, child := range allChildren {
			if _, ok := triggered[child]; ok {
				continue
			}
			if err := g.handleChildSignal(child, false, parentHits, completedParents, expectedParents, awaiting, &queue); err != nil {
				return nil, err
			}
		}

		parentHits[currentNode] = 0
		completedParents[currentNode] = 0
	}

	return state, nil
}

func (g *Graph) resolveNextNodes(ctx context.Context, node *Node, state State) ([]string, error) {
	switch node.Type {
	case NodeTypeCondition:
		result, err := node.Condition(ctx, state)
		if err != nil {
			return nil, fmt.Errorf("error evaluating condition at node %s: %w", node.Name, err)
		}
		nextNode := node.NextMap[result]
		if nextNode == "" {
			return nil, fmt.Errorf("no next node specified for node %s", node.Name)
		}
		return []string{nextNode}, nil
	default:
		var err error
		state, err = node.Execute(ctx, state)
		if err != nil {
			return nil, fmt.Errorf("error executing node %s: %w", node.Name, err)
		}
		nextNodes := node.nextList()
		if len(nextNodes) == 0 {
			return nil, fmt.Errorf("no next node specified for node %s", node.Name)
		}
		return nextNodes, nil
	}
}

func (g *Graph) handleChildSignal(child string, participated bool, parentHits map[string]int, completedParents map[string]int, expectedParents map[string]int, awaiting map[string]bool, queue *[]string) error {
	target, exists := g.nodes[child]
	if !exists {
		return fmt.Errorf("node %s not found", child)
	}

	if target.WaitAllParents {
		// if parent actually run, `participated` pass true
		if participated {
			parentHits[child]++ // if parent node run, add one
		}
		completedParents[child]++          // if node walk(run or not run), add one
		required := expectedParents[child] // number of parents include not run
		if required <= 0 {
			required = 1
		}
		if completedParents[child] < required || parentHits[child] == 0 || awaiting[child] {
			return nil
		}
		awaiting[child] = true
		*queue = append(*queue, child)
		return nil
	}

	if !participated {
		return nil
	}

	parentHits[child]++

	if awaiting[child] {
		return nil
	}

	awaiting[child] = true
	*queue = append(*queue, child)
	return nil
}

func (g *Graph) buildParentCounts() map[string]int {
	counts := make(map[string]int)
	for _, node := range g.nodes {
		for _, child := range g.staticChildren(node) {
			counts[child]++
		}
	}
	return counts
}

func (g *Graph) staticChildren(node *Node) []string {
	if node == nil {
		return nil
	}

	seen := make(map[string]struct{})
	add := func(out *[]string, name string) {
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		*out = append(*out, name)
	}

	var result []string
	if node.Type == NodeTypeCondition {
		for _, child := range node.NextMap {
			add(&result, child)
		}
	}

	for _, child := range node.NextNodes {
		add(&result, child)
	}
	return result
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
		node.addNext(to)
	}
	return b
}

// RequireAllParents marks a node to wait for all of its parents before executing.
func (b *Builder) RequireAllParents(name string) *Builder {
	node, exists := b.graph.nodes[name]
	if !exists {
		panic(fmt.Sprintf("node %s not found", name))
	}
	node.WaitAllParents = true
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
