package prompts_test

import (
	"strings"
	"testing"

	"github.com/user/gitx/internal/domain"
	"github.com/user/gitx/internal/prompts"
)

func TestCommitBuilder_Build_Conventional(t *testing.T) {
	b := prompts.NewCommitBuilder()
	input := prompts.CommitPromptInput{
		Diff: domain.Change{
			Files: []string{"auth.go"},
			Diff:  "diff --git a/auth.go b/auth.go\nindex abc..def\n--- a/auth.go\n+++ b/auth.go\n@@ -1 +1 @@\n- old\n+ new",
		},
		Branch: "feature/login",
		Style:  "conventional",
	}

	system, user, err := b.Build(input)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if !strings.Contains(system, "conventional commits") {
		t.Errorf("system prompt should mention conventional commits")
	}
	if strings.Contains(system, "emoji") {
		t.Errorf("system prompt should not mention emoji for conventional style")
	}
	if !strings.Contains(user, "feature/login") {
		t.Errorf("user prompt should contain branch name")
	}
	if !strings.Contains(user, "auth.go") {
		t.Errorf("user prompt should contain diff content")
	}
}

func TestCommitBuilder_Build_Gitmoji(t *testing.T) {
	b := prompts.NewCommitBuilder()
	input := prompts.CommitPromptInput{
		Diff: domain.Change{
			Files: []string{"main.go"},
			Diff:  "diff --git a/main.go b/main.go\nindex abc..def\n--- a/main.go\n+++ b/main.go\n@@ -1 +1 @@\n- old\n+ new",
		},
		Branch: "main",
		Style:  "gitmoji",
	}

	system, user, err := b.Build(input)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if !strings.Contains(system, "emoji") {
		t.Errorf("system prompt should mention emoji for gitmoji style")
	}
	if !strings.Contains(user, "main") {
		t.Errorf("user prompt should contain branch name")
	}
}

func TestCommitBuilder_Build_EmptyDiff(t *testing.T) {
	b := prompts.NewCommitBuilder()
	_, _, err := b.Build(prompts.CommitPromptInput{
		Diff: domain.Change{},
	})
	if err == nil {
		t.Fatal("expected error for empty diff")
	}
}

func TestCommitBuilder_Build_SystemPromptRules(t *testing.T) {
	b := prompts.NewCommitBuilder()
	input := prompts.CommitPromptInput{
		Diff: domain.Change{
			Files: []string{"test.go"},
			Diff:  "diff --git a/test.go b/test.go\nindex abc..def\n--- a/test.go\n+++ b/test.go\n@@ -1 +1 @@\n- a\n+ b",
		},
		Branch: "fix/issue",
		Style:  "conventional",
	}

	system, _, err := b.Build(input)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	rules := []string{
		"conventional commits",
		"imperative tense",
		"72 characters",
		"Do not invent",
	}
	for _, rule := range rules {
		if !strings.Contains(system, rule) {
			t.Errorf("system prompt should mention %q", rule)
		}
	}
}
