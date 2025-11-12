package token

import (
	"context"
	"testing"

	"github.com/sweetpotato0/ai-allin/rag/document"
)

func TestTokenChunkerRespectsOverlap(t *testing.T) {
	ch := New(WithMaxTokens(5), WithOverlapTokens(2))
	doc := document.Document{
		ID:      "tok-1",
		Content: "AADDCC 是 一 种 万能 药物 ， 但 也 可能 有 副作用 。",
	}

	chunks, err := ch.Chunk(context.Background(), doc)
	if err != nil {
		t.Fatalf("chunk error: %v", err)
	}
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	if chunks[0].Content == chunks[1].Content {
		t.Fatalf("expected overlapping but distinct chunks")
	}
}
