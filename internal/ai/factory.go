package ai

import (
	"fmt"

	"github.com/user/gitx/internal/config"
)

// NewProvider creates a provider based on the configuration.
// API keys are resolved from config file first, then environment variables.
func NewProvider(cfg config.AIConfig) (Provider, error) {
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = config.ResolveAPIKey(cfg.Provider)
	}
	if apiKey == "" {
		return nil, fmt.Errorf(
			"%s API key not found: set %s or use 'gitx setup'",
			cfg.Provider,
			config.EnvAPIKeyVar(cfg.Provider),
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
