package mcp

import (
	"context"
	"fmt"

	"github.com/sweetpotato0/ai-allin/agent"
)

// AttachAgent loads MCP tools and registers them with the provided agent.
func (c *Client) AttachAgent(ctx context.Context, ag *agent.Agent) error {
	if c == nil {
		return fmt.Errorf("mcp: client is nil")
	}
	if ag == nil {
		return fmt.Errorf("mcp: agent is nil")
	}

	tools, err := c.BuildTools(ctx)
	if err != nil {
		return err
	}

	for _, t := range tools {
		if err := ag.RegisterTool(t); err != nil {
			return fmt.Errorf("mcp: register tool %s: %w", t.Name, err)
		}
	}

	return nil
}
