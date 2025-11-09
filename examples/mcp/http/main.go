package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sweetpotato0/ai-allin/examples/mcp/demo"
)

func main() {
	host := flag.String("host", "127.0.0.1", "Host to bind")
	port := flag.Int("port", 8080, "Port to bind")
	path := flag.String("path", "/mcp", "HTTP path used for the MCP streamable endpoint")
	flag.Parse()

	server := demo.NewServer("ai-allin-http-demo")

	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		if r.URL.Path == *path {
			return server
		}
		return nil
	}, nil)

	mux := http.NewServeMux()
	mux.Handle(*path, handler)

	addr := fmt.Sprintf("%s:%d", *host, *port)
	log.Printf("Serving MCP streamable endpoint at http://%s%s", addr, *path)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("http server stopped: %v", err)
	}
}
