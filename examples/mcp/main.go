package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/sweetpotato0/ai-allin/agent"
	frameworkmcp "github.com/sweetpotato0/ai-allin/mcp"
)

func main() {
	var (
		transport = flag.String("transport", "stream", "Transport to MCP server: stream | stdio")
		endpoint  = flag.String("endpoint", "http://localhost:8080/mcp", "Streamable MCP endpoint")
		command   = flag.String("command", "./mcp-server", "Command to launch for stdio transport")
		prompt    = flag.String("prompt", "Use available tools to describe the weather in San Francisco.", "Prompt to send to the agent")
	)
	flag.Parse()

	ctx := context.Background()

	client, err := connect(ctx, strings.ToLower(*transport), *endpoint, *command)
	if err != nil {
		log.Fatalf("connect MCP: %v", err)
	}
	defer client.Close()

	if init := client.InitializeResult(); init != nil {
		fmt.Printf("Connected to MCP server %q (protocol %s)\n", init.ServerInfo.Name, init.ProtocolVersion)
		if init.Instructions != "" {
			fmt.Printf("Server instructions: %s\n", init.Instructions)
		}
	}

	fmt.Println("Fetching tool list...")
	tools, err := client.ListAllTools(ctx)
	if err != nil {
		log.Fatalf("list tools: %v", err)
	}
	for _, tool := range tools {
		fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
	}
	if len(tools) == 0 {
		fmt.Println("No tools were returned by the MCP server.")
	}

	ag := agent.New(
		agent.WithName("mcp-agent"),
		agent.WithSystemPrompt("You are a helpful assistant that can call MCP tools when needed."),
	)

	if err := client.AttachAgent(ctx, ag); err != nil {
		log.Fatalf("attach tools to agent: %v", err)
	}

	fmt.Println()
	fmt.Printf("Running agent with prompt: %q\n", *prompt)
	response, err := ag.Run(ctx, *prompt)
	if err != nil {
		log.Fatalf("agent run failed: %v", err)
	}

	fmt.Println("Agent response:")
	fmt.Println(response)
}

func connect(ctx context.Context, transport, endpoint, command string) (*frameworkmcp.Client, error) {
	switch transport {
	case "stream", "streamable", "http":
		return frameworkmcp.NewStreamableClient(ctx, endpoint)
	case "stdio", "command":
		return frameworkmcp.NewStdioClient(ctx, command)
	default:
		return nil, errors.New("unsupported transport: " + transport)
	}
}
