package document

import (
	"fmt"
	"sync/atomic"
)

// Document represents a knowledge source that can be chunked and indexed.
type Document struct {
	ID       string         `json:"id"`
	Title    string         `json:"title,omitempty"`
	Content  string         `json:"content"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Chunk represents a slice of a document that is indexed into a vector store.
type Chunk struct {
	ID         string         `json:"id"`
	DocumentID string         `json:"document_id"`
	Content    string         `json:"content"`
	Ordinal    int            `json:"ordinal"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

var (
	docCounter   atomic.Int64
	chunkCounter atomic.Int64
)

// EnsureDocumentID makes sure every document has a stable identifier.
func EnsureDocumentID(doc *Document) {
	if doc == nil {
		return
	}
	if doc.ID != "" {
		return
	}
	id := docCounter.Add(1)
	doc.ID = fmt.Sprintf("doc_%d", id)
}

// NextChunkID returns a globally unique chunk identifier derived from document ID.
func NextChunkID(docID string) string {
	next := chunkCounter.Add(1)
	if docID == "" {
		return fmt.Sprintf("chunk_%d", next)
	}
	return fmt.Sprintf("%s_chunk_%d", docID, next)
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
