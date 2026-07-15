package prompts_test

import (
	"strings"
	"testing"
	"time"

	"github.com/user/gitx/internal/domain"
	"github.com/user/gitx/internal/git"
	"github.com/user/gitx/internal/prompts"
)

func TestPRBuilder_Build(t *testing.T) {
	b := prompts.NewPRBuilder()
	input := prompts.PRPromptInput{
		Diff: domain.Change{
			Files: []string{"payment.go"},
			Diff:  "diff --git a/payment.go b/payment.go\nindex abc..def\n--- a/payment.go\n+++ b/payment.go\n@@ -1 +1 @@\n- old\n+ new",
		},
		Branch: "feat/payment-retry",
		Base:   "main",
		Commits: []git.CommitLog{
			{Hash: "abc123", Author: "Alice", Message: "add retry handler", Date: time.Now()},
			{Hash: "def456", Author: "Alice", Message: "improve fallback", Date: time.Now()},
		},
	}

	system, user, err := b.Build(input)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	sections := []string{"## Summary", "## Changes", "## Testing", "## Risks", "## Breaking Changes"}
	for _, s := range sections {
		if !strings.Contains(system, s) {
			t.Errorf("system prompt should contain section %q", s)
		}
	}

	if !strings.Contains(user, "feat/payment-retry") {
		t.Errorf("user prompt should contain branch name")
	}
	if !strings.Contains(user, "main") {
		t.Errorf("user prompt should contain base branch")
	}
	if !strings.Contains(user, "add retry handler") {
		t.Errorf("user prompt should contain commit messages")
	}
}

func TestPRBuilder_Build_EmptyDiffAndCommits(t *testing.T) {
	b := prompts.NewPRBuilder()
	_, _, err := b.Build(prompts.PRPromptInput{})
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestPRBuilder_Build_DiffOnly(t *testing.T) {
	b := prompts.NewPRBuilder()
	input := prompts.PRPromptInput{
		Diff: domain.Change{
			Files: []string{"main.go"},
			Diff:  "diff --git a/main.go b/main.go\n",
		},
		Branch: "feature",
		Base:   "main",
	}

	system, user, err := b.Build(input)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if !strings.Contains(system, "## Summary") {
		t.Errorf("system prompt should contain sections")
	}
	_ = user
}

func TestPRBuilder_Build_Rules(t *testing.T) {
	b := prompts.NewPRBuilder()
	input := prompts.PRPromptInput{
		Diff: domain.Change{
			Files: []string{"x.go"},
			Diff:  "diff --git a/x.go b/x.go\n",
		},
		Branch: "feature",
		Base:   "main",
		Commits: []git.CommitLog{
			{Hash: "aaa111", Author: "Test", Message: "fix: stuff"},
		},
	}

	system, _, err := b.Build(input)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	rules := []string{"concise", "why", "Do not invent"}
	for _, rule := range rules {
		if !strings.Contains(system, rule) {
			t.Errorf("system prompt should mention %q", rule)
		}
	}
}
