package services

import (
	"context"
	"errors"
	"testing"

	"github.com/user/gitx/internal/ai"
	"github.com/user/gitx/internal/domain"
	"github.com/user/gitx/internal/git"
	"github.com/user/gitx/internal/prompts"
)

func TestCommitService_Generate_Success(t *testing.T) {
	svc := NewCommitService(
		&mockGit{
			status:  git.StagedChanges{Files: []string{"main.go"}, IsEmpty: false},
			diff:    domain.Change{Files: []string{"main.go"}, Diff: "diff --git a/main.go b/main.go\n+new\n-old"},
			repo:    domain.RepoInfo{CurrentBranch: "main", IsClean: false},
		},
		&mockAI{text: "feat: add new feature"},
		prompts.NewCommitBuilder(),
	)

	result, err := svc.Generate(context.Background(), CommitModeStaged)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if result.Message.Title != "feat: add new feature" {
		t.Errorf("title = %q", result.Message.Title)
	}
	if result.Message.Style != "conventional" {
		t.Errorf("style = %q", result.Message.Style)
	}
}

func TestCommitService_Generate_NoStagedChanges(t *testing.T) {
	svc := NewCommitService(
		&mockGit{
			status: git.StagedChanges{IsEmpty: true},
		},
		&mockAI{},
		prompts.NewCommitBuilder(),
	)

	_, err := svc.Generate(context.Background(), CommitModeStaged)
	if !errors.Is(err, domain.ErrNoStagedChanges) {
		t.Errorf("expected ErrNoStagedChanges, got %v", err)
	}
}

func TestCommitService_Generate_ParseResponse(t *testing.T) {
	tests := []struct {
		name     string
		aiText   string
		want     string
		wantBody string
	}{
		{
			name:     "simple title",
			aiText:   "feat: add login",
			want:     "feat: add login",
			wantBody: "",
		},
		{
			name:     "title and body",
			aiText:   "feat(auth): add login\n\n- Added OAuth provider\n- Added token validation",
			want:     "feat(auth): add login",
			wantBody: "- Added OAuth provider\n- Added token validation",
		},
		{
			name:     "wrapped in markdown",
			aiText:   "```\nfeat: add login\n```",
			want:     "feat: add login",
			wantBody: "",
		},
		{
			name:     "with quotes",
			aiText:   "\"feat: add login\"",
			want:     "feat: add login",
			wantBody: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewCommitService(
				&mockGit{
					status: git.StagedChanges{Files: []string{"x.go"}, IsEmpty: false},
					diff:   domain.Change{Files: []string{"x.go"}, Diff: "diff --git a/x.go b/x.go\n"},
					repo:   domain.RepoInfo{CurrentBranch: "main"},
				},
				&mockAI{text: tt.aiText},
				prompts.NewCommitBuilder(),
			)

			result, err := svc.Generate(context.Background(), CommitModeStaged)
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}

			if result.Message.Title != tt.want {
				t.Errorf("title = %q, want %q", result.Message.Title, tt.want)
			}
			if result.Message.Body != tt.wantBody {
				t.Errorf("body = %q, want %q", result.Message.Body, tt.wantBody)
			}
		})
	}
}

func TestCleanResponse(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"  spaced  ", "spaced"},
		{"```\ncontent\n```", "content"},
		{"```\ncode block\n```\ntrailing", "code block"},
		{`"quoted text"`, "quoted text"},
		{"```go\npackage main\n```", "package main"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := cleanResponse(tt.input)
			if got != tt.want {
				t.Errorf("cleanResponse(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCut(t *testing.T) {
	before, after, found := cut("a\n\nb", "\n\n")
	if !found || before != "a" || after != "b" {
		t.Errorf("cut(a\\n\\nb) = (%q, %q, %v)", before, after, found)
	}

	before, after, found = cut("no-separator", "\n\n")
	if found {
		t.Errorf("expected not found")
	}
}

// --- Mocks ---

type mockGit struct {
	status git.StagedChanges
	diff   domain.Change
	repo   domain.RepoInfo
	log    []git.CommitLog
	tags   []string
	err    error
}

func (m *mockGit) DiffCached(_ context.Context) (domain.Change, error) { return m.diff, m.err }
func (m *mockGit) DiffUnstaged(_ context.Context) (domain.Change, error) { return m.diff, m.err }
func (m *mockGit) Diff(_ context.Context, _ string) (domain.Change, error) { return m.diff, m.err }
func (m *mockGit) Log(_ context.Context, _, _ string) ([]git.CommitLog, error) { return m.log, m.err }
func (m *mockGit) Commit(_ context.Context, _ domain.CommitMessage) error { return m.err }
func (m *mockGit) Status(_ context.Context) (git.StagedChanges, error) { return m.status, m.err }
func (m *mockGit) UnstagedStatus(_ context.Context) (git.StagedChanges, error) { return m.status, m.err }
func (m *mockGit) Tags(_ context.Context) ([]string, error) { return m.tags, m.err }
func (m *mockGit) RepoInfo(_ context.Context) (domain.RepoInfo, error) { return m.repo, m.err }
func (m *mockGit) Stage(_ context.Context, _ []string) error { return m.err }
func (m *mockGit) UnstageAll(_ context.Context) error { return m.err }

type mockAI struct {
	text string
	err  error
}

func (m *mockAI) Name() string { return "mock" }
func (m *mockAI) Generate(_ context.Context, req ai.Request) (ai.Response, error) {
	if m.err != nil {
		return ai.Response{}, m.err
	}
	return ai.Response{
		Text: m.text,
		Usage: ai.TokenUsage{
			InputTokens:  50,
			OutputTokens: 10,
			TotalTokens:  60,
		},
	}, nil
}
