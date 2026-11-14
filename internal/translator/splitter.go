package translator

import (
	"regexp"
	"strings"
)

var (
	blankLineRe  = regexp.MustCompile(`\n\s*\n+`)
	multiSpaceRe = regexp.MustCompile(`[ \t]+`)
)

// SplitParagraphs normalizes raw text and segments it into logical paragraphs.
// It prefers splitting on blank lines (\n\n+), and falls back to single newline
// splits if the input uses single-line paragraph breaks.
func SplitParagraphs(raw string) []string {
	normalized := strings.ReplaceAll(raw, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	normalized = strings.TrimSpace(normalized)
	if normalized == "" {
		return nil
	}

	rawBlocks := blankLineRe.Split(normalized, -1)
	if len(rawBlocks) == 1 && strings.Count(normalized, "\n") >= 1 {
		// Input has newline breaks but no empty lines; split per non-empty line.
		rawBlocks = strings.Split(normalized, "\n")
	}

	paragraphs := make([]string, 0, len(rawBlocks))
	for _, block := range rawBlocks {
		// Unwrap soft line breaks inside a paragraph block into spaces.
		lines := strings.Split(block, "\n")
		cleanedLines := make([]string, 0, len(lines))
		for _, l := range lines {
			trimmed := strings.TrimSpace(l)
			if trimmed != "" {
				cleanedLines = append(cleanedLines, trimmed)
			}
		}
		joined := strings.Join(cleanedLines, " ")
		joined = multiSpaceRe.ReplaceAllString(joined, " ")
		joined = strings.TrimSpace(joined)
		if joined != "" {
			paragraphs = append(paragraphs, joined)
		}
	}
	return paragraphs
}
