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
	ChunkSize   int
	Overlap     int
	Separator   string
	IncludeMeta bool
}

// SimpleChunker splits documents by separator and enforces max character lengths.
type SimpleChunker struct {
	size    int
	overlap int
	sep     string
	addMeta bool
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

// NewSimpleChunker constructs a chunker with sane defaults for most knowledge bases.
func NewSimpleChunker(opts ...Option) *SimpleChunker {
	cfg := &Options{
		ChunkSize:   800,
		Overlap:     120,
		Separator:   "\n\n",
		IncludeMeta: true,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return &SimpleChunker{
		size:    cfg.ChunkSize,
		overlap: cfg.Overlap,
		sep:     cfg.Separator,
		addMeta: cfg.IncludeMeta,
	}
}

// Chunk splits the document into bounded pieces.
func (c *SimpleChunker) Chunk(ctx context.Context, doc document.Document) ([]document.Chunk, error) {
	document.EnsureDocumentID(&doc)

	parts := strings.Split(doc.Content, c.sep)
	chunks := make([]document.Chunk, 0, len(parts))
	currentOrdinal := 0

	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
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
	}

	return chunks, nil
}

func (c *SimpleChunker) newChunk(doc document.Document, ordinal int, content string) document.Chunk {
	chunk := document.Chunk{
		ID:         document.NextChunkID(doc.ID),
		DocumentID: doc.ID,
		Content:    strings.TrimSpace(content),
		Ordinal:    ordinal,
	}
	if c.addMeta && doc.Metadata != nil {
		chunk.Metadata = make(map[string]any, len(doc.Metadata))
		for k, v := range doc.Metadata {
			chunk.Metadata[k] = v
		}
	}
	return chunk
}
