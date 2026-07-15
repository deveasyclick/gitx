package security

import "regexp"

// patterns is the list of secret patterns scanned before sending data to AI.
// This is an MVP set covering the most common credential leaks.
var patterns = []*regexp.Regexp{
	// OpenAI API keys: sk-...
	regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}`),

	// Anthropic API keys: sk-ant-...
	regexp.MustCompile(`sk-ant-[a-zA-Z0-9]{20,}`),

	// DeepSeek API keys: sk-...
	// Shares the same pattern as OpenAI; dedup handled by scanner.

	// AWS access key IDs: AKIA...
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),

	// Private key blocks
	regexp.MustCompile(`-----BEGIN (RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`),

	// GitHub personal access tokens: ghp_, gho_, github_pat_
	regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}`),
	regexp.MustCompile(`gho_[a-zA-Z0-9]{36}`),
	regexp.MustCompile(`github_pat_[a-zA-Z0-9_]{82}`),

	// Generic bearer tokens in headers
	regexp.MustCompile(`(?i)(authorization|bearer)\s+(sk-|ghp_|gho_|github_pat_)[a-zA-Z0-9_]+`),
}
