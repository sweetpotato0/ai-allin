package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/sweetpotato0/ai-allin/agent"
	"github.com/sweetpotato0/ai-allin/contrib/provider/openai"
	frameworkmcp "github.com/sweetpotato0/ai-allin/tool/mcp"
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

	cfg := frameworkmcp.Config{
		Endpoint: *endpoint,
		Command:  *command,
	}

	switch strings.ToLower(*transport) {
	case "stream", "streamable", "http":
		cfg.Transport = frameworkmcp.TransportStreamable
	case "stdio", "command":
		cfg.Transport = frameworkmcp.TransportCommand
	default:
		log.Fatalf("unsupported transport: %s", *transport)
	}

	provider, err := frameworkmcp.NewProvider(ctx, cfg)

	if err != nil {
		log.Fatalf("connect MCP: %v", err)
	}
	defer provider.Close()

	fmt.Println("Fetching tool list...")
	tools, err := provider.Tools(ctx)
	if err != nil {
		log.Fatalf("list tools: %v", err)
	}
	for _, tool := range tools {
		fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
	}
	if len(tools) == 0 {
		fmt.Println("No tools were returned by the MCP server.")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY is required to run the Agentic RAG example")
	}

	baseURL := os.Getenv("OPENAI_API_BASE_URL")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY is required to run the Agentic RAG example")
	}
	llm := openai.New(openai.DefaultConfig().WithAPIKey(apiKey).WithBaseURL(baseURL).WithModel("gpt-4o"))

	ag := agent.New(
		agent.WithName("mcp-agent"),
		agent.WithSystemPrompt("You are a helpful assistant that can call MCP tools when needed."),
		agent.WithProvider(llm),
		agent.WithToolProvider(provider),
	)

	fmt.Println()
	fmt.Printf("Running agent with prompt: %q\n", *prompt)
	response, err := ag.Run(ctx, *prompt)
	if err != nil {
		log.Fatalf("agent run failed: %v", err)
	}

	fmt.Println("Agent response:")
	fmt.Println(response.Text())
}
