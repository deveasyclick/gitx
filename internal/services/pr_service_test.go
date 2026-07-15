package services

import (
	"context"
	"testing"
	"time"

	"github.com/user/gitx/internal/domain"
	"github.com/user/gitx/internal/git"
	"github.com/user/gitx/internal/prompts"
)

func TestPRService_Generate_Success(t *testing.T) {
	svc := NewPRService(
		&mockGit{
			diff: domain.Change{
				Files: []string{"payment.go"},
				Diff:  "diff --git a/payment.go b/payment.go\n+new retry logic",
			},
			repo: domain.RepoInfo{CurrentBranch: "feat/payment-retry"},
			log: []git.CommitLog{
				{Hash: "abc1234", Author: "Alice", Message: "add retry handler", Date: time.Now()},
			},
		},
		&mockAI{text: "## Summary\nAdds payment retry.\n\n## Changes\n- Added retry handler\n- Improved fallback\n\n## Testing\nUnit tests added\n\n## Risks\nNone\n\n## Breaking Changes\nNone"},
		prompts.NewPRBuilder(),
	)

	result, err := svc.Generate(context.Background(), "main")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if result.Description.Summary != "Adds payment retry." {
		t.Errorf("Summary = %q", result.Description.Summary)
	}
	if len(result.Description.Changes) != 2 {
		t.Errorf("Changes count = %d, want 2", len(result.Description.Changes))
	}
	if result.Description.Changes[0] != "Added retry handler" {
		t.Errorf("Changes[0] = %q", result.Description.Changes[0])
	}
	if result.Description.Testing != "Unit tests added" {
		t.Errorf("Testing = %q", result.Description.Testing)
	}
	if result.Description.Risks != "None" {
		t.Errorf("Risks = %q", result.Description.Risks)
	}
}

func TestPRService_Generate_EmptyDiff(t *testing.T) {
	svc := NewPRService(
		&mockGit{
			diff: domain.Change{},
			repo: domain.RepoInfo{CurrentBranch: "feature"},
		},
		&mockAI{text: "## Summary\nNothing changed"},
		prompts.NewPRBuilder(),
	)

	_, err := svc.Generate(context.Background(), "main")
	if err == nil {
		t.Fatal("expected error for empty diff and no commits")
	}
}

func TestExtractSection(t *testing.T) {
	text := `## Summary
Adds payment retry.

## Changes
- Added retry handler

## Testing
Unit tests added`

	if got := extractSection(text, "Summary"); got != "Adds payment retry." {
		t.Errorf("Summary = %q", got)
	}
	if got := extractSection(text, "Testing"); got != "Unit tests added" {
		t.Errorf("Testing = %q", got)
	}
	if got := extractSection(text, "Missing"); got != "" {
		t.Errorf("Missing section should be empty")
	}
}

func TestExtractBullets(t *testing.T) {
	text := `## Changes
- Added retry handler
- Improved fallback
* Another item`

	bullets := extractBullets(text, "Changes")
	if len(bullets) != 3 {
		t.Fatalf("expected 3 bullets, got %d: %v", len(bullets), bullets)
	}
	if bullets[0] != "Added retry handler" {
		t.Errorf("bullets[0] = %q", bullets[0])
	}
}

func TestParsePRDescription(t *testing.T) {
	text := "## Summary\nAdds OAuth.\n\n## Changes\n- Added login\n- Added refresh\n\n## Risks\nLow\n\n## Breaking Changes\nAPI change in v2"

	pr := parsePRDescription(text)
	if pr.Summary != "Adds OAuth." {
		t.Errorf("Summary = %q", pr.Summary)
	}
	if len(pr.Changes) != 2 {
		t.Errorf("Changes = %v", pr.Changes)
	}
	if pr.BreakingNotes != "API change in v2" {
		t.Errorf("BreakingNotes = %q", pr.BreakingNotes)
	}
}
