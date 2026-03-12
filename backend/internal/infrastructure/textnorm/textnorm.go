package textnorm

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	urlRe     = regexp.MustCompile(`(?i)\b(?:https?://|www\.)\S+\b`)
	multiWSRe = regexp.MustCompile(`\s+`)
)

func ForEmbedding(input string) string {
	s := strings.TrimSpace(input)
	if s == "" {
		return ""
	}

	s = urlRe.ReplaceAllString(s, " ")
	s = strings.Map(func(r rune) rune {
		if unicode.Is(unicode.So, r) || unicode.Is(unicode.Cf, r) {
			return -1
		}
		return r
	}, s)

	s = multiWSRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
