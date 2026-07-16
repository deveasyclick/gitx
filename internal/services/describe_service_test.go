package services

import (
	"context"
	"testing"
	"time"

	"github.com/user/gitx/internal/domain"
	"github.com/user/gitx/internal/git"
	"github.com/user/gitx/internal/prompts"
)

func TestDescribeService_Generate_CommitsOnly(t *testing.T) {
	svc := NewDescribeService(
		&mockGit{
			repo: domain.RepoInfo{CurrentBranch: "feat/payment-retry"},
			log: []git.CommitLog{
				{Hash: "abc1234", Author: "Alice", Message: "feat: add retry handler", Date: time.Now()},
				{Hash: "def5678", Author: "Alice", Message: "fix: handle timeout", Date: time.Now()},
			},
		},
		&mockAI{text: "## Overview\nAdds payment retry with timeout handling.\n\n## Commits\n- Added retry handler with backoff\n- Fixed timeout edge case"},
		prompts.NewDescribeBuilder(),
	)

	result, err := svc.Generate(context.Background(), DescribeOptions{})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if result.Description.Overview != "Adds payment retry with timeout handling." {
		t.Errorf("Overview = %q", result.Description.Overview)
	}
	if len(result.Description.Commits) != 2 {
		t.Errorf("Commits count = %d, want 2", len(result.Description.Commits))
	}
	if len(result.Description.StagedChanges) != 0 {
		t.Errorf("expected no staged changes, got %d", len(result.Description.StagedChanges))
	}
	if len(result.Description.UnstagedChanges) != 0 {
		t.Errorf("expected no unstaged changes, got %d", len(result.Description.UnstagedChanges))
	}
}

func TestDescribeService_Generate_WithStaged(t *testing.T) {
	svc := NewDescribeService(
		&mockGit{
			repo: domain.RepoInfo{CurrentBranch: "feat/payment"},
			log: []git.CommitLog{
				{Hash: "abc1234", Author: "Alice", Message: "feat: add payment", Date: time.Now()},
			},
			diff: domain.Change{
				Files: []string{"payment.go"},
				Diff:  "diff --git a/payment.go b/payment.go\n+new payment flow",
			},
		},
		&mockAI{text: "## Overview\nExtends payment system.\n\n## Commits\n- Added payment feature\n\n## Staged Changes\n- Added payment flow handler"},
		prompts.NewDescribeBuilder(),
	)

	result, err := svc.Generate(context.Background(), DescribeOptions{Staged: true})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if result.Description.Overview != "Extends payment system." {
		t.Errorf("Overview = %q", result.Description.Overview)
	}
	if len(result.Description.StagedChanges) != 1 || result.Description.StagedChanges[0] != "Added payment flow handler" {
		t.Errorf("StagedChanges = %v", result.Description.StagedChanges)
	}
}

func TestDescribeService_Generate_WithUnstaged(t *testing.T) {
	svc := NewDescribeService(
		&mockGit{
			repo: domain.RepoInfo{CurrentBranch: "fix/timeout"},
			log: []git.CommitLog{
				{Hash: "abc1234", Author: "Bob", Message: "fix: timeout", Date: time.Now()},
			},
			diff: domain.Change{
				Files: []string{"handler.go"},
				Diff:  "diff --git a/handler.go b/handler.go\n+retry logic",
			},
		},
		&mockAI{text: "## Overview\nFixes timeout with retry.\n\n## Commits\n- Fixed timeout issue\n\n## Unstaged Changes\n- Added retry logic to handler"},
		prompts.NewDescribeBuilder(),
	)

	result, err := svc.Generate(context.Background(), DescribeOptions{Unstaged: true})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if len(result.Description.UnstagedChanges) != 1 || result.Description.UnstagedChanges[0] != "Added retry logic to handler" {
		t.Errorf("UnstagedChanges = %v", result.Description.UnstagedChanges)
	}
}

func TestDescribeService_Generate_NothingToDescribe(t *testing.T) {
	svc := NewDescribeService(
		&mockGit{
			repo: domain.RepoInfo{CurrentBranch: "main"},
		},
		&mockAI{},
		prompts.NewDescribeBuilder(),
	)

	_, err := svc.Generate(context.Background(), DescribeOptions{})
	if err == nil {
		t.Fatal("expected error for nothing to describe")
	}
}

func TestDescribeService_Generate_WithBase(t *testing.T) {
	svc := NewDescribeService(
		&mockGit{
			repo: domain.RepoInfo{CurrentBranch: "feat/payment"},
			log: []git.CommitLog{
				{Hash: "abc1234", Author: "Alice", Message: "feat: add payment", Date: time.Now()},
				{Hash: "def5678", Author: "Alice", Message: "feat: add validation", Date: time.Now()},
			},
		},
		&mockAI{text: "## Overview\nPayment feature complete.\n\n## Commits\n- Added payment feature\n- Added validation"},
		prompts.NewDescribeBuilder(),
	)

	result, err := svc.Generate(context.Background(), DescribeOptions{Base: "main"})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if len(result.Description.Commits) != 2 {
		t.Errorf("Commits count = %d, want 2", len(result.Description.Commits))
	}
}

func TestExtractSection(t *testing.T) {
	text := `## Overview
Adds payment retry.

## Commits
- Added retry handler

## Unstaged Changes
Unit tests pending`

	if got := extractSection(text, "Overview"); got != "Adds payment retry." {
		t.Errorf("Overview = %q", got)
	}
	if got := extractSection(text, "Unstaged Changes"); got != "Unit tests pending" {
		t.Errorf("Unstaged Changes = %q", got)
	}
	if got := extractSection(text, "Missing"); got != "" {
		t.Errorf("Missing section should be empty")
	}
}

func TestExtractBullets(t *testing.T) {
	text := `## Commits
- Added retry handler
- Improved fallback
* Another item`

	bullets := extractBullets(text, "Commits")
	if len(bullets) != 3 {
		t.Fatalf("expected 3 bullets, got %d: %v", len(bullets), bullets)
	}
	if bullets[0] != "Added retry handler" {
		t.Errorf("bullets[0] = %q", bullets[0])
	}
}

func TestParseDescribeResponse(t *testing.T) {
	text := "## Overview\nPayment work.\n\n## Commits\n- Added payment\n- Added refund\n\n## Staged Changes\n- Staged handler.go\n\n## Unstaged Changes\n- Unstaged test.go"

	desc := parseDescribeResponse(text)
	if desc.Overview != "Payment work." {
		t.Errorf("Overview = %q", desc.Overview)
	}
	if len(desc.Commits) != 2 {
		t.Errorf("Commits = %v", desc.Commits)
	}
	if len(desc.StagedChanges) != 1 {
		t.Errorf("StagedChanges = %v", desc.StagedChanges)
	}
	if len(desc.UnstagedChanges) != 1 {
		t.Errorf("UnstagedChanges = %v", desc.UnstagedChanges)
	}
}
