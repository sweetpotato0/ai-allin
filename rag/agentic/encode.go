package agentic

import (
	"encoding/json"
	"fmt"
	"strings"
)

// decodeJSON tries to unmarshal the raw model output into T after stripping fences.
func decodeJSON[T any](raw string) (*T, error) {
	clean := sanitizeJSON(raw)
	var out T
	if err := json.Unmarshal([]byte(clean), &out); err != nil {
		return nil, fmt.Errorf("decode JSON: %w", err)
	}
	return &out, nil
}

func sanitizeJSON(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "```") {
		trimmed = trimmed[3:]
		trimmed = strings.TrimPrefix(trimmed, "json")
		trimmed = strings.TrimPrefix(trimmed, "JSON")
		if idx := strings.Index(trimmed, "```"); idx >= 0 {
			trimmed = trimmed[:idx]
		}
	}
	return strings.TrimSpace(trimmed)
}
