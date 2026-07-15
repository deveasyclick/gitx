package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/user/gitx/internal/ai"
	"github.com/user/gitx/internal/domain"
	"github.com/user/gitx/internal/git"
	"github.com/user/gitx/internal/prompts"
	"github.com/user/gitx/internal/services"
)

// runGit executes a git command in the given directory.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// gitInit creates a temporary git repository and returns its path.
func gitInit(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, arg := range []struct{ key, value string }{
		{"user.email", "test@gitx.dev"},
		{"user.name", "GitX Test"},
	} {
		cmd := exec.Command("git", "config", arg.key, arg.value)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git config %s: %v\n%s", arg.key, err, out)
		}
	}
	return dir
}

// writeFile creates a file in the given directory.
func writeFile(t *testing.T, dir, path, content string) {
	t.Helper()
	full := filepath.Join(dir, path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// gitAdd stages all files in the repo.
func gitAdd(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
}

// startMockAI creates a test server that returns a canned AI response.
func startMockAI(t *testing.T, responseText string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify it's a chat completions request
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("unexpected method: %s", r.Method)
		}

		// Return canned response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "test-cmpl",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": responseText,
					},
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     50,
				"completion_tokens": 10,
				"total_tokens":      60,
			},
		})
	}))
}

func TestCommitIntegration_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Set up temp git repo
	repoDir := gitInit(t)

	// Create initial commit so HEAD exists
	writeFile(t, repoDir, "README.md", "# Project\n")
	gitAdd(t, repoDir)
	runGit(t, repoDir, "commit", "-m", "initial commit")

	// Now make real staged changes
	writeFile(t, repoDir, "auth.go", `package auth

func Login() string {
    return "token"
}
`)
	gitAdd(t, repoDir)

	mockProvider := &mockAIProvider{
		name: "openai",
		text: "feat(auth): add login function\n\n- Added Login() to auth package",
	}

	gitClient := git.NewExecClient(repoDir)
	svc := services.NewCommitService(gitClient, mockProvider, prompts.NewCommitBuilder())

	ctx := context.Background()
	result, err := svc.Generate(ctx, services.CommitModeStaged)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if result.Message.Title != "feat(auth): add login function" {
		t.Errorf("title = %q, want %q", result.Message.Title, "feat(auth): add login function")
	}
	if result.Message.Body == "" {
		t.Error("expected non-empty body")
	}
	if result.Provider != "openai" {
		t.Errorf("provider = %q", result.Provider)
	}
	if result.InputTokens <= 0 {
		t.Errorf("expected positive input tokens, got %d", result.InputTokens)
	}
}

func TestCommitIntegration_MockServerDirect(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	repoDir := gitInit(t)

	// Create initial commit so HEAD exists
	writeFile(t, repoDir, "README.md", "# Project\n")
	gitAdd(t, repoDir)
	runGit(t, repoDir, "commit", "-m", "initial")

	// Make staged changes
	writeFile(t, repoDir, "main.go", `package main
func main() {}
`)
	gitAdd(t, repoDir)

	mockProvider := &mockAIProvider{
		name: "openai",
		text: "feat: initial commit",
	}

	gitClient := git.NewExecClient(repoDir)
	svc := services.NewCommitService(gitClient, mockProvider, prompts.NewCommitBuilder())

	result, err := svc.Generate(context.Background(), services.CommitModeStaged)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if result.Message.Title != "feat: initial commit" {
		t.Errorf("title = %q", result.Message.Title)
	}
}

