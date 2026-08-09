package prompts_test

import (
	"strings"
	"testing"

	"github.com/user/gitx/internal/domain"
	"github.com/user/gitx/internal/prompts"
)

func TestReleaseChangelogBuilder_Build(t *testing.T) {
	b := prompts.NewReleaseBuilder()
	input := prompts.ReleaseChangelogInput{
		Version:     "v1.2.0",
		PreviousTag: "v1.1.0",
		CurrentTag:  "v1.2.0",
		Commits: []domain.CommitSummary{
			{Hash: "abc1234", Message: "feat: add payment retries"},
			{Hash: "def4567", Message: "fix: resolve token refresh issue"},
			{Hash: "ghi7890", Message: "feat: add OAuth support"},
		},
	}

	system, user, err := b.BuildReleaseChangelog(input)
	if err != nil {
		t.Fatalf("BuildReleaseChangelog: %v", err)
	}

	// System should contain section headers
	sections := []string{"### Added", "### Fixed", "### Changed", "### Removed"}
	for _, s := range sections {
		if !strings.Contains(system, s) {
			t.Errorf("system prompt should contain section %q", s)
		}
	}

	// System should contain rules
	rules := []string{"Group", "bullet points", "Do not invent", "present-tense"}
	for _, rule := range rules {
		if !strings.Contains(system, rule) {
			t.Errorf("system prompt should mention %q", rule)
		}
	}

	// User should contain version and tags
	if !strings.Contains(user, "v1.2.0") {
		t.Errorf("user prompt should contain version")
	}
	if !strings.Contains(user, "v1.1.0") {
		t.Errorf("user prompt should contain previous tag")
	}
	if !strings.Contains(user, "Previous tag") {
		t.Errorf("user prompt should mention Previous tag")
	}

	// User should contain commit messages and hashes
	if !strings.Contains(user, "add payment retries") {
		t.Errorf("user prompt should contain commit messages")
	}
	if !strings.Contains(user, "abc1234") {
		t.Errorf("user prompt should contain short commit hash")
	}
}

func TestReleaseChangelogBuilder_IncludesDiff(t *testing.T) {
	b := prompts.NewReleaseBuilder()
	system, user, err := b.BuildReleaseChangelog(prompts.ReleaseChangelogInput{
		Version:     "v1.1.0",
		PreviousTag: "v1.0.0",
		CurrentTag:  "v1.1.0",
		Commits:     []domain.CommitSummary{{Hash: "abc1234", Message: "in progress"}},
		Diff:        "diff --git a/auth.go b/auth.go\n@@ -1 +1 @@\n-login\n+login with 2FA",
	})
	if err != nil {
		t.Fatalf("BuildReleaseChangelog: %v", err)
	}
	if !strings.Contains(system, "source of truth") {
		t.Errorf("system prompt should say the diff is the source of truth")
	}
	if !strings.Contains(user, "Diff:") {
		t.Errorf("user prompt should contain a Diff section")
	}
	if !strings.Contains(user, "auth.go") {
		t.Errorf("user prompt should contain diff content")
	}
}

func TestReleaseChangelogBuilder_TruncatesHugeDiff(t *testing.T) {
	b := prompts.NewReleaseBuilder()
	_, user, err := b.BuildReleaseChangelog(prompts.ReleaseChangelogInput{
		Version: "v1.1.0",
		Commits: []domain.CommitSummary{{Hash: "abc", Message: "feat: thing"}},
		Diff:    strings.Repeat("x", 60000),
	})
	if err != nil {
		t.Fatalf("BuildReleaseChangelog: %v", err)
	}
	if !strings.Contains(user, "truncated") {
		t.Errorf("user prompt should mention truncation")
	}
}

func TestReleaseChangelogBuilder_DiffWithoutCommits(t *testing.T) {
	b := prompts.NewReleaseBuilder()
	_, _, err := b.BuildReleaseChangelog(prompts.ReleaseChangelogInput{
		Version: "v1.1.0",
		Diff:    "diff --git a/x.go b/x.go\n@@ -1 +1 @@\n-a\n+b",
	})
	if err != nil {
		t.Fatalf("expected success with diff and no commits, got: %v", err)
	}
}

func TestReleaseChangelogBuilder_NoPreviousTag(t *testing.T) {
	b := prompts.NewReleaseBuilder()
	input := prompts.ReleaseChangelogInput{
		Version:    "v1.0.0",
		CurrentTag: "v1.0.0",
		Commits: []domain.CommitSummary{
			{Hash: "abc123", Message: "feat: initial release"},
		},
	}

	_, user, err := b.BuildReleaseChangelog(input)
	if err != nil {
		t.Fatalf("BuildReleaseChangelog: %v", err)
	}

	if strings.Contains(user, "Previous tag") {
		t.Errorf("should not mention previous tag when none exists")
	}
	if !strings.Contains(user, "Release:") {
		t.Errorf("should mention Release: when no previous tag")
	}
}

func TestReleaseChangelogBuilder_NoCommits(t *testing.T) {
	b := prompts.NewReleaseBuilder()
	_, _, err := b.BuildReleaseChangelog(prompts.ReleaseChangelogInput{
		Version: "v1.0.0",
		Commits: []domain.CommitSummary{},
	})
	if err == nil {
		t.Fatal("expected error for no commits")
	}
}

