package tiktoken

import (
	"github.com/pkoukk/tiktoken-go"
)

type Tokenizer struct {
	enc *tiktoken.Tiktoken
}

func NewTiktokenTokenizer(name string) (*Tokenizer, error) {
	enc, err := tiktoken.EncodingForModel(name)
	if err != nil {
		// try by name
		enc, err = tiktoken.GetEncoding(name)
		if err != nil {
			return nil, err
		}
	}
	return &Tokenizer{enc: enc}, nil
}

func (t *Tokenizer) Encode(text string) []int {
	return t.enc.Encode(text, nil, nil)
}

func (t *Tokenizer) CountTokens(text string) int {
	count := 0
	for _, v := range t.Encode(text) {
		count += v
	}
	return count
}

// GetTextSlice returns substring that corresponds to token window (approx via decoding)
func (t *Tokenizer) DecodeIds(ids []int) string {
	return t.enc.Decode(ids)
}
