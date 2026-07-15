package services

import (
	"context"
	"testing"

	"github.com/user/gitx/internal/git"
	"github.com/user/gitx/internal/prompts"
)

func TestChangelogService_GenerateRange_Success(t *testing.T) {
	svc := NewChangelogService(
		&mockGit{
			log: []git.CommitLog{
				{Hash: "abc1234", Message: "feat: add payment retries"},
				{Hash: "def4567", Message: "fix: resolve token refresh"},
			},
		},
		&mockAI{text: "## Added\n- Payment retries\n\n## Fixed\n- Token refresh issue"},
		prompts.NewChangelogBuilder(),
	)

	result, err := svc.GenerateRange(context.Background(), "v1.0.0", "v2.0.0")
	if err != nil {
		t.Fatalf("GenerateRange: %v", err)
	}

	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result.Entries))
	}

	entry := result.Entries[0]
	if entry.Version != "v2.0.0" {
		t.Errorf("Version = %q", entry.Version)
	}
	if len(entry.Added) != 1 || entry.Added[0] != "Payment retries" {
		t.Errorf("Added = %v", entry.Added)
	}
	if len(entry.Fixed) != 1 || entry.Fixed[0] != "Token refresh issue" {
		t.Errorf("Fixed = %v", entry.Fixed)
	}
}

func TestChangelogService_GenerateLatest_Success(t *testing.T) {
	svc := NewChangelogService(
		&mockGit{
			tags: []string{"v2.0.0", "v1.0.0"},
			log: []git.CommitLog{
				{Hash: "abc1234", Message: "feat: new feature"},
			},
		},
		&mockAI{text: "## Added\n- New feature"},
		prompts.NewChangelogBuilder(),
	)

	result, err := svc.GenerateLatest(context.Background())
	if err != nil {
		t.Fatalf("GenerateLatest: %v", err)
	}

	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result.Entries))
	}
	if result.Entries[0].Version != "v2.0.0" {
		t.Errorf("Version = %q", result.Entries[0].Version)
	}
}

func TestChangelogService_GenerateLatest_NoTags(t *testing.T) {
	svc := NewChangelogService(
		&mockGit{
			tags: []string{},
		},
		&mockAI{},
		prompts.NewChangelogBuilder(),
	)

	_, err := svc.GenerateLatest(context.Background())
	if err == nil {
		t.Fatal("expected error for no tags")
	}
}

func TestChangelogService_GenerateRange_NoCommits(t *testing.T) {
	svc := NewChangelogService(
		&mockGit{
			log: []git.CommitLog{},
		},
		&mockAI{},
		prompts.NewChangelogBuilder(),
	)

	_, err := svc.GenerateRange(context.Background(), "v1.0.0", "v2.0.0")
	if err == nil {
		t.Fatal("expected error for no commits")
	}
}

func TestParseChangelogEntry(t *testing.T) {
	text := "## Added\n- Feature X\n- Feature Y\n\n## Fixed\n- Bug Z\n\n## Changed\n- Refactored module"

	entry := parseChangelogEntry(text, "v3.0.0")
	if entry.Version != "v3.0.0" {
		t.Errorf("Version = %q", entry.Version)
	}
	if len(entry.Added) != 2 {
		t.Errorf("Added = %v", entry.Added)
	}
	if len(entry.Fixed) != 1 || entry.Fixed[0] != "Bug Z" {
		t.Errorf("Fixed = %v", entry.Fixed)
	}
	if len(entry.Changed) != 1 || entry.Changed[0] != "Refactored module" {
		t.Errorf("Changed = %v", entry.Changed)
	}
}