func TestReleasePRBuilder_Build(t *testing.T) {
	b := prompts.NewReleaseBuilder()
	input := prompts.ReleasePRInput{
		Version:       "v2.0.0",
		BaseBranch:    "main",
		HeadBranch:    "release/v2.0.0",
		ChangelogText: "### Added\n- Payment retries\n",
		Commits: []domain.CommitSummary{
			{Hash: "abc1234", Message: "feat: payment retries"},
		},
		FilesChanged: []string{"payment.go", "handler.go"},
		DiffStat:     "2 files changed, 50 insertions(+)",
	}

	system, user, err := b.BuildPRDescription(input)
	if err != nil {
		t.Fatalf("BuildPRDescription: %v", err)
	}

	// System should describe the output structure
	expectedParts := []string{"pull request", "release", "changelog", "summary"}
	for _, part := range expectedParts {
		if !strings.Contains(system, part) {
			t.Errorf("system prompt should contain %q", part)
		}
	}

	// User should contain version, branches, changelog
	if !strings.Contains(user, "v2.0.0") {
		t.Errorf("user prompt should contain version")
	}
	if !strings.Contains(user, "release/v2.0.0") {
		t.Errorf("user prompt should contain head branch")
	}
	if !strings.Contains(user, "main") {
		t.Errorf("user prompt should contain base branch")
	}
	if !strings.Contains(user, "Payment retries") {
		t.Errorf("user prompt should contain changelog text")
	}
	if !strings.Contains(user, "Files changed:") {
		t.Errorf("user prompt should contain Files changed line")
	}
	if !strings.Contains(user, "50 insertions") {
		t.Errorf("user prompt should contain diff stat")
	}
	if !strings.Contains(user, "abc1234") {
		t.Errorf("user prompt should contain commit hash")
	}
}

func TestVersionSuggestionBuilder_Build(t *testing.T) {
	b := prompts.NewReleaseBuilder()
	commits := []domain.CommitSummary{
		{Hash: "abc123", Message: "feat: add payment retries"},
		{Hash: "def456", Message: "fix: resolve timeout"},
	}

	system, user, err := b.BuildVersionSuggestion(commits)
	if err != nil {
		t.Fatalf("BuildVersionSuggestion: %v", err)
	}

	// System should describe version bump types
	expectedTypes := []string{"major", "minor", "patch"}
	for _, tp := range expectedTypes {
		if !strings.Contains(system, tp) {
			t.Errorf("system prompt should mention %q", tp)
		}
	}

	// User should contain commit information
	if !strings.Contains(user, "add payment retries") {
		t.Errorf("user prompt should contain commit messages")
	}
	if !strings.Contains(user, "abc123") {
		t.Errorf("user prompt should contain commit hash")
	}

	// Should end with a clear instruction
	if !strings.Contains(user, "version bump") {
		t.Errorf("user prompt should ask about version bump")
	}
}

func TestVersionSuggestionBuilder_EmptyCommits(t *testing.T) {
	b := prompts.NewReleaseBuilder()
	_, user, err := b.BuildVersionSuggestion([]domain.CommitSummary{})
	if err != nil {
		t.Fatalf("BuildVersionSuggestion with empty commits: %v", err)
	}

	// Empty commits should still produce a prompt (no validation error)
	if !strings.Contains(user, "version bump") {
		t.Errorf("should still produce a prompt even with empty commits")
	}
}

func TestReleaseChangelogBuilder_Rules(t *testing.T) {
	b := prompts.NewReleaseBuilder()
	input := prompts.ReleaseChangelogInput{
		Version: "v1.0.0",
		Commits: []domain.CommitSummary{
			{Hash: "abc123", Message: "chore: cleanup"},
		},
	}

	system, _, err := b.BuildReleaseChangelog(input)
	if err != nil {
		t.Fatalf("BuildReleaseChangelog: %v", err)
	}

	rules := []string{"Group", "concise", "conventional commits", "categorization"}
	for _, rule := range rules {
		if !strings.Contains(system, rule) {
			t.Errorf("system prompt should mention %q", rule)
		}
	}
}

func TestReleasePRBuilder_NoFilesChanged(t *testing.T) {
	b := prompts.NewReleaseBuilder()
	input := prompts.ReleasePRInput{
		Version:    "v1.0.0",
		BaseBranch: "main",
		HeadBranch: "feature",
		Commits: []domain.CommitSummary{
			{Hash: "abc123", Message: "feat: new feature"},
		},
	}

	_, user, err := b.BuildPRDescription(input)
	if err != nil {
		t.Fatalf("BuildPRDescription: %v", err)
	}

	// Should not mention Files changed when no files are given
	if strings.Contains(user, "Files changed:") {
		t.Errorf("should not mention Files changed when empty")
	}

	// Should still contain the branches and version
	if !strings.Contains(user, "v1.0.0") || !strings.Contains(user, "main") || !strings.Contains(user, "feature") {
		t.Errorf("user prompt should contain version and branches")
	}
}
