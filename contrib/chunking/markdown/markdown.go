package markdown

import (
	"context"
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"

	"github.com/sweetpotato0/ai-allin/rag/chunking"
	"github.com/sweetpotato0/ai-allin/rag/document"
)

// Chunker splits markdown documents by heading hierarchy using a goldmark AST.
type Chunker struct {
	maxHeadingLevel int
	maxCharacters   int
	minCharacters   int
	fallback        chunking.Chunker
	parser          goldmark.Markdown
}

// Option customises the markdown chunker.
type Option func(*Chunker)

// WithMaxHeadingLevel caps which heading level starts a new chunk (default 3).
func WithMaxHeadingLevel(level int) Option {
	return func(c *Chunker) {
		if level > 0 {
			c.maxHeadingLevel = level
		}
	}
}

// WithMaxCharacters enforces the upper bound for section payloads before falling back to the base chunker.
func WithMaxCharacters(chars int) Option {
	return func(c *Chunker) {
		if chars > 0 {
			c.maxCharacters = chars
		}
	}
}

// WithMinCharacters merges adjoining sections until they reach the provided size.
func WithMinCharacters(chars int) Option {
	return func(c *Chunker) {
		if chars >= 0 {
			c.minCharacters = chars
		}
	}
}

// WithFallbackChunker swaps the chunker used when a markdown section exceeds the max character limit.
func WithFallbackChunker(ch chunking.Chunker) Option {
	return func(c *Chunker) {
		if ch != nil {
			c.fallback = ch
		}
	}
}

// New creates a production-ready markdown chunker.
func New(opts ...Option) *Chunker {
	ch := &Chunker{
		maxHeadingLevel: 3,
		maxCharacters:   1200,
		minCharacters:   240,
		parser:          goldmark.New(),
		fallback: chunking.NewSimpleChunker(
			chunking.WithChunkSize(800),
			chunking.WithOverlap(120),
		),
	}
	for _, opt := range opts {
		opt(ch)
	}
	return ch
}

// Chunk implements chunking.Chunker.
func (c *Chunker) Chunk(ctx context.Context, doc document.Document) ([]document.Chunk, error) {
	document.EnsureDocumentID(&doc)

	sections := c.splitSections(doc.Content)
	if len(sections) == 0 {
		return c.fallback.Chunk(ctx, doc)
	}

	chunks := make([]document.Chunk, 0, len(sections))
	ordinal := 0
	for _, sec := range sections {
		payload := strings.TrimSpace(sec.raw)
		if payload == "" {
			continue
		}

		if len(payload) <= c.maxCharacters {
			ordinal++
			chunk := document.Chunk{
				ID:         document.NextChunkID(doc.ID),
				DocumentID: doc.ID,
				Content:    payload,
				Ordinal:    ordinal,
				Metadata:   mergeMetadata(doc.Metadata, sec.metadata),
			}
			chunks = append(chunks, chunk)
			continue
		}

		tmpDoc := document.Document{
			ID:       doc.ID,
			Title:    doc.Title,
			Content:  payload,
			Metadata: mergeMetadata(doc.Metadata, sec.metadata),
		}
		splits, err := c.fallback.Chunk(ctx, tmpDoc)
		if err != nil {
			return nil, err
		}
		for _, split := range splits {
			ordinal++
			chunk := split.Clone()
			chunk.ID = document.NextChunkID(doc.ID)
			chunk.DocumentID = doc.ID
			chunk.Ordinal = ordinal
			chunk.Metadata = mergeMetadata(tmpDoc.Metadata, chunk.Metadata)
			chunks = append(chunks, chunk)
		}
	}

	return chunks, nil
}

type markdownSection struct {
	raw      string
	level    int
	title    string
	metadata map[string]any
}

type headingInfo struct {
	start int
	level int
	title string
}

func (c *Chunker) splitSections(content string) []markdownSection {
	source := []byte(content)
	reader := text.NewReader(source)
	root := c.parser.Parser().Parse(reader)

	var headings []headingInfo
	_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		heading, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}
		if heading.Level > c.maxHeadingLevel {
			return ast.WalkContinue, nil
		}
		lines := heading.Lines()
		if lines == nil || lines.Len() == 0 {
			return ast.WalkContinue, nil
		}
		start := lines.At(0).Start
		headings = append(headings, headingInfo{
			start: start,
			level: heading.Level,
			title: strings.TrimSpace(string(heading.Text(source))),
		})
		return ast.WalkSkipChildren, nil
	})

	if len(headings) == 0 {
		raw := strings.TrimSpace(content)
		if raw == "" {
			return nil
		}
		return []markdownSection{{raw: raw}}
	}

	var sections []markdownSection
	if intro := strings.TrimSpace(string(source[:headings[0].start])); intro != "" {
		sections = append(sections, markdownSection{raw: intro})
	}
	for i, h := range headings {
		end := len(source)
		if i+1 < len(headings) {
			end = headings[i+1].start
		}
		raw := strings.TrimSpace(string(source[h.start:end]))
		if raw == "" {
			continue
		}
		meta := map[string]any{
			"section_title": h.title,
			"section_level": h.level,
		}
		sections = append(sections, markdownSection{
			raw:      raw,
			level:    h.level,
			title:    h.title,
			metadata: meta,
		})
	}
	return c.mergeShortSections(sections)
}

func (c *Chunker) mergeShortSections(sections []markdownSection) []markdownSection {
	if c.minCharacters <= 0 || len(sections) == 0 {
		return sections
	}
	merged := make([]markdownSection, 0, len(sections))
	var buffer *markdownSection
	for idx, sec := range sections {
		current := sec
		if buffer != nil {
			current = combineSections(*buffer, sec)
			buffer = nil
		}
		if len(current.raw) < c.minCharacters && idx < len(sections)-1 {
			tmp := current
			buffer = &tmp
			continue
		}
		merged = append(merged, current)
	}
	if buffer != nil {
		merged = append(merged, *buffer)
	}
	return merged
}

func combineSections(a, b markdownSection) markdownSection {
	meta := mergeMetadata(a.metadata, b.metadata)
	raw := strings.TrimSpace(fmt.Sprintf("%s\n\n%s", a.raw, b.raw))
	return markdownSection{
		raw:      raw,
		level:    a.level,
		title:    firstNonEmpty(a.title, b.title),
		metadata: meta,
	}
}

func mergeMetadata(base, extra map[string]any) map[string]any {
	if base == nil && extra == nil {
		return nil
	}
	out := make(map[string]any)
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
