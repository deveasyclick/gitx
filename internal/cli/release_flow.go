package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/gitx/internal/domain"
	"github.com/user/gitx/internal/git"
	"github.com/user/gitx/internal/services"
	"github.com/user/gitx/internal/ui"
)

// errNothingToRelease signals that the branch comparison found nothing to release.
var errNothingToRelease = errors.New("nothing to release")

// releaseRunner carries the shared state of a release run across workflow steps.
type releaseRunner struct {
	cmd   *cobra.Command
	flags releaseFlags
	svc   *services.ReleaseService
	git   git.Client
}

// run executes the full release workflow.
func (r *releaseRunner) run() error {
	info, err := r.compareAndShow()
	if err != nil {
		if errors.Is(err, errNothingToRelease) {
			return nil
		}
		return err
	}

	// Step 4: Ask if user wants to proceed
	fmt.Println()
	if !askProceed() {
		ui.PrintInfo("Release cancelled.")
		return nil
	}

	// Step 5: Choose version
	version := r.resolveVersion(info)

	// Step 6: Choose release branch strategy
	useReleaseBranch, releaseBranch := askReleaseBranch(version, info.HeadBranch)
	if useReleaseBranch {
		ui.PrintInfo(fmt.Sprintf("Will create release branch: %s", releaseBranch))
	}

	// Step 7: Generate changelog
	changelogText, err := r.generateChangelog(info, version)
	if err != nil {
		return err
	}

	// Step 8: Bump version in files
	versionFiles := r.bumpVersionFiles(version)

	// Step 9: Generate PR description (optional)
	prDescription, err := r.generatePR(info, version, changelogText)
	if err != nil {
		return err
	}

	// Step 10: Checkout guidance + summary
	r.finishAndSummarize(info, version, useReleaseBranch, releaseBranch, versionFiles, prDescription)
	return nil
}

// compareAndShow resolves the base (a branch or a version tag), compares it
// against the current branch, and prints the comparison (steps 1-3).
func (r *releaseRunner) compareAndShow() (domain.ReleaseInfo, error) {
	tagBase := r.flags.tag
	if tagBase != "" && r.flags.base != "" {
		ui.PrintError("--base and --tag are mutually exclusive")
		return domain.ReleaseInfo{}, fmt.Errorf("release: use either --base or --tag, not both")
	}
	if tagBase == "" && r.flags.base == "" {
		ui.PrintError("--base (branch) or --tag (version tag) is required")
		return domain.ReleaseInfo{}, fmt.Errorf("release: --base or --tag is required (e.g., --base main or --tag v1.2.3)")
	}

	var (
		result *services.CompareResult
		err    error
	)
	if tagBase != "" {
		ui.PrintInfo(fmt.Sprintf("🔍 Comparing against tag %s...", tagBase))
		result, err = r.svc.CompareWithTag(r.cmd.Context(), tagBase)
	} else {
		ui.PrintInfo("🔍 Comparing branches...")
		result, err = r.svc.CompareBranches(r.cmd.Context(), r.flags.base, "")
	}
	if err != nil {
		ui.PrintError(err.Error())
		return domain.ReleaseInfo{}, fmt.Errorf("compare")
	}

	info := result.Info

	fmt.Println()
	ui.PrintInfo("📦 Release Comparison")
	fmt.Println()
	if tagBase != "" {
		fmt.Printf("  Base tag:    %s\n", info.BaseBranch)
		fmt.Printf("  Head branch: %s\n", info.HeadBranch)
		fmt.Printf("  Commits:     %d commit(s) since tag\n", len(info.Commits))
	} else {
		fmt.Printf("  Base branch:  %s\n", info.BaseBranch)
		fmt.Printf("  Head branch:  %s\n", info.HeadBranch)
		fmt.Printf("  Commits:      %d commit(s) ahead\n", len(info.Commits))
	}
	if info.Diff.DiffStat != "" {
		fmt.Printf("  Changes:      %s\n", info.Diff.DiffStat)
	} else {
		fmt.Printf("  Files:        %d file(s) changed\n", len(info.Diff.Files))
	}

	if len(info.Commits) > 0 {
		fmt.Println()
		if tagBase != "" {
			ui.PrintInfo("Commits since tag:")
		} else {
			ui.PrintInfo("Commits in head not in base:")
		}
		maxShow := 20
		for i, c := range info.Commits {
			if i >= maxShow {
				fmt.Printf("  ... and %d more\n", len(info.Commits)-maxShow)
				break
			}
			hash := c.Hash
			if len(hash) > 7 {
				hash = hash[:7]
			}
			fmt.Printf("  • %s %s\n", hash, c.Message)
		}
	}

	if len(info.Commits) == 0 && info.Diff.IsEmpty() {
		fmt.Println()
		ui.PrintInfo("✨ Nothing to release — no commits or changes since the base.")
		return domain.ReleaseInfo{}, errNothingToRelease
	}

	return info, nil
}

