package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/user/gitx/internal/ui"
	"github.com/user/gitx/internal/update"
)

var (
	verbose bool
	jsonOut bool
	noColor bool

	// currentVersion is the build version, set from the value injected into
	// the main package by goreleaser's ldflags. "dev" for local builds.
	currentVersion = "dev"
)

// gitxRepo is the GitHub repository used for update checks.
const gitxRepo = "deveasyclick/gitx"

func Execute(version string) error {
	currentVersion = version
	maybeCheckForUpdates()
	return rootCmd.Execute()
}

var rootCmd = &cobra.Command{
	Use:   "gitx",
	Short: "AI-powered Git assistant",
	Long: `GitX extends Git with intelligent workflows.

Generate commit messages, pull request descriptions,
and changelogs using AI.

Git remains the source of truth. GitX provides intelligence
and automation around existing Git workflows.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Silences errors on commands so we control formatting
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "output in JSON format")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")

	rootCmd.AddCommand(newCommitCmd())
	rootCmd.AddCommand(newDescribeCmd())
	rootCmd.AddCommand(newChangelogCmd())
	rootCmd.AddCommand(newReleaseCmd())
	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newSetupCmd())
	rootCmd.AddCommand(newDoctorCmd())
	rootCmd.AddCommand(newVersionCmd())

	// Disable the built-in completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Silence usage on errors so we print the error message directly
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
}

// outputLevel returns the UI output level based on global flags.
func outputLevel() ui.OutputLevel {
	if jsonOut {
		return ui.OutputJSON
	}
	if verbose {
		return ui.OutputVerbose
	}
	return ui.OutputNormal
}

// maybeCheckForUpdates prints a notice when a newer gitx release exists. It
// never blocks for long and never fails the command: the check is skipped for
// dev builds, non-interactive shells, and GITX_NO_UPDATE_CHECK=1, and network
// failures are ignored.
func maybeCheckForUpdates() {
	if currentVersion == "" || currentVersion == "dev" {
		return
	}
	if os.Getenv("GITX_NO_UPDATE_CHECK") == "1" {
		return
	}
	if !ui.IsInteractive() {
		return
	}
	// doctor reports its own update status; avoid a duplicate notice.
	for _, arg := range os.Args[1:] {
		if arg == "doctor" {
			return
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	latest, err := update.Check(ctx, currentVersion, update.Options{
		Repo:     gitxRepo,
		CacheDir: updateCacheDir(),
	})
	if err != nil || latest == "" {
		return // fail open: an update check never breaks a command
	}

	fmt.Fprintf(ui.Stderr, "\n⚠️  A new version of gitx is available: %s (you have %s)\n", latest, currentVersion)
	fmt.Fprintf(ui.Stderr, "    Update: curl -fsSL https://raw.githubusercontent.com/%s/main/scripts/install.sh | sh\n", gitxRepo)
	fmt.Fprintf(ui.Stderr, "    See:    https://github.com/%s/releases/tag/%s\n\n", gitxRepo, latest)
}

// updateCacheDir returns the directory used to cache update checks, or ""
// when no cache location can be determined (which disables caching).
func updateCacheDir() string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(cacheDir, "gitx")
}
