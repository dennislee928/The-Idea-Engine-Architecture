package main

import (
	"regexp"
	"strings"
	"time"
)

var whitespacePattern = regexp.MustCompile(`\s+`)
var htmlPattern = regexp.MustCompile(`<[^>]+>`)

func matchedKeywords(keywords []string, values ...string) []string {
	if len(keywords) == 0 {
		return nil
	}

	joined := strings.ToLower(strings.Join(values, " "))
	var matches []string
	seen := make(map[string]struct{})
	for _, keyword := range keywords {
		keyword = strings.TrimSpace(keyword)
		if keyword == "" {
			continue
		}
		if strings.Contains(joined, strings.ToLower(keyword)) {
			if _, exists := seen[keyword]; !exists {
				seen[keyword] = struct{}{}
				matches = append(matches, keyword)
			}
		}
	}
	return matches
}

func normalizeText(text string) string {
	text = htmlPattern.ReplaceAllString(text, " ")
	text = whitespacePattern.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

func parseFlexibleTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Now().UTC()
	}

	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC()
		}
	}
	return time.Now().UTC()
}
