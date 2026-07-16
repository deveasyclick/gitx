package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/user/gitx/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	if cfg.AI.Provider != "openai" {
		t.Errorf("default provider = %q, want %q", cfg.AI.Provider, "openai")
	}
	if cfg.AI.Model != "gpt-5-mini" {
		t.Errorf("default model = %q, want %q", cfg.AI.Model, "gpt-5-mini")
	}
	if cfg.Commit.Style != "conventional" {
		t.Errorf("default style = %q, want %q", cfg.Commit.Style, "conventional")
	}
}

func TestMerge(t *testing.T) {
	base := config.DefaultConfig()
	base.AI.Provider = "anthropic"
	base.AI.Model = "claude-sonnet-4-5"

	override := &config.Config{}
	override.AI.Provider = "ollama"

	base.Merge(override)

	if base.AI.Provider != "ollama" {
		t.Errorf("provider after merge = %q, want %q", base.AI.Provider, "ollama")
	}
	if base.AI.Model != "claude-sonnet-4-5" {
		t.Errorf("model should be unchanged = %q, want %q", base.AI.Model, "claude-sonnet-4-5")
	}
}

func TestMergeNil(t *testing.T) {
	base := config.DefaultConfig()
	base.Merge(nil) // should not panic
	if base.AI.Provider != "openai" {
		t.Errorf("provider after nil merge = %q", base.AI.Provider)
	}
}

func TestMergeEmptyFields(t *testing.T) {
	base := config.DefaultConfig()
	base.AI.Provider = "anthropic"

	override := &config.Config{}
	base.Merge(override)

	if base.AI.Provider != "anthropic" {
		t.Errorf("provider should not be overwritten by empty = %q", base.AI.Provider)
	}
}

func TestMergeAPIKey(t *testing.T) {
	base := config.DefaultConfig()
	base.AI.APIKey = "sk-old-key"

	override := &config.Config{}
	override.AI.APIKey = "sk-new-key"

	base.Merge(override)
	if base.AI.APIKey != "sk-new-key" {
		t.Errorf("api_key after merge = %q, want %q", base.AI.APIKey, "sk-new-key")
	}

	// Empty should not overwrite
	override2 := &config.Config{}
	base.Merge(override2)
	if base.AI.APIKey != "sk-new-key" {
		t.Errorf("api_key should not be overwritten by empty = %q", base.AI.APIKey)
	}
}

func TestLoadDefaultsWhenNoFiles(t *testing.T) {
	// Temporarily set HOME to an isolated dir with no config
	home := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", home)
	defer os.Setenv("HOME", oldHome)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.AI.Provider != "openai" {
		t.Errorf("provider = %q, want %q", cfg.AI.Provider, "openai")
	}
	if cfg.AI.Model != "gpt-5-mini" {
		t.Errorf("model = %q, want %q", cfg.AI.Model, "gpt-5-mini")
	}
	if cfg.Commit.Style != "conventional" {
		t.Errorf("style = %q, want %q", cfg.Commit.Style, "conventional")
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	home := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", home)
	defer os.Setenv("HOME", oldHome)

	saved := &config.Config{}
	saved.AI.Provider = "anthropic"
	saved.AI.Model = "claude-sonnet-4-5"
	saved.AI.APIKey = "sk-ant-test-key-12345"
	saved.Commit.Style = "gitmoji"

	if err := config.Save(saved); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	globalPath := filepath.Join(home, ".config", "gitx", "config.yaml")
	if _, err := os.Stat(globalPath); os.IsNotExist(err) {
		t.Fatalf("config file not written to %s", globalPath)
	}

	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.AI.Provider != "anthropic" {
		t.Errorf("provider = %q, want %q", loaded.AI.Provider, "anthropic")
	}
	if loaded.AI.Model != "claude-sonnet-4-5" {
		t.Errorf("model = %q, want %q", loaded.AI.Model, "claude-sonnet-4-5")
	}
	if loaded.AI.APIKey != "sk-ant-test-key-12345" {
		t.Errorf("api_key = %q, want %q", loaded.AI.APIKey, "sk-ant-test-key-12345")
	}
	if loaded.Commit.Style != "gitmoji" {
		t.Errorf("style = %q, want %q", loaded.Commit.Style, "gitmoji")
	}
}


func TestKnownModels(t *testing.T) {
	tests := []struct {
		provider string
		want     int // expected number of known models
	}{
		{"openai", 6},
		{"deepseek", 2},
		{"anthropic", 3},
		{"google", 2},
		{"openrouter", 3},
		{"ollama", 3},
		{"unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			models := config.KnownModels(tt.provider)
			if len(models) != tt.want {
				t.Errorf("KnownModels(%q) = %d models, want %d", tt.provider, len(models), tt.want)
			}
		})
	}
}