// resolveVersion picks the release version (step 5): an explicit flag, an AI
// suggestion, or a bump type — then lets the user confirm or edit it.
func (r *releaseRunner) resolveVersion(info domain.ReleaseInfo) string {
	bumpType := r.flags.bump
	version := r.flags.version

	if version == "" {
		if bumpType == "" {
			// Ask AI for suggestion
			ui.PrintInfo("🤖 Analyzing commits to suggest version bump...")
			suggested, err := r.svc.SuggestVersion(r.cmd.Context(), info.Commits)
			if err != nil {
				// Fallback to minor
				suggested = domain.BumpMinor
				ui.PrintVerbose(fmt.Sprintf("AI suggestion failed, defaulting to %s: %s", suggested, err), outputLevel())
			}
			bumpType = string(suggested)
			ui.PrintInfo(fmt.Sprintf("AI suggests a %s bump", bumpType))
		}
		version = services.SuggestNextVersion(info.CurrentVersion, bumpType)
	}

	// Let user confirm or edit version
	return askVersion(version, info.CurrentVersion)
}

// generateChangelog generates, reviews, and writes the changelog entry
// (step 7). It returns "" when the step was skipped.
func (r *releaseRunner) generateChangelog(info domain.ReleaseInfo, version string) (string, error) {
	if r.flags.noChangelog {
		return "", nil
	}

	ui.PrintInfo("📝 Generating changelog entry...")

	text, err := runWithRetry("Changelog generation failed", "changelog", func() (string, error) {
		return withSpinner("Generating changelog with AI...", func() (string, error) {
			return r.svc.GenerateReleaseChangelog(r.cmd.Context(), info, version)
		})
	})
	if err != nil {
		return "", err
	}
	if text == "" {
		return "", nil // skipped
	}

	fmt.Println()
	ui.PrintInfo("Generated changelog entry:")
	fmt.Println()
	fmt.Println(text)
	fmt.Println()

	// Offer to regenerate or accept
	text = reviewLoop(reviewLoopOptions{
		prompt:    "[A] Accept  [R] Regenerate  [C] Copy  [E] Edit\n(default: Accept): ",
		copiedMsg: "Changelog copied to clipboard.",
		editedMsg: "Changelog updated from editor.",
		regenerate: func() (string, error) {
			return withSpinner("Regenerating changelog...", func() (string, error) {
				return r.svc.GenerateReleaseChangelog(r.cmd.Context(), info, version)
			})
		},
	}, text)

	// Write changelog file
	if _, err := runWithRetry("Changelog write failed", "changelog write", func() (string, error) {
		if err := r.svc.UpdateChangelogFile(r.flags.changelog, version, text); err != nil {
			return "", err
		}
		ui.PrintSuccess(fmt.Sprintf("✓ Updated %s", r.flags.changelog))
		return "", nil
	}); err != nil {
		return "", err
	}

	return text, nil
}

// bumpVersionFiles updates version strings in known files (step 8).
func (r *releaseRunner) bumpVersionFiles(version string) []string {
	if r.flags.noVersion {
		return nil
	}

	ui.PrintInfo(fmt.Sprintf("🔖 Bumping version to %s...", version))

	versionFiles, err := r.svc.BumpVersionInFiles(version)
	if err != nil {
		ui.PrintWarning(fmt.Sprintf("Version bump issue: %s", err.Error()))
	}
	if len(versionFiles) > 0 {
		for _, f := range versionFiles {
			ui.PrintSuccess(fmt.Sprintf("✓ Updated %s", f))
		}
	} else {
		ui.PrintInfo("No version files found to bump (checked package.json, package-lock.json)")
	}
	return versionFiles
}