func TestPRIntegration_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	repoDir := gitInit(t)

	// Create initial commit on main
	writeFile(t, repoDir, "README.md", "# Project\n")
	gitAdd(t, repoDir)
	runGit(t, repoDir, "commit", "-m", "initial commit")

	// Create a feature branch with changes
	runGit(t, repoDir, "checkout", "-b", "feat/payment")
	writeFile(t, repoDir, "payment.go", `package payment
func Process() {}
`)
	gitAdd(t, repoDir)
	runGit(t, repoDir, "commit", "-m", "feat: add payment processing")

	mockProvider := &mockAIProvider{
		name: "openai",
		text: "## Summary\nAdds payment processing.\n\n## Changes\n- Added Process() function\n\n## Testing\nManual testing\n\n## Risks\nNone\n\n## Breaking Changes\nNone",
	}

	gitClient := git.NewExecClient(repoDir)
	svc := services.NewPRService(gitClient, mockProvider, prompts.NewPRBuilder())

	result, err := svc.Generate(context.Background(), "main")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if result.Description.Summary != "Adds payment processing." {
		t.Errorf("Summary = %q", result.Description.Summary)
	}
	if len(result.Description.Changes) != 1 {
		t.Errorf("Changes = %v", result.Description.Changes)
	}
}

func TestChangelogIntegration_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	repoDir := gitInit(t)

	// Create commits with tags
	writeFile(t, repoDir, "a.go", "package a\n")
	gitAdd(t, repoDir)
	runGit(t, repoDir, "commit", "-m", "initial")
	runGit(t, repoDir, "tag", "v1.0.0")

	writeFile(t, repoDir, "b.go", "package b\n")
	gitAdd(t, repoDir)
	runGit(t, repoDir, "commit", "-m", "feat: add feature b")

	writeFile(t, repoDir, "c.go", "package c\n")
	gitAdd(t, repoDir)
	runGit(t, repoDir, "commit", "-m", "fix: resolve issue")
	runGit(t, repoDir, "tag", "v2.0.0")

	mockProvider := &mockAIProvider{
		name: "openai",
		text: "## Added\n- Feature b\n\n## Fixed\n- Resolved issue",
	}

	gitClient := git.NewExecClient(repoDir)
	svc := services.NewChangelogService(gitClient, mockProvider, prompts.NewChangelogBuilder())

	result, err := svc.GenerateRange(context.Background(), "v1.0.0", "v2.0.0")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result.Entries))
	}
	entry := result.Entries[0]
	if entry.Version != "v2.0.0" {
		t.Errorf("Version = %q", entry.Version)
	}
	if len(entry.Added) < 1 {
		t.Errorf("expected Added items, got %v", entry.Added)
	}
}

// TestCommitIntegration_NoStagedChanges verifies the error path.
func TestCommitIntegration_NoStagedChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	repoDir := gitInit(t)
	// No files staged

	gitClient := git.NewExecClient(repoDir)
	svc := services.NewCommitService(gitClient, &mockAIProvider{}, prompts.NewCommitBuilder())

	_, err := svc.Generate(context.Background(), services.CommitModeStaged)
	if err == nil {
		t.Fatal("expected error for no staged changes")
	}
}

