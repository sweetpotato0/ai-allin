package document

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"time"
)

// Document represents a knowledge source that can be chunked and indexed.
type Document struct {
	ID       string         `json:"id"`
	Title    string         `json:"title,omitempty"`
	Content  string         `json:"content"`
	Source   string         `json:"source"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Chunk represents a slice of a document that is indexed into a vector store.
type Chunk struct {
	ID         string         `json:"id"`
	DocumentID string         `json:"document_id"`
	Content    string         `json:"content"`
	Section    string         `json:"section"`
	StartRune  int            `json:"start_rune"`
	EndRune    int            `json:"end_rune"`
	TokenCount int            `json:"token_count"`
	Ordinal    int            `json:"ordinal"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type Summary struct {
	ChunkID   string   `json:"chunk_id"`
	Summary   string   `json:"summary"`
	KeyPoints []string `json:"key_points"`
}

// GenDocumentID makes sure every document has a stable identifier.
func GenDocumentID(prefix, s string) string {
	prefix = "doc_" + prefix + fmt.Sprintf("%d", time.Now().UnixNano()) + s[:min(40, len(s))]
	h := sha1.Sum([]byte(prefix + s))
	return hex.EncodeToString(h[:])[:16]
}

// GenChunkID returns a globally unique chunk identifier derived from document ID.
func GenChunkID(prefix, docID string) string {
	next := prefix + fmt.Sprintf("%d", time.Now().UnixNano())
	if docID == "" {
		return fmt.Sprintf("chunk_%s", next)
	}
	return fmt.Sprintf("%s_chunk_%s", docID, next)
}

// Clone returns a deep copy of the document.
func (d Document) Clone() Document {
	out := d
	if d.Metadata != nil {
		out.Metadata = make(map[string]any, len(d.Metadata))
		for k, v := range d.Metadata {
			out.Metadata[k] = v
		}
	}
	return out
}

// Clone returns a deep copy of the chunk.
func (c Chunk) Clone() Chunk {
	out := c
	if c.Metadata != nil {
		out.Metadata = make(map[string]any, len(c.Metadata))
		for k, v := range c.Metadata {
			out.Metadata[k] = v
		}
	}
	return out
}
