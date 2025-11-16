package summarizer

import (
	"context"

	"github.com/sweetpotato0/ai-allin/rag/document"
)

type Summarizer interface {
	SummarizeChunks(ctx context.Context, chunks []document.Chunk) ([]document.Summary, error)
}
