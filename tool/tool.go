package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// Parameter defines a tool parameter
type Parameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"` // string, number, boolean, object, array
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Enum        []string    `json:"enum,omitempty"`
	Default     interface{} `json:"default,omitempty"`
}

// Tool represents a callable tool/function
type Tool struct {
	Name        string                                                        `json:"name"`
	Description string                                                        `json:"description"`
	Parameters  []Parameter                                                   `json:"parameters"`
	Handler     func(context.Context, map[string]interface{}) (string, error) `json:"-"`
}

// Execute runs the tool with given arguments
func (t *Tool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	if t.Handler == nil {
		return "", fmt.Errorf("tool %s has no handler", t.Name)
	}

	// Validate required parameters
	if err := t.ValidateArgs(args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	return t.Handler(ctx, args)
}

// ValidateArgs validates the provided arguments against the tool's parameters
func (t *Tool) ValidateArgs(args map[string]interface{}) error {
	for _, param := range t.Parameters {
		if param.Required {
			if _, ok := args[param.Name]; !ok {
				return fmt.Errorf("missing required parameter: %s", param.Name)
			}
		}
	}
	return nil
}

// ToJSONSchema returns the tool definition in JSON schema format for LLM
func (t *Tool) ToJSONSchema() map[string]interface{} {
	properties := make(map[string]interface{})
	required := make([]string, 0)

	for _, param := range t.Parameters {
		prop := map[string]interface{}{
			"type":        param.Type,
			"description": param.Description,
		}
		if len(param.Enum) > 0 {
			prop["enum"] = param.Enum
		}
		if param.Default != nil {
			prop["default"] = param.Default
		}
		properties[param.Name] = prop

		if param.Required {
			required = append(required, param.Name)
		}
	}

	return map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name":        t.Name,
			"description": t.Description,
			"parameters": map[string]interface{}{
				"type":       "object",
				"properties": properties,
				"required":   required,
			},
		},
	}
}

// Registry manages a collection of tools
// All operations are thread-safe using RWMutex protection
type Registry struct {
	mu    sync.RWMutex // Protects tools map
	tools map[string]*Tool
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]*Tool),
	}
}

// Register adds a tool to the registry
func (r *Registry) Register(tool *Tool) error {
	if tool.Name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[tool.Name]; exists {
		return fmt.Errorf("tool %s already registered", tool.Name)
	}
	r.tools[tool.Name] = tool
	return nil
}

// Upsert adds or replaces a tool definition in the registry.
func (r *Registry) Upsert(tool *Tool) error {
	if tool == nil || tool.Name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.tools == nil {
		r.tools = make(map[string]*Tool)
	}
	r.tools[tool.Name] = tool
	return nil
}

// Get retrieves a tool by name
func (r *Registry) Get(name string) (*Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool %s not found", name)
	}
	return tool, nil
}

// List returns all registered tools
func (r *Registry) List() []*Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]*Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ToJSONSchemas returns all tools in JSON schema format
func (r *Registry) ToJSONSchemas() []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schemas := make([]map[string]interface{}, 0, len(r.tools))
	for _, tool := range r.tools {
		schemas = append(schemas, tool.ToJSONSchema())
	}
	return schemas
}

// Execute runs a tool by name with given arguments
func (r *Registry) Execute(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	tool, err := r.Get(name)
	if err != nil {
		return "", err
	}
	return tool.Execute(ctx, args)
}

// MarshalJSON customizes JSON marshaling for Registry
func (r *Registry) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.ToJSONSchemas())
}
