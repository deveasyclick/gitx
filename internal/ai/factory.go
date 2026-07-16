package ai

import (
	"fmt"

	"github.com/user/gitx/internal/config"
)

// NewProvider creates a provider based on the configuration.
// API keys must be set in the config file or passed by the caller.
func NewProvider(cfg config.AIConfig) (Provider, error) {
	apiKey := cfg.APIKey
	if apiKey == "" {
		return nil, fmt.Errorf(
			"%s API key not found: run 'gitx setup' to configure it",
			cfg.Provider,
		)
	}

	switch cfg.Provider {
	case "openai":
		return newOpenAI(openAIConfig{
			apiKey: apiKey,
			model:  defaultModel(cfg.Model, "gpt-5-mini"),
		}), nil

	case "deepseek":
		return newDeepSeek(deepSeekConfig{
			apiKey: apiKey,
			model:  defaultModel(cfg.Model, "deepseek-v4-flash"),
		}), nil

	default:
		return nil, fmt.Errorf("unknown provider %q", cfg.Provider)
	}
}

func defaultModel(model, fallback string) string {
	if model != "" {
		return model
	}
	return fallback
}