// TestCommitIntegration_UnstagedMode verifies that --unstaged correctly unstages
// previously staged files and stages only the originally-unstaged files.
func TestCommitIntegration_UnstagedMode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	repoDir := gitInit(t)

	// Initial commit so HEAD exists
	writeFile(t, repoDir, "README.md", "# Project\n")
	gitAdd(t, repoDir)
	runGit(t, repoDir, "commit", "-m", "initial commit")

	// Create two tracked files
	writeFile(t, repoDir, "staged.go", `package main

const Staged = "staged"
`)
	writeFile(t, repoDir, "unstaged.go", `package main

const Unstaged = "unstaged"
`)
	gitAdd(t, repoDir)
	runGit(t, repoDir, "commit", "-m", "add staged.go and unstaged.go")

	// Modify staged.go and stage it
	writeFile(t, repoDir, "staged.go", `package main

const Staged = "staged-modified"
`)
	runGit(t, repoDir, "add", "staged.go")

	// Modify unstaged.go but do NOT stage it
	writeFile(t, repoDir, "unstaged.go", `package main

const Unstaged = "unstaged-modified"
`)

	gitClient := git.NewExecClient(repoDir)
	ctx := context.Background()

	// Verify initial state: staged.go is staged, unstaged.go is unstaged
	status, err := gitClient.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(status.Files) != 1 || status.Files[0] != "staged.go" {
		t.Fatalf("expected [staged.go] staged, got %v", status.Files)
	}

	unstagedStatus, err := gitClient.UnstagedStatus(ctx)
	if err != nil {
		t.Fatalf("UnstagedStatus: %v", err)
	}
	if len(unstagedStatus.Files) != 1 || unstagedStatus.Files[0] != "unstaged.go" {
		t.Fatalf("expected [unstaged.go] unstaged, got %v", unstagedStatus.Files)
	}

	// Execute the fixed logic from ensureStagedForCommit (CommitModeUnstaged):
	//   1. Capture unstaged files BEFORE unstaging
	//   2. Unstage everything
	//   3. Stage only the originally-unstaged files
	unstaged, err := gitClient.UnstagedStatus(ctx)
	if err != nil {
		t.Fatalf("UnstagedStatus (capture): %v", err)
	}

	if err := gitClient.UnstageAll(ctx); err != nil {
		t.Fatalf("UnstageAll: %v", err)
	}

	if len(unstaged.Files) > 0 {
		if err := gitClient.Stage(ctx, unstaged.Files); err != nil {
			t.Fatalf("Stage: %v", err)
		}
	}

	// Verify: only unstaged.go should be staged now; staged.go should NOT be staged
	status, err = gitClient.Status(ctx)
	if err != nil {
		t.Fatalf("Status after: %v", err)
	}
	if len(status.Files) != 1 || status.Files[0] != "unstaged.go" {
		t.Fatalf("expected [unstaged.go] staged after fix, got %v", status.Files)
	}

	// Verify the commit only contains the unstaged change
	commitMsg := "test: add unstaged file"
	if err := gitClient.Commit(ctx, domain.CommitMessage{
		Title: commitMsg,
		Body:  "",
		Style: "conventional",
	}); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Verify only unstaged.go is in the commit
	diff, err := gitClient.Diff(ctx, "HEAD~1")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if len(diff.Files) != 1 || diff.Files[0] != "unstaged.go" {
		t.Fatalf("expected commit to contain [unstaged.go], got %v", diff.Files)
	}
}

// TestCommitIntegration_UnstagedMode_WithStagedOnly verifies that when there are
// only staged changes (no unstaged), --unstaged unstages everything and leaves
// nothing staged (no changes to commit).
func TestCommitIntegration_UnstagedMode_WithStagedOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	repoDir := gitInit(t)

	// Initial commit so HEAD exists
	writeFile(t, repoDir, "README.md", "# Project\n")
	gitAdd(t, repoDir)
	runGit(t, repoDir, "commit", "-m", "initial commit")

	// Create a file and stage it
	writeFile(t, repoDir, "only-staged.go", `package main

const OnlyStaged = "staged"
`)
	gitAdd(t, repoDir)

	gitClient := git.NewExecClient(repoDir)
	ctx := context.Background()

	// Verify only-staged.go is staged
	status, err := gitClient.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(status.Files) != 1 || status.Files[0] != "only-staged.go" {
		t.Fatalf("expected [only-staged.go] staged, got %v", status.Files)
	}

	// Verify no unstaged changes
	unstaged, err := gitClient.UnstagedStatus(ctx)
	if err != nil {
		t.Fatalf("UnstagedStatus: %v", err)
	}
	if !unstaged.IsEmpty {
		t.Fatalf("expected no unstaged files, got %v", unstaged.Files)
	}

	// Execute the fixed logic: capture unstaged (empty), unstage all, stage nothing
	if err := gitClient.UnstageAll(ctx); err != nil {
		t.Fatalf("UnstageAll: %v", err)
	}

	// Nothing to re-stage since there were no unstaged files

	// Verify staging area is now empty
	status, err = gitClient.Status(ctx)
	if err != nil {
		t.Fatalf("Status after: %v", err)
	}
	if !status.IsEmpty {
		t.Fatalf("expected empty staging area after unstage, got %v", status.Files)
	}
}

