package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/gitx/internal/config"
	"github.com/user/gitx/internal/ui"
)

func newSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Interactive configuration setup",
		Long: `Walk through an interactive setup to configure your AI provider,
model, and API key. All settings are saved to ~/.config/gitx/config.yaml.

Run this at any time to switch providers or models.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return setupRun()
		},
	}
}

func setupRun() error {
	reader := bufio.NewReader(os.Stdin)

	// Load current config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	fmt.Println()
	ui.PrintInfo("╔══════════════════════════════════════════╗")
	ui.PrintInfo("║        GitX Interactive Setup            ║")
	ui.PrintInfo("╚══════════════════════════════════════════╝")
	fmt.Println()

	// Show current configuration
	if cfg.AI.Provider != "" || cfg.AI.Model != "" || cfg.AI.APIKey != "" {
		ui.PrintInfo("Current configuration:")
		if cfg.AI.Provider != "" {
			fmt.Printf("  Provider: %s\n", cfg.AI.Provider)
		}
		if cfg.AI.Model != "" {
			fmt.Printf("  Model:    %s\n", cfg.AI.Model)
		}
		if cfg.AI.APIKey != "" {
			masked := cfg.AI.APIKey[:4] + "..." + cfg.AI.APIKey[len(cfg.AI.APIKey)-4:]
			fmt.Printf("  API Key:  %s\n", masked)
		}
		fmt.Println()
	}

	// --- Step 1: Select Provider ---
	provider, err := promptSelect(reader, "Select AI provider", []string{
		"openai",
		"deepseek",
		"anthropic",
		"google",
		"openrouter",
		"ollama",
	})
	if err != nil {
		return err
	}
	cfg.AI.Provider = provider
	fmt.Println()

	// --- Step 2: Select Model ---
	model, err := promptModel(reader, provider)
	if err != nil {
		return err
	}
	cfg.AI.Model = model
	fmt.Println()

	// --- Step 3: API Key ---
	apiKey, err := promptAPIKey(reader, provider, cfg.AI.APIKey)
	if err != nil {
		return err
	}
	if apiKey != "" {
		cfg.AI.APIKey = apiKey
	}
	fmt.Println()

	// --- Step 4: Confirm and Save ---
	fmt.Println("Summary:")
	fmt.Printf("  Provider: %s\n", cfg.AI.Provider)
	fmt.Printf("  Model:    %s\n", cfg.AI.Model)
	if cfg.AI.APIKey != "" {
		masked := cfg.AI.APIKey[:4] + "..." + cfg.AI.APIKey[len(cfg.AI.APIKey)-4:]
		fmt.Printf("  API Key:  %s\n", masked)
	} else {
		fmt.Printf("  API Key:  (not set)\n")
	}
	fmt.Println()

	confirmed, err := promptConfirm(reader, "Save this configuration?")
	if err != nil {
		return err
	}
	if !confirmed {
		ui.PrintInfo("Setup cancelled. No changes saved.")
		return nil
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	ui.PrintSuccess("Configuration saved to ~/.config/gitx/config.yaml")
	fmt.Println()

	// Check clipboard tool for the copy feature
	if err := checkAndSuggestClipboard(reader); err != nil {
		return err
	}

	// Verify the setup works
	ui.PrintInfo("To verify your setup, run: gitx doctor")
	fmt.Println()

	return nil
}

// checkAndSuggestClipboard checks if a clipboard tool is available and offers to install one.
func checkAndSuggestClipboard(reader *bufio.Reader) error {
	// Check if any clipboard tool is already available
	for _, name := range []string{"pbcopy", "wl-copy", "xsel", "xclip"} {
		if _, err := exec.LookPath(name); err == nil {
			return nil // already have one
		}
	}

	fmt.Println()
	ui.PrintInfo("No clipboard tool found. GitX needs one for the copy feature ([C] Copy).")

	installCmd := detectClipboardInstall()
	if installCmd == "" {
		ui.PrintInfo("Install one manually: sudo apt install xsel  (Debian/Ubuntu)  |  sudo dnf install xsel  (Fedora)")
		return nil
	}

	install, err := promptConfirm(reader, fmt.Sprintf("Install clipboard tool? (%s)", installCmd))
	if err != nil {
		return err
	}
	if !install {
		ui.PrintInfo("Skipped. Install manually when needed.")
		return nil
	}

	cmd := exec.Command("sh", "-c", installCmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	ui.PrintSuccess("Clipboard tool installed!")
	return nil
}

// detectClipboardInstall returns the package manager command to install a clipboard tool.
func detectClipboardInstall() string {
	switch runtime.GOOS {
	case "darwin":
		// macOS has pbcopy built-in, so we shouldn't reach here.
		// But if we do, Homebrew can install xsel as a fallback.
		if hasCommand("brew") {
			return "brew install xsel"
		}
		return ""
	case "linux":
		switch {
		case hasCommand("apt-get"):
			return "sudo apt-get install -y xsel"
		case hasCommand("dnf"):
			return "sudo dnf install -y xsel"
		case hasCommand("yum"):
			return "sudo yum install -y xsel"
		case hasCommand("pacman"):
			return "sudo pacman -S --noconfirm xsel"
		case hasCommand("apk"):
			return "sudo apk add xsel"
		}
		return ""
	default:
		return ""
	}
}

func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// promptSelect shows a numbered list of options and returns the selected one.
func promptSelect(reader *bufio.Reader, prompt string, options []string) (string, error) {
	for {
		fmt.Printf("%s:\n", prompt)
		for i, opt := range options {
			fmt.Printf("  %d. %s\n", i+1, opt)
		}
		fmt.Printf("Enter number (1-%d): ", len(options))

		input, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("read input: %w", err)
		}
		input = strings.TrimSpace(input)

		var n int
		if _, err := fmt.Sscanf(input, "%d", &n); err != nil || n < 1 || n > len(options) {
			fmt.Printf("Invalid selection. Please enter a number between 1 and %d.\n\n", len(options))
			continue
		}

		return options[n-1], nil
	}
}

// promptModel shows known models for the provider and lets the user pick or type a custom one.
func promptModel(reader *bufio.Reader, provider string) (string, error) {
	knownModels := config.KnownModels(provider)

	if len(knownModels) == 0 {
		fmt.Printf("Enter model for %s: ", provider)
		input, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("read input: %w", err)
		}
		return strings.TrimSpace(input), nil
	}

	fmt.Println("Select model:")
	for i, m := range knownModels {
		fmt.Printf("  %d. %s\n", i+1, m)
	}
	fmt.Printf("  %d. Custom model\n", len(knownModels)+1)
	fmt.Printf("Enter number (1-%d): ", len(knownModels)+1)

	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read input: %w", err)
	}
	input = strings.TrimSpace(input)

	var n int
	if _, err := fmt.Sscanf(input, "%d", &n); err != nil || n < 1 || n > len(knownModels)+1 {
		// Invalid input, use first model as default
		fmt.Printf("Using default model: %s\n", knownModels[0])
		return knownModels[0], nil
	}

	if n == len(knownModels)+1 {
		// Custom model
		fmt.Print("Enter model name: ")
		custom, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("read input: %w", err)
		}
		return strings.TrimSpace(custom), nil
	}

	return knownModels[n-1], nil
}

// promptAPIKey asks for an API key. Shows existing key if present.
func promptAPIKey(reader *bufio.Reader, provider string, existingKey string) (string, error) {
	if existingKey != "" {
		masked := existingKey[:4] + "..." + existingKey[len(existingKey)-4:]
		keep, err := promptConfirm(reader, fmt.Sprintf(
			"Keep existing API key (%s)?", masked,
		))
		if err != nil {
			return "", err
		}
		if keep {
			return "", nil // keep existing
		}
	}

	fmt.Printf("Enter your %s API key (or press Enter to skip): ", provider)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read input: %w", err)
	}
	input = strings.TrimSpace(input)

	if input == "" {
		return "", nil
	}

	return input, nil
}

// promptConfirm asks a yes/no question.
func promptConfirm(reader *bufio.Reader, prompt string) (bool, error) {
	for {
		fmt.Printf("%s [Y/n]: ", prompt)
		input, err := reader.ReadString('\n')
		if err != nil {
			return false, fmt.Errorf("read input: %w", err)
		}
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "y", "yes", "":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			fmt.Println("Please answer 'y' or 'n'.")
		}
	}
}