// generatePR optionally generates and reviews the release PR description
// (step 9). It returns "" when PR generation was skipped.
func (r *releaseRunner) generatePR(info domain.ReleaseInfo, version, changelogText string) (string, error) {
	generatePR := r.flags.pr
	if r.flags.noPR {
		generatePR = false
	} else if !r.flags.pr {
		generatePR = askGeneratePR()
	}
	if !generatePR {
		return "", nil
	}

	ui.PrintInfo("📋 Generating PR description...")

	prDescription, err := runWithRetry("PR description generation failed", "pr description", func() (string, error) {
		return withSpinner("Generating PR description with AI...", func() (string, error) {
			return r.svc.GeneratePRDescription(r.cmd.Context(), info, version, changelogText)
		})
	})
	if err != nil {
		return "", err
	}

	fmt.Println()
	ui.PrintInfo("Generated PR description:")
	fmt.Println()
	fmt.Println(prDescription)
	fmt.Println()

	return reviewLoop(reviewLoopOptions{
		prompt:    "[A] Accept  [C] Copy  [E] Edit\n(default: Accept): ",
		copiedMsg: "PR description copied to clipboard.",
		editedMsg: "PR description updated from editor.",
	}, prDescription), nil
}

// finishAndSummarize prints checkout instructions and the release summary
// (step 10).
func (r *releaseRunner) finishAndSummarize(info domain.ReleaseInfo, version string, useReleaseBranch bool, releaseBranch string, versionFiles []string, prDescription string) {
	if useReleaseBranch {
		if askCheckoutBranch(releaseBranch) {
			// Best effort: stage everything for the release
			_ = r.git.StageAll(r.cmd.Context())
			// Just guide the user — we don't force checkout
			fmt.Println()
			ui.PrintInfo(fmt.Sprintf("Run: git checkout -b %s", releaseBranch))
			ui.PrintInfo(fmt.Sprintf("Then: git add -A && git commit -m \"release: %s\"", version))
			ui.PrintInfo(fmt.Sprintf("Then: git tag %s", version))
		}
	} else {
		fmt.Println()
		ui.PrintInfo("Release prepared on current branch.")
		ui.PrintInfo(fmt.Sprintf("Review changes, then: git add -A && git commit -m \"release: %s\"", version))
		ui.PrintInfo(fmt.Sprintf("Then: git tag %s", version))
	}

	// ── Summary ──
	fmt.Println()
	ui.PrintSuccess("✅ Release summary:")
	fmt.Printf("  Version:   %s\n", version)
	fmt.Printf("  Branch:    %s", info.HeadBranch)
	if useReleaseBranch {
		fmt.Printf(" → %s", releaseBranch)
	}
	fmt.Println()
	if len(versionFiles) > 0 {
		fmt.Printf("  Files:     %s\n", strings.Join(versionFiles, ", "))
	}
	if prDescription != "" {
		// Show first line of PR
		firstLine := strings.SplitN(prDescription, "\n", 2)[0]
		fmt.Printf("  PR title:  %s\n", firstLine)
	}
}

// withSpinner runs fn while showing a spinner in interactive terminals.
func withSpinner(msg string, fn func() (string, error)) (string, error) {
	spinner := ui.NewSpinner(msg)
	if ui.IsInteractive() {
		spinner.Start()
	}
	defer func() {
		if ui.IsInteractive() {
			spinner.Stop()
		}
	}()
	return fn()
}

// runWithRetry runs fn and, on failure, lets the user Retry, Skip, or Quit.
// It returns (result, nil) when fn succeeds or the user skips, and
// (zero, abortErr) when the user quits.
func runWithRetry[T any](failMsg, abortErr string, fn func() (T, error)) (T, error) {
	var zero T
	for {
		result, err := fn()
		if err == nil {
			return result, nil
		}
		ui.PrintError(fmt.Sprintf("%s: %s", failMsg, err))
		switch askRetryOrSkip(failMsg) {
		case retryAction:
			continue
		case skipAction:
			return zero, nil
		case quitAction:
			return zero, fmt.Errorf("%s", abortErr)
		}
	}
}
