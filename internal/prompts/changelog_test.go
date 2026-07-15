package prompts_test

import (
	"strings"
	"testing"

	"github.com/user/gitx/internal/git"
	"github.com/user/gitx/internal/prompts"
)

func TestChangelogBuilder_Build(t *testing.T) {
	b := prompts.NewChangelogBuilder()
	input := prompts.ChangelogPromptInput{
		From: "v1.0.0",
		To:   "v2.0.0",
		Commits: []git.CommitLog{
			{Hash: "abc1234", Message: "feat: add payment retries"},
			{Hash: "def4567", Message: "fix: resolve token refresh issue"},
			{Hash: "ghi7890", Message: "feat: add OAuth support"},
		},
		Tags: []string{"v1.0.0", "v2.0.0"},
	}

	system, user, err := b.Build(input)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	sections := []string{"## Added", "## Fixed", "## Changed", "## Removed"}
	for _, s := range sections {
		if !strings.Contains(system, s) {
			t.Errorf("system prompt should contain section %q", s)
		}
	}

	if !strings.Contains(user, "v1.0.0") {
		t.Errorf("user prompt should contain from tag")
	}
	if !strings.Contains(user, "v2.0.0") {
		t.Errorf("user prompt should contain to tag")
	}
	if !strings.Contains(user, "add payment retries") {
		t.Errorf("user prompt should contain commit messages")
	}
	if !strings.Contains(user, "abc1234") {
		t.Errorf("user prompt should contain commit hashes")
	}
}

func TestChangelogBuilder_Build_NoCommits(t *testing.T) {
	b := prompts.NewChangelogBuilder()
	_, _, err := b.Build(prompts.ChangelogPromptInput{
		From:    "v1.0.0",
		To:      "v2.0.0",
		Commits: []git.CommitLog{},
	})
	if err == nil {
		t.Fatal("expected error for no commits")
	}
}

func TestChangelogBuilder_Build_NoTags(t *testing.T) {
	b := prompts.NewChangelogBuilder()
	input := prompts.ChangelogPromptInput{
		From: "v1.0.0",
		To:   "HEAD",
		Commits: []git.CommitLog{
			{Hash: "abc123", Message: "fix: critical bug"},
		},
	}

	_, user, err := b.Build(input)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if !strings.Contains(user, "v1.0.0") {
		t.Errorf("user prompt should contain from tag")
	}
	if !strings.Contains(user, "HEAD") {
		t.Errorf("user prompt should contain to ref")
	}
}

func TestChangelogBuilder_Build_SinceOnly(t *testing.T) {
	b := prompts.NewChangelogBuilder()
	input := prompts.ChangelogPromptInput{
		From: "v1.0.0",
		Commits: []git.CommitLog{
			{Hash: "abc123", Message: "feat: new feature"},
		},
	}

	_, user, err := b.Build(input)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if !strings.Contains(user, "Since") {
		t.Errorf("user prompt should say Since: when only from is set")
	}
}

func TestChangelogBuilder_Build_Rules(t *testing.T) {
	b := prompts.NewChangelogBuilder()
	input := prompts.ChangelogPromptInput{
		Commits: []git.CommitLog{
			{Hash: "abc123", Message: "chore: cleanup"},
		},
	}

	system, _, err := b.Build(input)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	rules := []string{"Group", "bullet points", "Do not invent", "concise"}
	for _, rule := range rules {
		if !strings.Contains(system, rule) {
			t.Errorf("system prompt should mention %q", rule)
		}
	}
}
