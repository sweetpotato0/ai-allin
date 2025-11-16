package preprocess

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/PuerkitoBio/goquery"
)

// CleanBasic: space、OCR common error and space line.
func CleanBasic(text string) string {
	if text == "" {
		return ""
	}

	// remove control chars except newline
	b := strings.Map(func(r rune) rune {
		if r == '\n' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, text)

	// fix common ligatures / OCR artifacts
	fixes := map[string]string{
		"ﬁ": "fi", "ﬂ": "fl",
		"—": "-", "–": "-",
		"·": ".", "•": "-",
	}
	for k, v := range fixes {
		b = strings.ReplaceAll(b, k, v)
	}

	// collapse spaces & tabs
	reSpaces := regexp.MustCompile(`[ \t]+`)
	b = reSpaces.ReplaceAllString(b, " ")

	// collapse multiple newlines to two
	reNewlines := regexp.MustCompile(`\n{3,}`)
	b = reNewlines.ReplaceAllString(b, "\n\n")

	return strings.TrimSpace(b)
}

// HTMLToText: lightweight extraction of content, keep headings and paragraphs
func HTMLToText(html string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", err
	}
	var out []string
	doc.Find("h1,h2,h3,h4,p,li,pre,code,table").Each(func(i int, s *goquery.Selection) {
		switch goquery.NodeName(s) {
		case "h1":
			out = append(out, "# "+strings.TrimSpace(s.Text()))
		case "h2":
			out = append(out, "## "+strings.TrimSpace(s.Text()))
		case "h3":
			out = append(out, "### "+strings.TrimSpace(s.Text()))
		case "p":
			out = append(out, strings.TrimSpace(s.Text()))
		case "li":
			out = append(out, "- "+strings.TrimSpace(s.Text()))
		case "pre", "code":
			out = append(out, "```\n"+strings.TrimSpace(s.Text())+"\n```")
		case "table":
			out = append(out, parseTable(s))
		}
	})
	return strings.Join(out, "\n\n"), nil
}

func parseTable(sel *goquery.Selection) string {
	var rows []string
	sel.Find("tr").Each(func(i int, tr *goquery.Selection) {
		var cols []string
		tr.Find("th,td").Each(func(j int, td *goquery.Selection) {
			cols = append(cols, strings.TrimSpace(td.Text()))
		})
		if len(cols) > 0 {
			rows = append(rows, "| "+strings.Join(cols, " | ")+" |")
		}
	})
	return strings.Join(rows, "\n")
}

// RemoveDuplicateParagraphs dedupe by exact paragraph text
func RemoveDuplicateParagraphs(text string) string {
	parts := strings.Split(text, "\n\n")
	seen := map[string]struct{}{}
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return strings.Join(out, "\n\n")
}

// Preprocess: pipeline
func Preprocess(raw string) string {
	t := CleanBasic(raw)
	t = RemoveWebNoise(t)
	t = RemoveDuplicateParagraphs(t)
	return t
}

// RemoveWebNoise: simple pattern-based noise removal (can be extended)
func RemoveWebNoise(s string) string {
	patterns := []string{
		"相关链接", "你可能还喜欢", "热门文章", "版权", "版权所有", "Cookie", "隐私政策", "广告",
	}
	lines := strings.Split(s, "\n")
	var out []string
	for _, l := range lines {
		skip := false
		for _, p := range patterns {
			if strings.Contains(l, p) {
				skip = true
				break
			}
		}
		if !skip {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}
