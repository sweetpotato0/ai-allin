package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sweetpotato0/ai-allin/examples/mcp/demo"
)

func main() {
	server := demo.NewServer("ai-allin-stdio-demo")
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("stdio server stopped: %v", err)
	}
}
