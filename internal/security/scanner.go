package security

import "strings"

const redacted = "[REDACTED]"

// Scan checks text for common secret patterns.
// Returns the cleaned text with secrets replaced, and whether anything was found.
func Scan(input string) (cleaned string, found bool) {
	if input == "" {
		return input, false
	}

	cleaned = input
	for _, p := range patterns {
		cleaned = p.ReplaceAllString(cleaned, redacted)
	}

	found = strings.Contains(cleaned, redacted)
	return cleaned, found
}
