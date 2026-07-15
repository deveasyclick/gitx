package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds all GitX configuration.
type Config struct {
	AI     AIConfig     `mapstructure:"ai"`
	Commit CommitConfig `mapstructure:"commit"`
}

// AIConfig controls AI provider selection.
type AIConfig struct {
	Provider string `mapstructure:"provider"`
	Model    string `mapstructure:"model"`
	APIKey   string `mapstructure:"api_key"`
}

// CommitConfig controls commit message generation.
type CommitConfig struct {
	Style string `mapstructure:"style"`
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		AI: AIConfig{
			Provider: "openai",
			Model:    "gpt-5-mini",
		},
		Commit: CommitConfig{
			Style: "conventional",
		},
	}
}

// KnownModels returns the list of known models for a given provider.
func KnownModels(provider string) []string {
	switch provider {
	case "openai":
		return []string{
			"gpt-5-mini",
			"gpt-5",
			"gpt-4o",
			"gpt-4o-mini",
			"o3",
			"o4-mini",
		}
	case "deepseek":
		return []string{
			"deepseek-v4-flash",
			"deepseek-v4-pro",
		}
	case "anthropic":
		return []string{
			"claude-sonnet-4-5",
			"claude-haiku-3-5",
			"claude-opus-4",
		}
	case "google":
		return []string{
			"gemini-2.5-pro",
			"gemini-2.5-flash",
		}
	case "openrouter":
		return []string{
			"openrouter/auto",
			"anthropic/claude-sonnet-4-5",
			"openai/gpt-5",
		}
	case "ollama":
		return []string{
			"llama3",
			"mistral",
			"codellama",
		}
	default:
		return nil
	}
}

// configPath returns the path to the global config file.
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home directory: %w", err)
	}
	return filepath.Join(home, ".config", "gitx", "config.yaml"), nil
}

// Load reads the global config file (~/.config/gitx/config.yaml) if it exists,
// then applies environment variable overrides. Returns the merged config.
func Load() (*Config, error) {
	cfg := DefaultConfig()

	path, err := configPath()
	if err != nil {
		return nil, fmt.Errorf("config path: %w", err)
	}

	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		// File doesn't exist — use defaults
	} else {
		fileCfg := new(Config)
		if err := v.Unmarshal(fileCfg); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		cfg.Merge(fileCfg)
	}

	// Environment variable overrides
	applyEnvOverrides(cfg)

	return cfg, nil
}

// Merge applies non-zero fields from other into cfg.
func (c *Config) Merge(other *Config) {
	if other == nil {
		return
	}
	if other.AI.Provider != "" {
		c.AI.Provider = other.AI.Provider
	}
	if other.AI.Model != "" {
		c.AI.Model = other.AI.Model
	}
	if other.AI.APIKey != "" {
		c.AI.APIKey = other.AI.APIKey
	}
	if other.Commit.Style != "" {
		c.Commit.Style = other.Commit.Style
	}
}

// applyEnvOverrides checks GITX_PROVIDER and GITX_MODEL environment variables.
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("GITX_PROVIDER"); v != "" {
		cfg.AI.Provider = v
	}
	if v := os.Getenv("GITX_MODEL"); v != "" {
		cfg.AI.Model = v
	}
}

// Save writes cfg to ~/.config/gitx/config.yaml, creating parent directories as needed.
func Save(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config directory %s: %w", dir, err)
	}

	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	v.Set("ai.provider", cfg.AI.Provider)
	v.Set("ai.model", cfg.AI.Model)
	v.Set("ai.api_key", cfg.AI.APIKey)
	v.Set("commit.style", cfg.Commit.Style)

	if err := v.WriteConfig(); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}
