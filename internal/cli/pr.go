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

type prFlags struct {
	base   string
	output string
}

func newPRCmd() *cobra.Command {
	var flags prFlags

	cmd := &cobra.Command{
		Use:   "pr",
		Short: "Generate a pull request description",
		Long: `Analyze the current branch and generate a pull request description.

Collects commit history and diff against the base branch,
then generates a structured PR description.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return prRun(cmd, flags)
		},
	}

	cmd.Flags().StringVar(&flags.base, "base", "main", "base branch for comparison")
	cmd.Flags().StringVar(&flags.output, "output", "", "write output to file")

	return cmd
}

func prRun(cmd *cobra.Command, flags prFlags) error {
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
	svc := services.NewPRService(gitClient, provider, prompts.NewPRBuilder())

	ui.PrintInfo("Generating pull request description...")

	result, err := svc.Generate(cmd.Context(), flags.base)
	if err != nil {
		ui.PrintError(err.Error())
		return fmt.Errorf("pr")
	}

	formatted := formatPR(result.Description)
	ui.PrintInfo(formatted)

	if flags.output != "" {
		if err := os.WriteFile(flags.output, []byte(formatted), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", flags.output, err)
		}
		ui.PrintSuccess("Written to " + flags.output)
	}

	return nil
}

func formatPR(d domain.PullRequest) string {
	var b strings.Builder

	b.WriteString("Pull Request Description")
	b.WriteString("\n\n")

	if d.Summary != "" {
		b.WriteString("## Summary\n")
		b.WriteString(d.Summary)
		b.WriteString("\n\n")
	}

	if len(d.Changes) > 0 {
		b.WriteString("## Changes\n")
		for _, c := range d.Changes {
			b.WriteString("- ")
			b.WriteString(c)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if d.Testing != "" {
		b.WriteString("## Testing\n")
		b.WriteString(d.Testing)
		b.WriteString("\n\n")
	}

	if d.Risks != "" {
		b.WriteString("## Risks\n")
		b.WriteString(d.Risks)
		b.WriteString("\n\n")
	}

	if d.BreakingNotes != "" {
		b.WriteString("## Breaking Changes\n")
		b.WriteString(d.BreakingNotes)
		b.WriteString("\n")
	}

	return strings.TrimSpace(b.String())
}
