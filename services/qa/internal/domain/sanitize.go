package domain

import (
	"strings"
	"unicode/utf8"
)

const maxDetailLength = 500

// blockedDetailSubstrings marks content that must never be exposed as user-facing step detail.
var blockedDetailSubstrings = []string{
	"system prompt",
	"chain of thought",
	"思维链",
	"internal reasoning",
	"you are a",
	"作为助手",
}

// SanitizeStepDetail converts raw pipeline output into a safe business summary.
// Empty input is allowed; unsafe or overly long content is rejected.
func SanitizeStepDetail(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	lower := strings.ToLower(trimmed)
	for _, blocked := range blockedDetailSubstrings {
		if strings.Contains(lower, strings.ToLower(blocked)) {
			return ""
		}
	}

	if utf8.RuneCountInString(trimmed) > maxDetailLength {
		runes := []rune(trimmed)
		return string(runes[:maxDetailLength]) + "…"
	}

	return trimmed
}
