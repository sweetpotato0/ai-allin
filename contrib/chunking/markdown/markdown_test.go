package markdown

import (
	"context"
	"testing"

	"github.com/sweetpotato0/ai-allin/rag/document"
)

func TestMarkdownChunkerSplitsByHeadings(t *testing.T) {
	ch := New(WithMaxHeadingLevel(2), WithMaxCharacters(200), WithMinCharacters(0))
	doc := document.Document{
		ID: "doc-1",
		Content: `
# AADDCC

AADDCC 是一种万能药物。

## 副作用

吃多了会精神异常。
`,
	}

	chunks, err := ch.Chunk(context.Background(), doc)
	if err != nil {
		t.Fatalf("chunk error: %v", err)
	}
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}
	if chunks[0].Metadata["section_title"] == "" {
		t.Fatalf("expected section metadata to be present")
	}
}
