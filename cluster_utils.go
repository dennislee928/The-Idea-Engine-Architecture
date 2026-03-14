package main

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
)

var clusterNonWordPattern = regexp.MustCompile(`[^\p{L}\p{N}\s]+`)

var clusterStopWords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "app": {}, "are": {}, "but": {}, "for": {}, "from": {},
	"get": {}, "how": {}, "into": {}, "not": {}, "that": {}, "the": {}, "their": {}, "there": {},
	"they": {}, "this": {}, "too": {}, "use": {}, "users": {}, "using": {}, "with": {},
	"一直": {}, "有人": {}, "也": {}, "太": {}, "真的": {}, "感覺": {}, "現在": {}, "這樣": {},
	"求救": {}, "請問": {}, "就是": {}, "那個": {}, "好像": {}, "有點": {}, "問題": {},
}

var clusterNoisePhrases = []string{
	"有人也這樣嗎", "請問一下", "真的很", "到底要怎麼", "怎麼辦", "求救", "help me",
	"does anyone else", "anyone else", "i hate", "too many", "so many",
}

func BuildPainCluster(corePainPoint string, matchedKeywords []string) (string, string) {
	label := normalizeClusterLabel(corePainPoint)
	base := strings.ToLower(label)
	base = clusterNonWordPattern.ReplaceAllString(base, " ")
	for _, phrase := range clusterNoisePhrases {
		base = strings.ReplaceAll(base, phrase, " ")
	}
	base = whitespacePattern.ReplaceAllString(base, " ")
	base = strings.TrimSpace(base)

	var tokens []string
	if containsHan(base) {
		tokens = append(tokens, chineseClusterTokens(base)...)
	}
	tokens = append(tokens, englishClusterTokens(base)...)
	tokens = append(tokens, keywordClusterTokens(matchedKeywords)...)
	tokens = uniqueStrings(tokens)

	key := strings.Join(tokens, "::")
	if key == "" {
		key = compactClusterFallback(base, matchedKeywords)
	}
	if key == "" {
		key = "uncategorized"
	}
	if label == "" {
		label = humanizeClusterKey(key)
	}
	return key, label
}

func normalizeClusterLabel(value string) string {
	value = normalizeText(value)
	if value == "" {
		return ""
	}

	runes := []rune(value)
	if len(runes) > 120 {
		return strings.TrimSpace(string(runes[:120])) + "..."
	}
	return value
}

func englishClusterTokens(text string) []string {
	parts := strings.Fields(text)
	var tokens []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, exists := clusterStopWords[part]; exists {
			continue
		}
		if utfLen(part) <= 2 {
			continue
		}
		tokens = append(tokens, part)
	}
	sort.Strings(tokens)
	if len(tokens) > 5 {
		tokens = tokens[:5]
	}
	return tokens
}

func chineseClusterTokens(text string) []string {
	compact := strings.ReplaceAll(text, " ", "")
	runes := []rune(compact)
	var tokens []string
	for index := 0; index < len(runes) && len(tokens) < 5; index += 2 {
		end := index + 2
		if end > len(runes) {
			end = len(runes)
		}
		token := strings.TrimSpace(string(runes[index:end]))
		if token == "" {
			continue
		}
		if _, exists := clusterStopWords[token]; exists {
			continue
		}
		tokens = append(tokens, token)
	}
	return tokens
}

func keywordClusterTokens(keywords []string) []string {
	var tokens []string
	for _, keyword := range keywords {
		keyword = strings.ToLower(normalizeText(keyword))
		keyword = clusterNonWordPattern.ReplaceAllString(keyword, "")
		keyword = strings.TrimSpace(keyword)
		if keyword == "" {
			continue
		}
		tokens = append(tokens, keyword)
		if len(tokens) >= 3 {
			break
		}
	}
	sort.Strings(tokens)
	return tokens
}

func compactClusterFallback(base string, keywords []string) string {
	if len(keywords) > 0 {
		normalized := normalizeText(strings.Join(keywords, " "))
		normalized = strings.ToLower(clusterNonWordPattern.ReplaceAllString(normalized, ""))
		if normalized != "" {
			return normalized
		}
	}

	base = strings.ReplaceAll(base, " ", "")
	runes := []rune(base)
	if len(runes) > 16 {
		runes = runes[:16]
	}
	return string(runes)
}

func humanizeClusterKey(key string) string {
	if key == "" {
		return "Recurring pain point"
	}
	return strings.ReplaceAll(key, "::", " / ")
}

func uniqueStrings(values []string) []string {
	var result []string
	seen := make(map[string]struct{})
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func containsHan(value string) bool {
	for _, r := range value {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func utfLen(value string) int {
	return len([]rune(value))
}
