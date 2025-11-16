package chunking

import (
	"context"
	"strings"

	"github.com/sweetpotato0/ai-allin/rag/document"
	"github.com/sweetpotato0/ai-allin/rag/tokenizer"
)

// Chunker splits documents into chunks that can be embedded and indexed.
type Chunker interface {
	Chunk(ctx context.Context, doc document.Document) ([]document.Chunk, error)
}

type Options struct {
	Overlap   int
	MaxTokens int
	Tokenizer tokenizer.Tokenizer
}

var _ Chunker = (*SimpleChunker)(nil)

// SimpleChunker splits documents by separator and enforces max character lengths.
type SimpleChunker struct {
	overlap   int
	maxTokens int

	tk tokenizer.Tokenizer
}

// Option customizes the simple chunker.
type Option func(*Options)

// WithMaxToken overrides the default chunk size (characters).
func WithMaxToken(maxToken int) Option {
	return func(o *Options) {
		if maxToken > 0 {
			o.MaxTokens = maxToken
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

// WithTokenizer set tokenizer.
func WithTokenizer(t tokenizer.Tokenizer) Option {
	return func(o *Options) {
		if t != nil {
			o.Tokenizer = t
		}
	}
}

// NewSimpleChunker constructs a chunker with sane defaults for most knowledge bases.
func NewSimpleChunker(opts ...Option) *SimpleChunker {
	cfg := &Options{
		Overlap:   120,
		MaxTokens: 1024,
		Tokenizer: tokenizer.NewSimpleTokenizer(),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return &SimpleChunker{
		overlap:   cfg.Overlap,
		maxTokens: cfg.MaxTokens,
		tk:        cfg.Tokenizer,
	}
}

// Chunk splits the document into bounded pieces.
func (c *SimpleChunker) Chunk(ctx context.Context, doc document.Document) ([]document.Chunk, error) {
	text := strings.ReplaceAll(doc.Content, "\r\n", "\n")
	// split by headings
	parts := splitByHeadings(text)
	var chunks []document.Chunk
	for _, p := range parts {
		paras := paragraphSplit(p.SectionText)
		for _, para := range paras {
			para = strings.TrimSpace(para)
			if para == "" {
				continue
			}
			// estimate tokens
			tokCount := c.tk.CountTokens(para)
			if tokCount <= c.maxTokens {
				chunks = append(chunks, document.Chunk{
					ID:         document.GenChunkID("", doc.ID),
					DocumentID: doc.ID,
					Section:    p.Heading,
					Content:    para,
					TokenCount: tokCount,
				})
			} else {
				// split para by token windows
				sub := c.splitLargeText(doc.ID, para)
				for _, s := range sub {
					s.Section = p.Heading
					chunks = append(chunks, s)
				}
			}
		}
	}
	// sliding window merge adjacent small chunks to reach min size
	chunks = mergeTinyChunks(chunks, 40)
	return chunks, nil
}

type headingPart struct {
	Heading     string
	SectionText string
}

func splitByHeadings(text string) []headingPart {
	lines := strings.Split(text, "\n")
	cur := headingPart{Heading: "root", SectionText: ""}
	var out []headingPart
	for _, l := range lines {
		trim := strings.TrimSpace(l)
		if strings.HasPrefix(trim, "#") {
			if cur.SectionText != "" {
				out = append(out, cur)
			}
			title := strings.TrimSpace(strings.TrimLeft(trim, "#"))
			if title == "" {
				title = "heading"
			}
			cur = headingPart{Heading: title, SectionText: ""}
			continue
		}
		cur.SectionText += l + "\n"
	}
	if cur.SectionText != "" {
		out = append(out, cur)
	}
	return out
}

func paragraphSplit(text string) []string {
	// split by two newlines
	parts := strings.Split(text, "\n\n")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (c *SimpleChunker) splitLargeText(docID, text string) []document.Chunk {
	// token-level split using tokenizer
	ids := c.tk.Encode(text)
	w := c.maxTokens
	o := c.overlap
	var out []document.Chunk
	for i := 0; i < len(ids); i += (w - o) {
		end := i + w
		if end > len(ids) {
			end = len(ids)
		}
		segIDs := ids[i:end]
		segText := c.tk.DecodeIds(segIDs)
		out = append(out, document.Chunk{
			ID:         document.GenDocumentID(docID, segText),
			DocumentID: docID,
			Content:    segText,
			TokenCount: len(segIDs),
		})
		if end == len(ids) {
			break
		}
	}
	return out
}

func mergeTinyChunks(chunks []document.Chunk, minTokens int) []document.Chunk {
	if len(chunks) == 0 {
		return chunks
	}
	var out []document.Chunk
	for i := 0; i < len(chunks); i++ {
		c := chunks[i]
		if c.TokenCount >= minTokens {
			out = append(out, c)
			continue
		}
		if len(out) > 0 {
			prev := &out[len(out)-1]
			prev.Content = prev.Content + "\n\n" + c.Content
			prev.TokenCount = prev.TokenCount + c.TokenCount
		} else if i+1 < len(chunks) {
			// merge with next
			chunks[i+1].Content = c.Content + "\n\n" + chunks[i+1].Content
			chunks[i+1].TokenCount += c.TokenCount
		} else {
			out = append(out, c)
		}
	}
	return out
}
