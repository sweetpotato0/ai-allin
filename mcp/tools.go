package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sweetpotato0/ai-allin/tool"
)

// ToolError is returned when the MCP server reports an error response.
type ToolError struct {
	Name    string
	Message string
}

func (e *ToolError) Error() string {
	return fmt.Sprintf("mcp tool %s: %s", e.Name, e.Message)
}

// ListTools retrieves a single page of tools from the MCP server.
func (c *Client) ListTools(ctx context.Context, cursor string) (*sdkmcp.ListToolsResult, error) {
	if c.session == nil {
		return nil, ErrClientClosed
	}
	params := &sdkmcp.ListToolsParams{}
	if cursor != "" {
		params.Cursor = cursor
	}
	return c.session.ListTools(ctx, params)
}

// ListAllTools returns the full set of tools exposed by the MCP server.
func (c *Client) ListAllTools(ctx context.Context) ([]*sdkmcp.Tool, error) {
	if c.session == nil {
		return nil, ErrClientClosed
	}

	params := &sdkmcp.ListToolsParams{}
	var (
		cursor string
		tools  []*sdkmcp.Tool
	)

	for {
		if cursor != "" {
			params.Cursor = cursor
		}
		res, err := c.session.ListTools(ctx, params)
		if err != nil {
			return nil, err
		}
		tools = append(tools, res.Tools...)
		if res.NextCursor == "" {
			break
		}
		cursor = res.NextCursor
	}

	return tools, nil
}

// CallTool invokes a remote MCP tool and returns the textual response.
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	if c.session == nil {
		return "", ErrClientClosed
	}

	params := &sdkmcp.CallToolParams{
		Name:      name,
		Arguments: args,
	}

	result, err := c.session.CallTool(ctx, params)
	if err != nil {
		return "", err
	}

	message := normalizeContent(result.Content)
	if result.IsError {
		if message == "" {
			message = "tool returned error without message"
		}
		return "", &ToolError{Name: name, Message: message}
	}

	return message, nil
}

// BuildTools converts MCP tool definitions to ai-allin tool registrations.
func (c *Client) BuildTools(ctx context.Context) ([]*tool.Tool, error) {
	defs, err := c.ListAllTools(ctx)
	if err != nil {
		return nil, err
	}

	tools := make([]*tool.Tool, 0, len(defs))
	for _, def := range defs {
		if def == nil {
			continue
		}

		description := def.Description
		if description == "" && def.Annotations != nil {
			description = def.Annotations.Title
		}

		params := parametersFromSchema(def.InputSchema)

		remoteName := def.Name
		toolDef := &tool.Tool{
			Name:        remoteName,
			Description: description,
			Parameters:  params,
		}

		toolDef.Handler = func(ctx context.Context, args map[string]interface{}) (string, error) {
			if args == nil {
				args = make(map[string]interface{})
			}
			return c.CallTool(ctx, remoteName, args)
		}

		tools = append(tools, toolDef)
	}

	return tools, nil
}

// RegisterTools fetches remote tools and registers them with a local registry.
func (c *Client) RegisterTools(ctx context.Context, registry *tool.Registry) error {
	tools, err := c.BuildTools(ctx)
	if err != nil {
		return err
	}
	for _, t := range tools {
		if err := registry.Register(t); err != nil {
			return fmt.Errorf("register tool %s: %w", t.Name, err)
		}
	}
	return nil
}

func normalizeContent(content []sdkmcp.Content) string {
	if len(content) == 0 {
		return ""
	}

	parts := make([]string, 0, len(content))
	for _, c := range content {
		switch v := c.(type) {
		case *sdkmcp.TextContent:
			parts = append(parts, v.Text)
		default:
			if data, err := c.MarshalJSON(); err == nil {
				parts = append(parts, string(data))
			}
		}
	}

	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func parametersFromSchema(schema any) []tool.Parameter {
	schemaMap := toMap(schema)
	if schemaMap == nil {
		return nil
	}

	typeVal, _ := schemaMap["type"].(string)
	if strings.ToLower(typeVal) != "object" {
		return nil
	}

	propsRaw, ok := schemaMap["properties"].(map[string]any)
	if !ok || len(propsRaw) == 0 {
		return nil
	}

	requiredSet := make(map[string]struct{})
	if requiredRaw, ok := schemaMap["required"]; ok {
		if list, ok := requiredRaw.([]any); ok {
			for _, item := range list {
				if name, ok := item.(string); ok {
					requiredSet[name] = struct{}{}
				}
			}
		}
	}

	names := make([]string, 0, len(propsRaw))
	for name := range propsRaw {
		names = append(names, name)
	}
	sort.Strings(names)

	parameters := make([]tool.Parameter, 0, len(names))
	for _, name := range names {
		prop := propsRaw[name]
		propMap, ok := prop.(map[string]any)
		if !ok {
			continue
		}

		param := tool.Parameter{
			Name:        name,
			Description: stringValue(propMap["description"]),
			Type:        stringValue(propMap["type"]),
			Default:     propMap["default"],
		}

		if _, ok := requiredSet[name]; ok {
			param.Required = true
		}

		if enums, ok := toStringSlice(propMap["enum"]); ok {
			param.Enum = enums
		}

		if param.Type == "" {
			param.Type = inferType(propMap)
		}

		parameters = append(parameters, param)
	}

	return parameters
}

func inferType(prop map[string]any) string {
	if _, ok := prop["items"]; ok {
		return "array"
	}
	if _, ok := prop["properties"]; ok {
		return "object"
	}
	return "string"
}

func stringValue(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func toStringSlice(v any) ([]string, bool) {
	raw, ok := v.([]any)
	if !ok {
		return nil, false
	}
	values := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok {
			values = append(values, s)
		}
	}
	return values, true
}

func toMap(v any) map[string]any {
	switch value := v.(type) {
	case map[string]any:
		return value
	case json.RawMessage:
		var out map[string]any
		if err := json.Unmarshal(value, &out); err != nil {
			return nil
		}
		return out
	case []byte:
		var out map[string]any
		if err := json.Unmarshal(value, &out); err != nil {
			return nil
		}
		return out
	default:
		return nil
	}
}
