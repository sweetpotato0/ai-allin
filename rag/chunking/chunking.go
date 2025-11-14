package chunking

import (
	"context"
	"strings"

	"github.com/sweetpotato0/ai-allin/rag/document"
)

// Chunker splits documents into chunks that can be embedded and indexed.
type Chunker interface {
	Chunk(ctx context.Context, doc document.Document) ([]document.Chunk, error)
}

type Options struct {
	ChunkSize     int
	Overlap       int
	Separator     string
	IncludeMeta   bool
	MinChunkSize  int
	HeadingPrefix string
	TagSections   bool
}

// SimpleChunker splits documents by separator and enforces max character lengths.
type SimpleChunker struct {
	size          int
	overlap       int
	sep           string
	addMeta       bool
	minSize       int
	headingPrefix string
	tagSections   bool
}

// Option customizes the simple chunker.
type Option func(*Options)

// WithChunkSize overrides the default chunk size (characters).
func WithChunkSize(size int) Option {
	return func(o *Options) {
		if size > 0 {
			o.ChunkSize = size
		}
	}
}

// WithOverlap configures overlap (characters) between consecutive chunks.
func WithOverlap(overlap int) Option {
	return func(o *Options) {
		if overlap >= 0 {
			o.Overlap = overlap
		}
	}
}

// WithSeparator sets the logical separator used before windowing.
func WithSeparator(sep string) Option {
	return func(o *Options) {
		if sep != "" {
			o.Separator = sep
		}
	}
}

// WithMetadataCopy toggles whether document metadata should be copied to chunks.
func WithMetadataCopy(enabled bool) Option {
	return func(o *Options) {
		o.IncludeMeta = enabled
	}
}

// WithMinChunkSize enforces a lower bound for chunk length by merging short segments.
func WithMinChunkSize(size int) Option {
	return func(o *Options) {
		if size > 0 {
			o.MinChunkSize = size
		}
	}
}

// WithHeadingPrefix marks chunks starting with the prefix as titles when section tagging is enabled.
func WithHeadingPrefix(prefix string) Option {
	return func(o *Options) {
		if strings.TrimSpace(prefix) != "" {
			o.HeadingPrefix = prefix
		}
	}
}

// WithSectionTagging enables automatic section metadata on produced chunks.
func WithSectionTagging(enabled bool) Option {
	return func(o *Options) {
		o.TagSections = enabled
	}
}

// NewSimpleChunker constructs a chunker with sane defaults for most knowledge bases.
func NewSimpleChunker(opts ...Option) *SimpleChunker {
	cfg := &Options{
		ChunkSize:     800,
		Overlap:       120,
		Separator:     "\n\n",
		IncludeMeta:   true,
		MinChunkSize:  0,
		HeadingPrefix: "#",
		TagSections:   true,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.ChunkSize < cfg.Overlap {
		cfg.Overlap = cfg.ChunkSize / 3
	}
	return &SimpleChunker{
		size:          cfg.ChunkSize,
		overlap:       cfg.Overlap,
		sep:           cfg.Separator,
		addMeta:       cfg.IncludeMeta,
		minSize:       cfg.MinChunkSize,
		headingPrefix: cfg.HeadingPrefix,
		tagSections:   cfg.TagSections,
	}
}

// Chunk splits the document into bounded pieces.
func (c *SimpleChunker) Chunk(ctx context.Context, doc document.Document) ([]document.Chunk, error) {
	document.EnsureDocumentID(&doc)

	parts := strings.Split(doc.Content, c.sep)
	chunks := make([]document.Chunk, 0, len(parts))
	currentOrdinal := 0
	var buffer string

	for idx, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if c.minSize > 0 {
			if buffer != "" {
				part = buffer + "\n\n" + part
				buffer = ""
			}
			if len(part) < c.minSize && idx < len(parts)-1 {
				buffer = part
				continue
			}
		}
		for len(part) > c.size {
			currentOrdinal++
			window := part[:c.size]
			part = part[c.size-c.overlap:]
			chunks = append(chunks, c.newChunk(doc, currentOrdinal, window))
		}
		currentOrdinal++
		chunks = append(chunks, c.newChunk(doc, currentOrdinal, part))
	}

	if len(chunks) == 0 {
		currentOrdinal++
		chunks = append(chunks, c.newChunk(doc, currentOrdinal, doc.Content))
	} else if buffer != "" {
		currentOrdinal++
		chunks = append(chunks, c.newChunk(doc, currentOrdinal, buffer))
	}

	return chunks, nil
}

func (c *SimpleChunker) newChunk(doc document.Document, ordinal int, content string) document.Chunk {
	text := strings.TrimSpace(content)
	chunk := document.Chunk{
		ID:         document.NextChunkID(doc.ID),
		DocumentID: doc.ID,
		Content:    text,
		Ordinal:    ordinal,
	}
	if c.addMeta && doc.Metadata != nil {
		chunk.Metadata = cloneMetadata(doc.Metadata)
	}
	if c.tagSections {
		if chunk.Metadata == nil {
			chunk.Metadata = make(map[string]any)
		}
		section := "body"
		if strings.HasPrefix(strings.TrimSpace(text), c.headingPrefix) {
			section = "title"
		}
		chunk.Metadata["section"] = section
	}
	return chunk
}

func cloneMetadata(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
