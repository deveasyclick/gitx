package cli

import (
	"context"
	"fmt"
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

type commitFlags struct {
	dryRun   bool
	provider string
	model    string
	staged   bool
	unstaged bool
	group    bool
}

func newCommitCmd() *cobra.Command {
	var flags commitFlags

	cmd := &cobra.Command{
		Use:   "commit",
		Short: "Generate a commit message from changes",
		Long: `Analyze changes and generate a commit message.

By default, uses staged changes. If no changes are staged,
automatically falls back to unstaged (tracked) changes.
Use --staged or --unstaged to explicitly control the source.
Use --group to split changes by directory into separate commits.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return commitRun(cmd, flags)
		},
	}

	cmd.Flags().BoolVar(&flags.dryRun, "dry-run", false, "generate but do not commit")
	cmd.Flags().StringVar(&flags.provider, "provider", "", "override AI provider")
	cmd.Flags().StringVar(&flags.model, "model", "", "override AI model")
	cmd.Flags().BoolVar(&flags.staged, "staged", false, "use only staged changes")
	cmd.Flags().BoolVar(&flags.unstaged, "unstaged", false, "use only unstaged changes")
	cmd.Flags().BoolVar(&flags.group, "group", false, "split changes by directory into separate commits")

	return cmd
}

func commitRun(cmd *cobra.Command, flags commitFlags) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if flags.provider != "" {
		cfg.AI.Provider = flags.provider
	}
	if flags.model != "" {
		cfg.AI.Model = flags.model
	}

	provider, err := ai.NewProvider(cfg.AI)
	if err != nil {
		ui.PrintError(err.Error())
		return fmt.Errorf("setup")
	}

	gitClient := git.NewExecClient(".")
	svc := services.NewCommitService(gitClient, provider, prompts.NewCommitBuilder())

	mode := resolveCommitMode(flags)
	modeLabel := modeLabel(mode)

	if flags.group {
		return groupedCommitRun(cmd, svc, gitClient, mode, modeLabel, flags.dryRun)
	}

	return singleCommitRun(cmd, svc, gitClient, mode, modeLabel, flags.dryRun)
}

// singleCommitRun handles the standard single commit flow.
func singleCommitRun(cmd *cobra.Command, svc *services.CommitService, gitClient *git.ExecClient, mode services.CommitMode, modeLabel string, dryRun bool) error {
	spinner := ui.NewSpinner(fmt.Sprintf("Generating commit from %s...", modeLabel))
	if ui.IsInteractive() {
		spinner.Start()
	}

	result, err := svc.Generate(cmd.Context(), mode)
	if ui.IsInteractive() {
		spinner.Stop()
	}
	if err != nil {
		ui.PrintError(err.Error())
		return fmt.Errorf("commit")
	}

	// Show files that will be committed
	printChangedFiles(gitClient, cmd.Context(), mode)

	ui.PrintCommitMessage(result.Message, outputLevel())

	if dryRun {
		return nil
	}

	return commitInteractionLoop(cmd, svc, gitClient, mode, result.Message)
}

// groupedCommitRun handles the grouped commit flow.
func groupedCommitRun(cmd *cobra.Command, svc *services.CommitService, gitClient *git.ExecClient, mode services.CommitMode, modeLabel string, dryRun bool) error {
	spinner := ui.NewSpinner(fmt.Sprintf("Grouping and generating commits from %s...", modeLabel))
	if ui.IsInteractive() {
		spinner.Start()
	}

	groupedResult, err := svc.GenerateGrouped(cmd.Context(), mode)
	if ui.IsInteractive() {
		spinner.Stop()
	}
	if err != nil {
		ui.PrintError(err.Error())
		return fmt.Errorf("grouped commit")
	}

	// Show all grouped messages
	fmt.Println()
	ui.PrintInfo(fmt.Sprintf("Generated %d commits grouped by directory:", len(groupedResult.Results)))
	fmt.Println()

	for i, g := range groupedResult.Results {
		dirLabel := g.Dir
		if dirLabel == "." {
			dirLabel = "root"
		}
		fmt.Printf("  ── [%d] %s ── (%d file(s))\n", i+1, dirLabel, len(g.Files))
		fmt.Printf("      %s\n", g.Message.Title)
		if g.Message.Body != "" {
			bodyPreview := strings.Split(g.Message.Body, "\n")[0]
			if len(bodyPreview) > 60 {
				bodyPreview = bodyPreview[:57] + "..."
			}
			fmt.Printf("      %s\n", bodyPreview)
		}
		fmt.Println()
	}

	if dryRun {
		return nil
	}

	// Let user choose a group to work with
	return groupedInteractionLoop(cmd, svc, gitClient, mode, groupedResult)
}

// commitInteractionLoop handles the Yes/No/Edit/Regenerate/Copy flow for a single message.
func commitInteractionLoop(cmd *cobra.Command, svc *services.CommitService, gitClient *git.ExecClient, mode services.CommitMode, msg domain.CommitMessage) error {
	for attempt := 1; attempt <= 3; attempt++ {
		choice := ui.ConfirmCommit()

		switch choice {
		case ui.ConfirmYes:
			if err := ensureStagedForCommit(cmd, gitClient, mode); err != nil {
				return err
			}
			if err := gitClient.Commit(cmd.Context(), msg); err != nil {
				ui.PrintError(fmt.Sprintf("Failed to commit: %s", err.Error()))
				return fmt.Errorf("commit")
			}
			ui.PrintSuccess("Committed successfully.")
			return nil

		case ui.ConfirmEdit:
			edited, err := ui.OpenEditor(msg.String())
			if err != nil {
				ui.PrintError(fmt.Sprintf("Editing failed: %s", err.Error()))
				return fmt.Errorf("edit")
			}
			if edited == "" || edited == msg.String() {
				continue
			}
			parsed := parseEditedMessage(edited)
			if err := ensureStagedForCommit(cmd, gitClient, mode); err != nil {
				return err
			}
			if err := gitClient.Commit(cmd.Context(), parsed); err != nil {
				ui.PrintError(fmt.Sprintf("Failed to commit: %s", err.Error()))
				return fmt.Errorf("commit")
			}
			ui.PrintSuccess("Committed with edits.")
			return nil

		case ui.ConfirmCopy:
			if err := ui.CopyToClipboard(msg.String()); err != nil {
				ui.PrintWarning(fmt.Sprintf("Copy failed: %s", err.Error()))
			} else {
				ui.PrintSuccess("Commit message copied to clipboard.")
			}
			attempt--

		case ui.ConfirmRegenerate:
			if attempt < 3 {
				spinner := ui.NewSpinner("Regenerating...")
				if ui.IsInteractive() {
					spinner.Start()
				}
				result, err := svc.Generate(cmd.Context(), mode)
				if ui.IsInteractive() {
					spinner.Stop()
				}
				if err != nil {
					ui.PrintError(err.Error())
					return fmt.Errorf("regenerate")
				}
				msg = result.Message
				ui.PrintCommitMessage(msg, outputLevel())
			} else {
				ui.PrintWarning("Max regeneration attempts reached.")
				edited, err := ui.OpenEditor(msg.String())
				if err != nil {
					ui.PrintError(fmt.Sprintf("Editing failed: %s", err.Error()))
					return fmt.Errorf("edit")
				}
				parsed := parseEditedMessage(edited)
				if err := ensureStagedForCommit(cmd, gitClient, mode); err != nil {
					return err
				}
				if err := gitClient.Commit(cmd.Context(), parsed); err != nil {
					ui.PrintError(fmt.Sprintf("Failed to commit: %s", err.Error()))
					return fmt.Errorf("commit")
				}
				ui.PrintSuccess("Committed with edits.")
				return nil
			}

		case ui.ConfirmNo:
			ui.PrintInfo("Commit cancelled.")
			return nil
		}
	}
	return nil
}

// groupedInteractionLoop lets the user review and commit grouped messages.
// Stages only the current group's files before committing, leaving other groups untouched.
func groupedInteractionLoop(cmd *cobra.Command, svc *services.CommitService, gitClient *git.ExecClient, mode services.CommitMode, groupedResult *services.GroupedResult) error {
	remaining := make([]services.GroupedGenerateResult, len(groupedResult.Results))
	copy(remaining, groupedResult.Results)

	for len(remaining) > 0 {
		g := remaining[0]
		dirLabel := g.Dir
		if dirLabel == "." {
			dirLabel = "root"
		}

		fmt.Println()
		ui.PrintInfo(fmt.Sprintf("Group: %s", dirLabel))
		fmt.Println("  Files:")
		for _, f := range g.Files {
			fmt.Printf("    - %s\n", f)
		}
		fmt.Println()
		ui.PrintCommitMessage(g.Message, outputLevel())

		choice := ui.ConfirmCommitGrouped()
		switch choice {
		case ui.ConfirmYes:
			if err := commitGroup(gitClient, cmd.Context(), g); err != nil {
				return err
			}
			remaining = remaining[1:]

		case ui.ConfirmYesAll:
			for _, rg := range remaining {
				if err := commitGroup(gitClient, cmd.Context(), rg); err != nil {
					return err
				}
			}
			remaining = nil

		case ui.ConfirmEdit:
			edited, err := ui.OpenEditor(g.Message.String())
			if err != nil {
				ui.PrintError(fmt.Sprintf("Editing failed: %s", err.Error()))
				return fmt.Errorf("edit")
			}
			if edited == "" || edited == g.Message.String() {
				continue
			}
			parsed := parseEditedMessage(edited)
			if err := gitClient.UnstageAll(cmd.Context()); err != nil {
				ui.PrintError(fmt.Sprintf("Failed to unstage: %s", err.Error()))
				return fmt.Errorf("edit")
			}
			if err := gitClient.Stage(cmd.Context(), g.Files); err != nil {
				ui.PrintError(fmt.Sprintf("Failed to stage %s: %s", dirLabel, err.Error()))
				return fmt.Errorf("edit")
			}
			if err := gitClient.Commit(cmd.Context(), parsed); err != nil {
				return fmt.Errorf("committing %s: %w", dirLabel, err)
			}
			ui.PrintSuccess(fmt.Sprintf("Committed %s with edits.", dirLabel))
			remaining = remaining[1:]

		case ui.ConfirmCopy:
			if err := ui.CopyToClipboard(g.Message.String()); err != nil {
				ui.PrintWarning(fmt.Sprintf("Copy failed: %s", err.Error()))
			} else {
				ui.PrintSuccess("Commit message copied to clipboard.")
			}

		case ui.ConfirmRegenerate:
			spinner := ui.NewSpinner(fmt.Sprintf("Regenerating message for %s...", dirLabel))
			if ui.IsInteractive() {
				spinner.Start()
			}
			newResult, err := svc.GenerateGrouped(cmd.Context(), mode)
			if ui.IsInteractive() {
				spinner.Stop()
			}
			if err != nil {
				ui.PrintError(err.Error())
				continue
			}
			for _, ng := range newResult.Results {
				if ng.Dir == g.Dir {
					remaining[0] = ng
					g = ng
					break
				}
			}
			ui.PrintInfo(fmt.Sprintf("New message for %s:", dirLabel))
			ui.PrintCommitMessage(g.Message, outputLevel())

		case ui.ConfirmStage:
			if err := gitClient.UnstageAll(cmd.Context()); err != nil {
				ui.PrintError(fmt.Sprintf("Failed to unstage: %s", err.Error()))
				return fmt.Errorf("stage")
			}
			if err := gitClient.Stage(cmd.Context(), g.Files); err != nil {
				ui.PrintError(fmt.Sprintf("Failed to stage %s: %s", dirLabel, err.Error()))
				return fmt.Errorf("stage")
			}
			ui.PrintSuccess(fmt.Sprintf("Staged %d file(s) for %s. Review with: git diff --cached", len(g.Files), dirLabel))

		case ui.ConfirmNo:
			ui.PrintInfo(fmt.Sprintf("Skipped %s.", dirLabel))
			remaining = remaining[1:]

		case ui.ConfirmQuit:
			ui.PrintInfo("Quit grouped commit.")
			remaining = nil
		}
	}

	if len(groupedResult.Results) > 0 && len(remaining) == 0 {
		ui.PrintSuccess("Done.")
	}
	return nil
}

// commitGroup stages and commits a single group.
func commitGroup(gitClient *git.ExecClient, ctx context.Context, g services.GroupedGenerateResult) error {
	dirLabel := g.Dir
	if dirLabel == "." {
		dirLabel = "root"
	}
	if err := gitClient.UnstageAll(ctx); err != nil {
		ui.PrintError(fmt.Sprintf("Failed to unstage: %s", err.Error()))
		return fmt.Errorf("commit")
	}
	if err := gitClient.Stage(ctx, g.Files); err != nil {
		ui.PrintError(fmt.Sprintf("Failed to stage %s: %s", dirLabel, err.Error()))
		return fmt.Errorf("commit")
	}
	if err := gitClient.Commit(ctx, g.Message); err != nil {
		ui.PrintError(fmt.Sprintf("Failed to commit %s: %s", dirLabel, err.Error()))
		return fmt.Errorf("commit")
	}
	ui.PrintSuccess(fmt.Sprintf("Committed %s.", dirLabel))
	return nil
}

// resolveCommitMode maps CLI flags to a CommitMode.
func resolveCommitMode(flags commitFlags) services.CommitMode {
	if flags.staged && flags.unstaged {
		return services.CommitModeStaged
	}
	if flags.unstaged {
		return services.CommitModeUnstaged
	}
	if flags.staged {
		return services.CommitModeStaged
	}
	return services.CommitModeAuto
}

func modeLabel(mode services.CommitMode) string {
	switch mode {
	case services.CommitModeStaged:
		return "staged changes"
	case services.CommitModeUnstaged:
		return "unstaged changes"
	default:
		return "changes"
	}
}

// printChangedFiles shows the files that will be committed.
func printChangedFiles(gitClient *git.ExecClient, ctx context.Context, mode services.CommitMode) {
	switch mode {
	case services.CommitModeStaged:
		status, err := gitClient.Status(ctx)
		if err == nil && !status.IsEmpty {
			fmt.Println()
			ui.PrintInfo("Staged files:")
			for _, f := range status.Files {
				fmt.Printf("  - %s\n", f)
			}
		}

	case services.CommitModeUnstaged:
		unstaged, err := gitClient.UnstagedStatus(ctx)
		if err == nil && !unstaged.IsEmpty {
			fmt.Println()
			ui.PrintInfo("Unstaged files:")
			for _, f := range unstaged.Files {
				fmt.Printf("  - %s\n", f)
			}
		}

	default: // CommitModeAuto
		// Show staged if any, otherwise unstaged
		status, err := gitClient.Status(ctx)
		if err == nil && !status.IsEmpty {
			fmt.Println()
			ui.PrintInfo("Files:")
			for _, f := range status.Files {
				fmt.Printf("  - %s\n", f)
			}
			return
		}
		unstaged, err := gitClient.UnstagedStatus(ctx)
		if err == nil && !unstaged.IsEmpty {
			fmt.Println()
			ui.PrintInfo("Unstaged files:")
			for _, f := range unstaged.Files {
				fmt.Printf("  - %s\n", f)
			}
		}
	}
}

// ensureStagedForCommit ensures the correct files are staged before committing.
// Behavior depends on the mode:
//   --staged:   commit only what's already staged (no change)
//   --unstaged: unstage everything, then stage only the unstaged files
//   auto:       stage everything before committing
func ensureStagedForCommit(cmd *cobra.Command, gitClient *git.ExecClient, mode services.CommitMode) error {
	switch mode {
	case services.CommitModeStaged:
		// Commit only what's already staged — don't stage anything new.
		return nil

	case services.CommitModeUnstaged:
		// Capture unstaged files BEFORE unstaging, so we only re-stage
		// the originally-unstaged files — not any that were already staged.
		unstaged, err := gitClient.UnstagedStatus(cmd.Context())
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to get unstaged files: %s", err.Error()))
			return fmt.Errorf("stage")
		}
		// Now unstage everything.
		if err := gitClient.UnstageAll(cmd.Context()); err != nil {
			ui.PrintError(fmt.Sprintf("Failed to unstage: %s", err.Error()))
			return fmt.Errorf("stage")
		}
		// Then stage only the files that were unstaged.
		if len(unstaged.Files) > 0 {
			if err := gitClient.Stage(cmd.Context(), unstaged.Files); err != nil {
				ui.PrintError(fmt.Sprintf("Failed to stage unstaged files: %s", err.Error()))
				return fmt.Errorf("stage")
			}
		}
		return nil

	default: // CommitModeAuto
		// Stage everything before committing.
		if err := gitClient.StageAll(cmd.Context()); err != nil {
			ui.PrintError(fmt.Sprintf("Failed to stage changes: %s", err.Error()))
			return fmt.Errorf("stage")
		}
		return nil
	}
}

func parseEditedMessage(text string) domain.CommitMessage {
	text = strings.TrimSpace(text)
	title, body, _ := strings.Cut(text, "\n")
	body = strings.TrimSpace(body)
	return domain.CommitMessage{
		Title: title,
		Body:  body,
		Style: "conventional",
	}
}