// TestCommitIntegration_UnstagedMode_WithBothStagedAndUnstagedSameFile tests the
// edge case where a file has BOTH staged and unstaged changes ("MM" in git status).
// The unstaged portion should still be captured and re-staged.
func TestCommitIntegration_UnstagedMode_WithBothStagedAndUnstagedSameFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	repoDir := gitInit(t)

	// Initial commit so HEAD exists
	writeFile(t, repoDir, "README.md", "# Project\n")
	gitAdd(t, repoDir)
	runGit(t, repoDir, "commit", "-m", "initial commit")

	// Create a file and commit it
	writeFile(t, repoDir, "shared.go", `package main

const Value = "v1"
`)
	gitAdd(t, repoDir)
	runGit(t, repoDir, "commit", "-m", "add shared.go")

	// Modify shared.go and stage it (first modification)
	writeFile(t, repoDir, "shared.go", `package main

const Value = "v2"
`)
	runGit(t, repoDir, "add", "shared.go")

	// Modify shared.go again WITHOUT staging (second modification)
	writeFile(t, repoDir, "shared.go", `package main

const Value = "v3"
`)

	gitClient := git.NewExecClient(repoDir)
	ctx := context.Background()

	// Verify: shared.go has both staged (v2) and unstaged (v3) changes
	status, err := gitClient.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(status.Files) != 1 || status.Files[0] != "shared.go" {
		t.Fatalf("expected [shared.go] staged, got %v", status.Files)
	}

	unstaged, err := gitClient.UnstagedStatus(ctx)
	if err != nil {
		t.Fatalf("UnstagedStatus: %v", err)
	}
	if len(unstaged.Files) != 1 || unstaged.Files[0] != "shared.go" {
		t.Fatalf("expected [shared.go] in unstaged, got %v", unstaged.Files)
	}

	// Execute the fixed logic
	captured, err := gitClient.UnstagedStatus(ctx)
	if err != nil {
		t.Fatalf("UnstagedStatus (capture): %v", err)
	}

	if err := gitClient.UnstageAll(ctx); err != nil {
		t.Fatalf("UnstageAll: %v", err)
	}

	if len(captured.Files) > 0 {
		if err := gitClient.Stage(ctx, captured.Files); err != nil {
			t.Fatalf("Stage: %v", err)
		}
	}

	// Verify: shared.go is still staged (it had unstaged changes, so it should be re-staged)
	status, err = gitClient.Status(ctx)
	if err != nil {
		t.Fatalf("Status after: %v", err)
	}
	if len(status.Files) != 1 || status.Files[0] != "shared.go" {
		t.Fatalf("expected [shared.go] staged after fix (it had unstaged changes), got %v", status.Files)
	}

	// Verify the committed content is v3 (the latest working-tree version, not the old staged v2)
	if err := gitClient.Commit(ctx, domain.CommitMessage{
		Title: "test: update shared.go",
		Body:  "",
		Style: "conventional",
	}); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Read the committed file content
	showCmd := exec.Command("git", "show", "HEAD:shared.go")
	showCmd.Dir = repoDir
	out, err := showCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git show: %v", err)
	}
	if !strings.Contains(string(out), `"v3"`) {
		t.Fatalf("expected committed content to contain v3, got:\n%s", out)
	}
}

// mockAIProvider implements services.aiProvider for testing.
type mockAIProvider struct {
	name string
	text string
	err  error
}

func (m *mockAIProvider) Name() string {
	if m.name != "" {
		return m.name
	}
	return "mock"
}

func (m *mockAIProvider) Generate(_ context.Context, req ai.Request) (ai.Response, error) {
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
