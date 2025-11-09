package demo

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	demoCities = []string{"san-francisco", "new-york", "london", "tokyo"}
	weatherMap = map[string]string{
		"san-francisco": "San Francisco: 🌁 foggy mornings around 16°C with afternoon sun.",
		"new-york":      "New York: 🌤 clear skies near 24°C and a light Hudson breeze.",
		"london":        "London: 🌦 scattered showers, 18°C, pack a light jacket.",
		"tokyo":         "Tokyo: ☀️ sunny, 27°C, humidity picking up after sunset.",
	}
)

// NewServer builds the demo MCP server shared by both transports.
func NewServer(name string) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    name,
		Version: "0.1.0",
		Title:   "ai-allin demo server",
	}, nil)

	addWeatherTool(server)
	addCityLister(server)
	addClockTool(server)

	return server
}

func addWeatherTool(server *mcp.Server) {
	type args struct {
		City string `json:"city" jsonschema:"Demo city to inspect (san-francisco, new-york, london, tokyo)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_weather",
		Description: "Return a short weather blurb for popular demo cities",
	}, func(ctx context.Context, req *mcp.CallToolRequest, a args) (*mcp.CallToolResult, any, error) {
		log.Printf("call get_weather. req: %#v, args: %#v\n", req, a)
		city := strings.ToLower(strings.TrimSpace(a.City))
		if city == "" {
			return nil, nil, fmt.Errorf("city is required (try one of %s)", strings.Join(demoCities, ", "))
		}

		report, ok := weatherMap[city]
		if !ok {
			return nil, nil, fmt.Errorf("unsupported city %q (valid: %s)", city, strings.Join(demoCities, ", "))
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: report},
			},
		}, nil, nil
	})
}

func addCityLister(server *mcp.Server) {
	type args struct{}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_cities",
		Description: "List the demo cities supported by get_weather",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ args) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: strings.Join(demoCities, ", ")},
			},
		}, nil, nil
	})
}

func addClockTool(server *mcp.Server) {
	type args struct {
		Timezone string `json:"timezone,omitempty" jsonschema:"Optional IANA timezone, defaults to UTC"`
		Format   string `json:"format,omitempty" jsonschema:"Optional Go time layout, defaults to RFC3339"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "system_time",
		Description: "Return the current time for the given timezone",
	}, func(ctx context.Context, req *mcp.CallToolRequest, a args) (*mcp.CallToolResult, any, error) {
		loc := time.UTC
		if tz := strings.TrimSpace(a.Timezone); tz != "" {
			location, err := time.LoadLocation(tz)
			if err != nil {
				return nil, nil, fmt.Errorf("load timezone %q: %w", tz, err)
			}
			loc = location
		}

		layout := time.RFC3339
		if f := strings.TrimSpace(a.Format); f != "" {
			layout = f
		}

		now := time.Now().In(loc).Format(layout)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: now},
			},
		}, nil, nil
	})
}
