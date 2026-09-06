package utils

import (
	"strings"
	"unicode"
)

// SanitizeOpenAIInputText removes invalid/control whitespace noise while
// preserving human-readable content for model prompts.
func SanitizeOpenAIInputText(text string) string {
	if text == "" {
		return ""
	}

	normalized := strings.ToValidUTF8(text, " ")
	var builder strings.Builder
	builder.Grow(len(normalized))

	lastSpace := false
	for _, r := range normalized {
		switch {
		case r == '\u0000':
			continue
		case unicode.IsControl(r) || unicode.IsSpace(r):
			if !lastSpace {
				builder.WriteByte(' ')
				lastSpace = true
			}
		default:
			builder.WriteRune(r)
			lastSpace = false
		}
	}

	return strings.TrimSpace(builder.String())
}

// StripMarkdownCodeFence unwraps a fenced markdown block when the whole value
// is wrapped in triple backticks.
func StripMarkdownCodeFence(text string) string {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "```") {
		return trimmed
	}

	trimmed = strings.TrimPrefix(trimmed, "```")
	if newline := strings.Index(trimmed, "\n"); newline >= 0 {
		trimmed = trimmed[newline+1:]
	}
	if idx := strings.LastIndex(trimmed, "```"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	return strings.TrimSpace(trimmed)
}

func NonEmptyStrings(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func HasText(value string) bool {
	return strings.TrimSpace(value) != ""
}

func TrimLower(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func UniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func TruncateText(value string, maxRunes int) string {
	value = strings.TrimSpace(value)
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes]) + "..."
}

func FirstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
