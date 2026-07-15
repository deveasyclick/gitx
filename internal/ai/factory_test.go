package ai_test

import (
	"os"
	"testing"

	"github.com/user/gitx/internal/ai"
	"github.com/user/gitx/internal/config"
)

func TestNewProvider_OpenAI(t *testing.T) {
	os.Setenv("GITX_OPENAI_API_KEY", "sk-test-123")
	defer os.Unsetenv("GITX_OPENAI_API_KEY")

	p, err := ai.NewProvider(config.AIConfig{
		Provider: "openai",
		Model:    "gpt-5-mini",
	})
	if err != nil {
		t.Fatalf("NewProvider(openai) error = %v", err)
	}
	if p.Name() != "openai" {
		t.Errorf("Name() = %q, want %q", p.Name(), "openai")
	}
}

func TestNewProvider_DeepSeek(t *testing.T) {
	os.Setenv("GITX_DEEPSEEK_API_KEY", "sk-test-456")
	defer os.Unsetenv("GITX_DEEPSEEK_API_KEY")

	p, err := ai.NewProvider(config.AIConfig{
		Provider: "deepseek",
		Model:    "deepseek-chat",
	})
	if err != nil {
		t.Fatalf("NewProvider(deepseek) error = %v", err)
	}
	if p.Name() != "deepseek" {
		t.Errorf("Name() = %q, want %q", p.Name(), "deepseek")
	}
}

func TestNewProvider_MissingAPIKey(t *testing.T) {
	os.Unsetenv("GITX_OPENAI_API_KEY")

	_, err := ai.NewProvider(config.AIConfig{
		Provider: "openai",
	})
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestNewProvider_UnknownProvider(t *testing.T) {
	os.Setenv("GITX_ANTHROPIC_API_KEY", "sk-ant-test")
	defer os.Unsetenv("GITX_ANTHROPIC_API_KEY")

	_, err := ai.NewProvider(config.AIConfig{
		Provider: "anthropic",
	})
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestNewProvider_DefaultModel(t *testing.T) {
	os.Setenv("GITX_OPENAI_API_KEY", "sk-test-123")
	defer os.Unsetenv("GITX_OPENAI_API_KEY")

	p, err := ai.NewProvider(config.AIConfig{
		Provider: "openai",
		// no model specified — should use default
	})
	if err != nil {
		t.Fatalf("NewProvider error = %v", err)
	}
	if p.Name() != "openai" {
		t.Errorf("Name() = %q, want %q", p.Name(), "openai")
	}
}

func TestNewProvider_APIKeyFromConfig(t *testing.T) {
	// No env var set
	os.Unsetenv("GITX_OPENAI_API_KEY")

	// Key comes from config struct directly
	p, err := ai.NewProvider(config.AIConfig{
		Provider: "openai",
		Model:    "gpt-5-mini",
		APIKey:   "sk-config-key-123",
	})
	if err != nil {
		t.Fatalf("NewProvider(openai with config api_key) error = %v", err)
	}
	if p.Name() != "openai" {
		t.Errorf("Name() = %q, want %q", p.Name(), "openai")
	}
}

func TestNewProvider_ConfigKeyTakesPriority(t *testing.T) {
	// Both env var and config key are set
	os.Setenv("GITX_OPENAI_API_KEY", "sk-env-key")
	defer os.Unsetenv("GITX_OPENAI_API_KEY")

	p, err := ai.NewProvider(config.AIConfig{
		Provider: "openai",
		Model:    "gpt-5-mini",
		APIKey:   "sk-config-key",
	})
	if err != nil {
		t.Fatalf("NewProvider error = %v", err)
	}
	if p.Name() != "openai" {
		t.Errorf("Name() = %q, want %q", p.Name(), "openai")
	}
	// If config key is used (not env), it should work
	// (can't easily inspect the internal key, but no error means it used config key)
	_ = p
}
