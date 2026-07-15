package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/gitx/internal/config"
	"github.com/user/gitx/internal/ui"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  `View and modify GitX configuration settings.`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Long:  `Get a configuration value. Keys: ai.provider, ai.model, commit.style`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return configGetRun(args[0])
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long:  `Set a configuration value. Keys: ai.provider, ai.model, commit.style`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return configSetRun(args[0], args[1])
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all configuration values",
		RunE: func(cmd *cobra.Command, args []string) error {
			return configListRun()
		},
	})

	return cmd
}

func configGetRun(key string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	val, ok := getConfigValue(cfg, key)
	if !ok {
		return fmt.Errorf("unknown config key: %q (valid: ai.provider, ai.model, ai.api_key, commit.style)", key)
	}

	if key == "ai.api_key" && val != "" {
		val = val[:4] + "..." + val[len(val)-4:]
	}

	fmt.Println(val)
	return nil
}

func configSetRun(key, value string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if !setConfigValue(cfg, key, value) {
		return fmt.Errorf("unknown config key: %q (valid: ai.provider, ai.model, ai.api_key, commit.style)", key)
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	displayVal := value
	if key == "ai.api_key" {
		displayVal = value[:4] + "..." + value[len(value)-4:]
	}
	ui.PrintSuccess("Set " + key + " = " + displayVal)
	return nil
}

func configListRun() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	apiKeyDisplay := ""
	if cfg.AI.APIKey != "" {
		apiKeyDisplay = cfg.AI.APIKey[:4] + "..." + cfg.AI.APIKey[len(cfg.AI.APIKey)-4:]
	}

	entries := []struct {
		key string
		val string
	}{
		{"ai.provider", cfg.AI.Provider},
		{"ai.model", cfg.AI.Model},
		{"ai.api_key", apiKeyDisplay},
		{"commit.style", cfg.Commit.Style},
	}

	for _, e := range entries {
		fmt.Printf("%s=%s\n", e.key, e.val)
	}

	return nil
}

// getConfigValue returns the value for a dot-notation key.
func getConfigValue(cfg *config.Config, key string) (string, bool) {
	switch strings.ToLower(key) {
	case "ai.provider":
		return cfg.AI.Provider, true
	case "ai.model":
		return cfg.AI.Model, true
	case "ai.api_key":
		return cfg.AI.APIKey, true
	case "commit.style":
		return cfg.Commit.Style, true
	default:
		return "", false
	}
}

// setConfigValue sets a value on the config struct by dot-notation key.
// Returns false if the key is unknown.
func setConfigValue(cfg *config.Config, key, value string) bool {
	switch strings.ToLower(key) {
	case "ai.provider":
		cfg.AI.Provider = value
		return true
	case "ai.model":
		cfg.AI.Model = value
		return true
	case "ai.api_key":
		cfg.AI.APIKey = value
		return true
	case "commit.style":
		cfg.Commit.Style = value
		return true
	default:
		return false
	}
}
