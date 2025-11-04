package enricher

import (
	"github.com/sweetpotato0/ai-allin/middleware"
)

// EnricherFunc enriches the context
type EnricherFunc func(*middleware.Context) error

// ContextEnricher adds additional data to the middleware context
type ContextEnricher struct {
	enricher EnricherFunc
}

// NewContextEnricher creates a context enriching middleware
func NewContextEnricher(enricher EnricherFunc) *ContextEnricher {
	return &ContextEnricher{enricher: enricher}
}

// Name returns the middleware name
func (m *ContextEnricher) Name() string {
	return "ContextEnricher"
}

// Execute enriches the context
func (m *ContextEnricher) Execute(ctx *middleware.Context, next middleware.Handler) error {
	if m.enricher != nil {
		if err := m.enricher(ctx); err != nil {
			return err
		}
	}
	return next(ctx)
}
