package ai_test

import (
	"testing"

	"github.com/user/gitx/internal/ai"
	"github.com/user/gitx/internal/config"
)

func TestNewProvider_OpenAI(t *testing.T) {
	p, err := ai.NewProvider(config.AIConfig{
		Provider: "openai",
		Model:    "gpt-5-mini",
		APIKey:   "sk-test-123",
	})
	if err != nil {
		t.Fatalf("NewProvider(openai) error = %v", err)
	}
	if p.Name() != "openai" {
		t.Errorf("Name() = %q, want %q", p.Name(), "openai")
	}
}

func TestNewProvider_DeepSeek(t *testing.T) {
	p, err := ai.NewProvider(config.AIConfig{
		Provider: "deepseek",
		Model:    "deepseek-chat",
		APIKey:   "sk-test-456",
	})
	if err != nil {
		t.Fatalf("NewProvider(deepseek) error = %v", err)
	}
	if p.Name() != "deepseek" {
		t.Errorf("Name() = %q, want %q", p.Name(), "deepseek")
	}
}

func TestNewProvider_MissingAPIKey(t *testing.T) {
	_, err := ai.NewProvider(config.AIConfig{
		Provider: "openai",
	})
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestNewProvider_UnknownProvider(t *testing.T) {
	_, err := ai.NewProvider(config.AIConfig{
		Provider: "anthropic",
		APIKey:   "sk-ant-test",
	})
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestNewProvider_DefaultModel(t *testing.T) {
	p, err := ai.NewProvider(config.AIConfig{
		Provider: "openai",
		APIKey:   "sk-test-123",
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
