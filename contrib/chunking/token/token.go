package token

import (
	"context"
	"regexp"
	"strings"

	"github.com/sweetpotato0/ai-allin/rag/document"
)

var tokenRegex = regexp.MustCompile(`\p{L}[\p{L}\p{M}]*|\p{N}+|[^\s]`)

// Chunker approximates token-aware chunking without depending on provider-specific codecs.
// It keeps whitespace intact while enforcing token windows and overlaps.
type Chunker struct {
	maxTokens     int
	overlapTokens int
}

// Option customises the token chunker.
type Option func(*Chunker)

// WithMaxTokens sets the maximum allowed tokens per chunk (default 256).
func WithMaxTokens(tokens int) Option {
	return func(c *Chunker) {
		if tokens > 0 {
			c.maxTokens = tokens
		}
	}
}

// WithOverlapTokens sets how many tokens are shared between consecutive chunks.
func WithOverlapTokens(tokens int) Option {
	return func(c *Chunker) {
		if tokens >= 0 {
			c.overlapTokens = tokens
		}
	}
}

// New creates a new token-aware chunker.
func New(opts ...Option) *Chunker {
	ch := &Chunker{
		maxTokens:     256,
		overlapTokens: 32,
	}
	for _, opt := range opts {
		opt(ch)
	}
	return ch
}

type segment struct {
	start      int
	end        int
	counts     bool
	tokenIndex int // -1 for whitespace segments
}

// Chunk implements chunking.Chunker.
func (c *Chunker) Chunk(ctx context.Context, doc document.Document) ([]document.Chunk, error) {
	document.EnsureDocumentID(&doc)
	segments, tokenSegments := buildSegments(doc.Content)
	if len(tokenSegments) == 0 {
		return []document.Chunk{
			{
				ID:         document.NextChunkID(doc.ID),
				DocumentID: doc.ID,
				Content:    doc.Content,
			},
		}, nil
	}

	var chunks []document.Chunk
	tokenStart := 0
	for tokenStart < len(tokenSegments) {
		tokenEnd := tokenStart + c.maxTokens
		if tokenEnd > len(tokenSegments) {
			tokenEnd = len(tokenSegments)
		}
		startSegment := tokenSegments[tokenStart]
		if startSegment > 0 && !segments[startSegment-1].counts {
			startSegment--
		}
		endSegment := tokenSegments[tokenEnd-1]
		endSegment++
		for endSegment < len(segments) && !segments[endSegment].counts {
			endSegment++
		}

		chunkText := extract(doc.Content, segments[startSegment:endSegment])
		chunks = append(chunks, document.Chunk{
			ID:         document.NextChunkID(doc.ID),
			DocumentID: doc.ID,
			Content:    chunkText,
		})

		if tokenEnd == len(tokenSegments) {
			break
		}
		tokenStart = tokenEnd - c.overlapTokens
		if tokenStart < 0 {
			tokenStart = 0
		}
	}

	return chunks, nil
}

func buildSegments(text string) ([]segment, []int) {
	var segments []segment
	var tokenSegments []int
	matches := tokenRegex.FindAllStringIndex(text, -1)
	prevEnd := 0
	tokenIndex := 0
	for _, loc := range matches {
		if loc[0] > prevEnd {
			segments = append(segments, segment{
				start:      prevEnd,
				end:        loc[0],
				counts:     false,
				tokenIndex: -1,
			})
		}
		segments = append(segments, segment{
			start:      loc[0],
			end:        loc[1],
			counts:     true,
			tokenIndex: tokenIndex,
		})
		tokenSegments = append(tokenSegments, len(segments)-1)
		tokenIndex++
		prevEnd = loc[1]
	}
	if prevEnd < len(text) {
		segments = append(segments, segment{
			start:      prevEnd,
			end:        len(text),
			counts:     false,
			tokenIndex: -1,
		})
	}
	return segments, tokenSegments
}

func extract(content string, segments []segment) string {
	var b strings.Builder
	for _, seg := range segments {
		b.WriteString(content[seg.start:seg.end])
	}
	return b.String()
}
