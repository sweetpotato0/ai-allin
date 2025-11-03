package provider

import (
	"context"

	"github.com/sweetpotato0/ai-allin/message"
)

// Provider .
type Provider interface {
	Generate(ctx context.Context, messages []*message.Message, tools []map[string]interface{}) (*message.Message, error)
}
