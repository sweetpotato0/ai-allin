package mcp

import (
	"strings"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestNormalizeContent(t *testing.T) {
	content := []sdkmcp.Content{
		&sdkmcp.TextContent{Text: "hello"},
		&sdkmcp.ResourceLink{URI: "file://foo", Name: "foo.txt"},
	}

	got := normalizeContent(content)
	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), got)
	}
	if lines[0] != "hello" {
		t.Fatalf("expected first line to be 'hello', got %q", lines[0])
	}
	if !strings.Contains(lines[1], "\"resource_link\"") {
		t.Fatalf("expected JSON output to include resource link type: %q", lines[1])
	}
}

func TestParametersFromSchema(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "search query",
			},
			"limit": map[string]any{
				"type":        "number",
				"description": "maximum items",
				"default":     10,
			},
		},
		"required": []any{"query"},
	}

	params := parametersFromSchema(schema)
	if len(params) != 2 {
		t.Fatalf("expected 2 parameters, got %d", len(params))
	}

	if params[0].Name != "limit" || params[1].Name != "query" {
		t.Fatalf("expected parameters sorted alphabetically, got %v", []string{params[0].Name, params[1].Name})
	}

	if !params[1].Required {
		t.Fatalf("expected 'query' to be required")
	}
}
