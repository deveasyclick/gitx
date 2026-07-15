package config_test

import (
	"os"
	"testing"

	"github.com/user/gitx/internal/config"
)

func TestEnvAPIKeyVar(t *testing.T) {
	tests := []struct {
		provider string
		want     string
	}{
		{"openai", "GITX_OPENAI_API_KEY"},
		{"anthropic", "GITX_ANTHROPIC_API_KEY"},
		{"openrouter", "GITX_OPENROUTER_API_KEY"},
		{"ollama", "GITX_OLLAMA_API_KEY"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			got := config.EnvAPIKeyVar(tt.provider)
			if got != tt.want {
				t.Errorf("EnvAPIKeyVar(%q) = %q, want %q", tt.provider, got, tt.want)
			}
		})
	}
}

func TestResolveAPIKey(t *testing.T) {
	const key = "sk-test-1234567890"
	os.Setenv("GITX_OPENAI_API_KEY", key)
	defer os.Unsetenv("GITX_OPENAI_API_KEY")

	got := config.ResolveAPIKey("openai")
	if got != key {
		t.Errorf("ResolveAPIKey() = %q, want %q", got, key)
	}
}

func TestResolveAPIKeyMissing(t *testing.T) {
	os.Unsetenv("GITX_OLLAMA_API_KEY")
	got := config.ResolveAPIKey("ollama")
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestHasAPIKey(t *testing.T) {
	os.Setenv("GITX_ANTHROPIC_API_KEY", "sk-ant-test")
	defer os.Unsetenv("GITX_ANTHROPIC_API_KEY")

	if !config.HasAPIKey("anthropic") {
		t.Error("HasAPIKey should be true when env var is set")
	}

	os.Unsetenv("GITX_OPENAI_API_KEY")
	if config.HasAPIKey("openai") {
		t.Error("HasAPIKey should be false when env var is unset")
	}
}

func TestOllamaURLDefault(t *testing.T) {
	os.Unsetenv("GITX_OLLAMA_URL")
	got := config.OllamaURL()
	if got != "http://localhost:11434" {
		t.Errorf("OllamaURL() = %q, want %q", got, "http://localhost:11434")
	}
}

func TestOllamaURLOverride(t *testing.T) {
	os.Setenv("GITX_OLLAMA_URL", "http://ollama.internal:8080")
	defer os.Unsetenv("GITX_OLLAMA_URL")

	got := config.OllamaURL()
	if got != "http://ollama.internal:8080" {
		t.Errorf("OllamaURL() = %q, want %q", got, "http://ollama.internal:8080")
	}
}
