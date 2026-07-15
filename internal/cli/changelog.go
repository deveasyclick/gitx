package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/gitx/internal/ai"
	"github.com/user/gitx/internal/config"
	"github.com/user/gitx/internal/domain"
	"github.com/user/gitx/internal/git"
	"github.com/user/gitx/internal/prompts"
	"github.com/user/gitx/internal/services"
	"github.com/user/gitx/internal/ui"
)

type changelogFlags struct {
	from   string
	to     string
	output string
	latest bool
}

func newChangelogCmd() *cobra.Command {
	var flags changelogFlags

	cmd := &cobra.Command{
		Use:   "changelog",
		Short: "Generate changelog entries",
		Long:  `Analyze git tags and commits to generate changelog entries.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return changelogRun(cmd, flags)
		},
	}

	cmd.Flags().StringVar(&flags.from, "from", "", "start version tag (inclusive)")
	cmd.Flags().StringVar(&flags.to, "to", "", "end version tag (inclusive)")
	cmd.Flags().StringVar(&flags.output, "output", "", "write output to file")
	cmd.Flags().BoolVar(&flags.latest, "latest", false, "generate for the latest tag only")

	return cmd
}

func changelogRun(cmd *cobra.Command, flags changelogFlags) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	provider, err := ai.NewProvider(cfg.AI)
	if err != nil {
		return err
	}

	gitClient := git.NewExecClient(".")
	svc := services.NewChangelogService(gitClient, provider, prompts.NewChangelogBuilder())

	ui.PrintInfo("Generating changelog...")

	var result *services.ChangelogResult

	switch {
	case flags.latest:
		result, err = svc.GenerateLatest(cmd.Context())
	case flags.from != "":
		to := flags.to
		if to == "" {
			to = "HEAD"
		}
		result, err = svc.GenerateRange(cmd.Context(), flags.from, to)
	default:
		// Default: use latest tag
		result, err = svc.GenerateLatest(cmd.Context())
	}

	if err != nil {
		ui.PrintError(err.Error())
		return fmt.Errorf("changelog")
	}

	formatted := formatChangelog(result.Entries)
	ui.PrintInfo(formatted)

	if flags.output != "" {
		if err := os.WriteFile(flags.output, []byte(formatted), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", flags.output, err)
		}
		ui.PrintSuccess("Written to " + flags.output)
	}

	return nil
}

func formatChangelog(entries []domain.ChangelogEntry) string {
	var b strings.Builder
	for _, entry := range entries {
		if entry.Version != "" {
			fmt.Fprintf(&b, "## %s\n\n", entry.Version)
		}
		writeSection(&b, "Added", entry.Added)
		writeSection(&b, "Fixed", entry.Fixed)
		writeSection(&b, "Changed", entry.Changed)
		writeSection(&b, "Removed", entry.Removed)
	}
	return strings.TrimSpace(b.String())
}

func writeSection(b *strings.Builder, name string, items []string) {
	if len(items) == 0 {
		return
	}
	fmt.Fprintf(b, "### %s\n", name)
	for _, item := range items {
		fmt.Fprintf(b, "- %s\n", item)
	}
	b.WriteString("\n")
}
