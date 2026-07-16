package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/user/gitx/internal/ai"
	"github.com/user/gitx/internal/config"
	"github.com/user/gitx/internal/git"
	"github.com/user/gitx/internal/prompts"
	"github.com/user/gitx/internal/services"
	"github.com/user/gitx/internal/ui"
)

type describeFlags struct {
	staged   bool
	unstaged bool
	base     string
	commits  int
	output   string
}

func newDescribeCmd() *cobra.Command {
	var flags describeFlags

	cmd := &cobra.Command{
		Use:   "describe",
		Short: "Describe current repository state",
		Long: `Analyze the current branch and describe its state.

By default, describes the last 10 commits. Use --commits to control
how many, --staged or --unstaged for working tree changes, or --base
to see all commits since that branch.`,
		Example: `  gitx describe
  gitx describe --commits 5
  gitx describe --staged
  gitx describe --unstaged
  gitx describe --base main
  gitx describe --base main --staged
  gitx describe --output state.md`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return describeRun(cmd, flags)
		},
	}

	cmd.Flags().BoolVarP(&flags.staged, "staged", "s", false, "include staged changes")
	cmd.Flags().BoolVarP(&flags.unstaged, "unstaged", "u", false, "include unstaged changes")
	cmd.Flags().StringVar(&flags.base, "base", "", "base branch for commit comparison (default: last 10 commits)")
	cmd.Flags().IntVarP(&flags.commits, "commits", "c", 10, "number of recent commits to include")
	cmd.Flags().StringVarP(&flags.output, "output", "o", "", "write output to file")

	return cmd
}

func describeRun(cmd *cobra.Command, flags describeFlags) error {
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
	svc := services.NewDescribeService(gitClient, provider, prompts.NewDescribeBuilder())

	ui.PrintInfo("Describing current state...")

	result, err := svc.Generate(cmd.Context(), services.DescribeOptions{
		Base:     flags.base,
		Staged:   flags.staged,
		Unstaged: flags.unstaged,
		Commits:  flags.commits,
	})
	if err != nil {
		ui.PrintError(err.Error())
		return fmt.Errorf("describe")
	}

	// Print the formatted description
	formatted := result.Description.String()
	fmt.Println()
	fmt.Println(formatted)
	fmt.Println()

	// Write to file if requested
	if flags.output != "" {
		if err := os.WriteFile(flags.output, []byte(formatted), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", flags.output, err)
		}
		ui.PrintSuccess("Written to " + flags.output)
	}

	return nil
}
