package chunking

import (
	"context"
	"strings"
	"testing"

	"github.com/sweetpotato0/ai-allin/rag/document"
)

func TestSimpleChunkerMergesShortSegmentsAndTags(t *testing.T) {
	ch := NewSimpleChunker(
		WithChunkSize(120),
		WithMinChunkSize(60),
		WithOverlap(80),
		WithSeparator("\n\n"),
	)

	doc := document.Document{
		ID:      "aaddcc",
		Content: "# AADDCC\n\n短段落描述。\n\n这是一个更长的段落，包含大量关于 AADDCC 的说明，以及它的副作用和应用场景，以确保内容足够长触发分段逻辑。",
	}

	chunks, err := ch.Chunk(context.Background(), doc)
	if err != nil {
		t.Fatalf("chunk error: %v", err)
	}
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks))
	}

	first := chunks[0]
	if section, ok := first.Metadata["section"]; !ok || section != "title" {
		t.Fatalf("expected first chunk section=title, got %#v", first.Metadata["section"])
	}
	if !strings.Contains(first.Content, "短段落") {
		t.Fatalf("expected merged short paragraph in first chunk, got %q", first.Content)
	}

	second := chunks[1]
	if section, ok := second.Metadata["section"]; !ok || section != "body" {
		t.Fatalf("expected second chunk section=body, got %#v", second.Metadata["section"])
	}
	if !strings.Contains(second.Content, "副作用") {
		t.Fatalf("expected longer body content in second chunk, got %q", second.Content)
	}
}
