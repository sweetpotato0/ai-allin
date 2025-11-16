package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/sweetpotato0/ai-allin/rag/agentic"
	"github.com/sweetpotato0/ai-allin/rag/preprocess"
)

func collectDocuments(docsDir string) ([]agentic.Document, error) {
	var documents []agentic.Document

	walkRoot := filepath.Clean(docsDir)
	if err := filepath.WalkDir(walkRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !isMarkdown(d.Name()) {
			return nil
		}
		doc, err := buildDocument(path, walkRoot)
		if err != nil {
			return err
		}
		documents = append(documents, doc)
		return nil
	}); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	return documents, nil
}

func buildDocument(path, base string) (agentic.Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return agentic.Document{}, fmt.Errorf("read %s: %w", path, err)
	}
	content := strings.ToValidUTF8(string(data), "")
	title := extractTitle(content, filepath.Base(path))
	content = preprocess.Preprocess(content)

	id := filepath.ToSlash(path)
	if base != "" {
		if rel, err := filepath.Rel(base, path); err == nil {
			id = filepath.ToSlash(filepath.Join(filepath.Base(base), rel))
		}
	}

	return agentic.Document{
		ID:      id,
		Title:   title,
		Content: content,
		Metadata: map[string]any{
			"path": path,
		},
	}, nil
}

func isMarkdown(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".mdx")
}

func extractTitle(content, fallback string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = strings.TrimPrefix(line, "#")
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return fallback
}
