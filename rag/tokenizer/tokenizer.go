package tokenizer

import (
	"strings"
	"unicode"
)

type Tokenizer interface {
	Encode(text string) []int
	CountTokens(text string) int
	// GetTextSlice returns substring that corresponds to token window (approx via decoding)
	DecodeIds(ids []int) string
}

var _ Tokenizer = (*SimpleTokenizer)(nil)

type SimpleTokenizer struct {
	vocab    map[string]int // token → id
	invVocab map[int]string // id → token
	nextID   int
}

// NewSimpleTokenizer creates new tokenizer with empty vocab.
func NewSimpleTokenizer() Tokenizer {
	return &SimpleTokenizer{
		vocab:    make(map[string]int),
		invVocab: make(map[int]string),
		nextID:   1, // reserve 0 for padding if needed
	}
}

// addToken registers token to vocab if not exists
func (t *SimpleTokenizer) addToken(tok string) int {
	if id, ok := t.vocab[tok]; ok {
		return id
	}
	id := t.nextID
	t.vocab[tok] = id
	t.invVocab[id] = tok
	t.nextID++
	return id
}

// ------------------------------------------------------------------
// Tokenization rules:
// - English letters → continuous word
// - Numbers → continuous number
// - Chinese characters → single rune
// - Punctuation → standalone token
// ------------------------------------------------------------------

func (t *SimpleTokenizer) splitTokens(s string) []string {
	var toks []string
	var buf strings.Builder

	flush := func() {
		if buf.Len() > 0 {
			toks = append(toks, buf.String())
			buf.Reset()
		}
	}

	for _, r := range s {
		switch {
		case unicode.IsSpace(r):
			flush()

		case unicode.Is(unicode.Han, r):
			flush()
			toks = append(toks, string(r))

		case unicode.IsLetter(r) || unicode.IsDigit(r):
			buf.WriteRune(r)

		default:
			flush()
			toks = append(toks, string(r))
		}
	}

	flush()
	return toks
}

// ------------------------------------------------------------------
// Encode
// ------------------------------------------------------------------

func (t *SimpleTokenizer) Encode(text string) []int {
	toks := t.splitTokens(text)
	ids := make([]int, 0, len(toks))
	for _, tok := range toks {
		id := t.addToken(tok)
		ids = append(ids, id)
	}
	return ids
}

func (t *SimpleTokenizer) CountTokens(text string) int {
	return len(t.splitTokens(text))
}

func (t *SimpleTokenizer) DecodeIds(ids []int) string {
	var sb strings.Builder
	for _, id := range ids {
		if tok, ok := t.invVocab[id]; ok {
			sb.WriteString(tok)
		}
	}
	return sb.String()
}
