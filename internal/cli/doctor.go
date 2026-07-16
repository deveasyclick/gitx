package cli

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/user/gitx/internal/config"
	"github.com/user/gitx/internal/git"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose installation",
		Long:  `Check that GitX is properly configured and ready to use.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return doctorRun()
		},
	}
}

type checkResult struct {
	name    string
	status  string // "✓" or "✗"
	detail  string
	fix     string // suggested fix, empty if no issue
}

func doctorRun() error {
	fmt.Println("GitX Doctor")
	fmt.Println("===========")
	fmt.Println()

	var results []checkResult

	// 1. Check git installation
	results = append(results, checkGit())

	// 2. Check repository
	results = append(results, checkRepository())

	// 3. Run git checks in repo context if available
	repoResult := checkRepository()
	inRepo := repoResult.status == "✓"
	results[len(results)-1] = repoResult

	if inRepo {
		results = append(results, checkGitConfig())
	}

	// 4. Check configuration
	results = append(results, checkConfig())

	// 5. Check AI provider
	results = append(results, checkAIProvider())

	// 6. Check clipboard tool (for copy feature)
	results = append(results, checkClipboard())

	// Print results
	allOK := true
	for _, r := range results {
		if r.status == "✓" {
			fmt.Printf(" %s %s\n", r.status, r.name)
		} else {
			fmt.Printf(" %s %s\n", r.status, r.name)
			if r.detail != "" {
				fmt.Printf("   Reason: %s\n", r.detail)
			}
			if r.fix != "" {
				fmt.Printf("   Fix: %s\n", r.fix)
			}
			allOK = false
		}
	}

	if allOK {
		fmt.Println()
		fmt.Println("All checks passed.")
	}

	return nil
}

func checkGit() checkResult {
	if _, err := exec.LookPath("git"); err != nil {
		return checkResult{
			name:   "Git installed",
			status: "✗",
			detail: "git not found in PATH",
			fix:    "Install git: https://git-scm.com/downloads",
		}
	}

	cmd := exec.Command("git", "--version")
	if out, err := cmd.Output(); err != nil {
		return checkResult{
			name:   "Git installed",
			status: "✗",
			detail: err.Error(),
		}
	} else {
		return checkResult{
			name:   "Git installed (" + string(out) + ")",
			status: "✓",
		}
	}
}

func checkRepository() checkResult {
	client := git.NewExecClient(".")

	info, err := client.RepoInfo(context.Background())
	if err != nil {
		return checkResult{
			name:   "Repository detected",
			status: "✗",
			detail: "not a git repository or no commits yet",
			fix:    "Run: git init && git add . && git commit -m \"initial\"",
		}
	}

	detail := info.CurrentBranch
	if info.Remote != "" {
		detail += " → " + info.Remote
	}
	return checkResult{
		name:   "Repository detected",
		status: "✓",
		detail: detail,
	}
}

func checkGitConfig() checkResult {
	client := git.NewExecClient(".")

	// Check user.name and user.email are set
	for _, key := range []string{"user.name", "user.email"} {
		cmd := exec.Command("git", "config", key)
		out, err := cmd.Output()
		if err != nil || len(out) == 0 {
			return checkResult{
				name:   "Git user configured",
				status: "✗",
				detail: key + " is not set",
				fix:    fmt.Sprintf("Run: git config --global %s <value>", key),
			}
		}
	}
	_ = client
	return checkResult{
		name:   "Git user configured",
		status: "✓",
	}
}

func checkConfig() checkResult {
	cfg, err := config.Load()
	if err != nil {
		return checkResult{
			name:   "GitX config found",
			status: "✗",
			detail: err.Error(),
		}
	}

	return checkResult{
		name:   "Config loaded",
		status: "✓",
		detail: fmt.Sprintf("provider=%s model=%s", cfg.AI.Provider, cfg.AI.Model),
	}
}

func checkAIProvider() checkResult {
	cfg, err := config.Load()
	if err != nil {
		return checkResult{
			name:   "AI provider configured",
			status: "✗",
			detail: "config not loaded",
		}
	}

	if cfg.AI.APIKey == "" {
		return checkResult{
			name:   "AI provider configured",
			status: "✗",
			detail: "API key not set",
			fix:    "Run: gitx setup",
		}
	}

	return checkResult{
		name:   "AI provider configured",
		status: "✓",
		detail: fmt.Sprintf("%s (%s)", cfg.AI.Provider, cfg.AI.Model),
	}
}

// clipboardTools lists clipboard utilities per platform.
var clipboardTools = []struct {
	name    string // binary name
	install string // install command hint
}{
	{"pbcopy", "Already available on macOS"},
	{"wl-copy", "sudo apt install wl-clipboard  (Ubuntu/Debian)  |  sudo dnf install wl-clipboard  (Fedora)"},
	{"xsel", "sudo apt install xsel  (Ubuntu/Debian)  |  sudo dnf install xsel  (Fedora)"},
	{"xclip", "sudo apt install xclip  (Ubuntu/Debian)  |  sudo dnf install xclip  (Fedora)"},
}

func checkClipboard() checkResult {
	for _, tool := range clipboardTools {
		if _, err := exec.LookPath(tool.name); err == nil {
			return checkResult{
				name:   "Clipboard tool (for copy)",
				status: "✓",
				detail: tool.name,
			}
		}
	}

	return checkResult{
		name:   "Clipboard tool (for copy)",
		status: "✗",
		detail: "no clipboard tool found",
		fix:    clipboardTools[2].install, // suggest xsel first (most common)
	}
}
