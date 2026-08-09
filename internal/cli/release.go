package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/user/gitx/internal/ai"
	"github.com/user/gitx/internal/config"
	"github.com/user/gitx/internal/git"
	"github.com/user/gitx/internal/prompts"
	"github.com/user/gitx/internal/services"
	"github.com/user/gitx/internal/ui"
)

type releaseFlags struct {
	base        string
	version     string
	bump        string
	changelog   string // path to CHANGELOG.md (default: CHANGELOG.md)
	noChangelog bool
	noVersion   bool
	noPR        bool
	pr          bool
}

func newReleaseCmd() *cobra.Command {
	var flags releaseFlags

	cmd := &cobra.Command{
		Use:   "release",
		Short: "Prepare and generate a release",
		Long: `Orchestrate a release workflow: compare branches, generate changelog,
bump versions, and optionally create a PR description.

--base is required. The workflow is interactive: you review changes,
choose version bumps, review AI-generated content, and decide what
to do at every step (PR description generation asks first).

Examples:
  gitx release --base main
  gitx release --base main --bump major
  gitx release --base main --version v2.0.0 --no-changelog
  gitx release --base main --pr`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return releaseRun(cmd, flags)
		},
	}

	cmd.Flags().StringVar(&flags.base, "base", "", "base branch (required, e.g., main, master)")
	cmd.Flags().StringVar(&flags.version, "version", "", "explicit version (e.g., v1.2.3)")
	cmd.Flags().StringVar(&flags.bump, "bump", "", "version bump type: major, minor, patch (AI-suggested if not set)")
	cmd.Flags().StringVar(&flags.changelog, "changelog", "CHANGELOG.md", "path to changelog file")
	cmd.Flags().BoolVar(&flags.noChangelog, "no-changelog", false, "skip changelog generation")
	cmd.Flags().BoolVar(&flags.noVersion, "no-version", false, "skip version bumping")
	cmd.Flags().BoolVar(&flags.noPR, "no-pr", false, "skip PR description generation")
	cmd.Flags().BoolVar(&flags.pr, "pr", false, "generate PR description without asking")

	return cmd
}

// releaseRun wires up the release dependencies and runs the workflow.
func releaseRun(cmd *cobra.Command, flags releaseFlags) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	provider, err := ai.NewProvider(cfg.AI)
	if err != nil {
		ui.PrintError(err.Error())
		return fmt.Errorf("setup")
	}

	gitClient := git.NewExecClient(".")
	svc := services.NewReleaseService(gitClient, provider, prompts.NewReleaseBuilder())

	r := &releaseRunner{cmd: cmd, flags: flags, svc: svc, git: gitClient}
	return r.run()
}
