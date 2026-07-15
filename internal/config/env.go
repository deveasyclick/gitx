package config

import (
	"os"
	"strings"
)

// EnvAPIKeyVar returns the environment variable name for a provider's API key.
// Example: EnvAPIKeyVar("openai") → "GITX_OPENAI_API_KEY"
func EnvAPIKeyVar(provider string) string {
	return "GITX_" + strings.ToUpper(provider) + "_API_KEY"
}

// ResolveAPIKey looks up the API key for the given provider from the environment.
// Returns the key value, or empty string if not set.
func ResolveAPIKey(provider string) string {
	return os.Getenv(EnvAPIKeyVar(provider))
}

// HasAPIKey reports whether an API key is configured for the given provider.
func HasAPIKey(provider string) bool {
	return ResolveAPIKey(provider) != ""
}

// OllamaURL returns the Ollama base URL from GITX_OLLAMA_URL, or the default.
func OllamaURL() string {
	if v := os.Getenv("GITX_OLLAMA_URL"); v != "" {
		return v
	}
	return "http://localhost:11434"
}

// LogLevel returns the configured log level from GITX_LOG_LEVEL.
// Returns empty string if not set.
func LogLevel() string {
	return os.Getenv("GITX_LOG_LEVEL")
}
