package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/user/gitx/internal/domain"
)

// --- Unit tests for parsing helpers ---

func TestParseFilesFromDiff(t *testing.T) {
	diff := `diff --git a/main.go b/main.go
index abc..def 100644
--- a/main.go
+++ b/main.go
@@ -1 +1 @@
-foo
+bar
diff --git a/auth/auth.go b/auth/auth.go
new file mode 100644
index 000..abc
--- /dev/null
+++ b/auth/auth.go
@@ -0,0 +1 @@
+package auth
`

	files := parseFilesFromDiff(diff)
	expected := []string{"main.go", "auth/auth.go"}

	if len(files) != len(expected) {
		t.Fatalf("got %d files, want %d: %v", len(files), len(expected), files)
	}
	for i, f := range files {
		if f != expected[i] {
			t.Errorf("file[%d] = %q, want %q", i, f, expected[i])
		}
	}
}

func TestParseFilesFromDiffEmpty(t *testing.T) {
	files := parseFilesFromDiff("")
	if len(files) != 0 {
		t.Errorf("expected no files, got %v", files)
	}
}

func TestParseFilesFromDiffNoChanges(t *testing.T) {
	files := parseFilesFromDiff("no diff lines here")
	if len(files) != 0 {
		t.Errorf("expected no files, got %v", files)
	}
}

func TestParseCommitLog(t *testing.T) {
	layout := "2006-01-02 15:04:05 -0700"
	input := `abc123|Alice|2025-06-01 10:00:00 +0000|feat: add login
def456|Bob|2025-06-02 12:30:00 +0000|fix: resolve timeout
`

	logs := parseCommitLog(input)
	if len(logs) != 2 {
		t.Fatalf("got %d commits, want 2", len(logs))
	}

	if logs[0].Hash != "abc123" {
		t.Errorf("hash = %q, want %q", logs[0].Hash, "abc123")
	}
	if logs[0].Author != "Alice" {
		t.Errorf("author = %q, want %q", logs[0].Author, "Alice")
	}
	if logs[0].Message != "feat: add login" {
		t.Errorf("message = %q, want %q", logs[0].Message, "feat: add login")
	}
	expectedTime, _ := time.Parse(layout, "2025-06-01 10:00:00 +0000")
	if !logs[0].Date.Equal(expectedTime) {
		t.Errorf("date = %v, want %v", logs[0].Date, expectedTime)
	}

	if logs[1].Message != "fix: resolve timeout" {
		t.Errorf("message = %q, want %q", logs[1].Message, "fix: resolve timeout")
	}
}

func TestParseCommitLogMalformedLine(t *testing.T) {
	input := "abc123|Alice|2025-06-01\n"
	logs := parseCommitLog(input)
	// Too few pipe-delimited parts → skipped
	if len(logs) != 0 {
		t.Errorf("expected 0 logs for malformed line, got %d", len(logs))
	}
}

func TestParseCommitLogEmpty(t *testing.T) {
	logs := parseCommitLog("")
	if len(logs) != 0 {
		t.Errorf("expected 0 logs, got %d", len(logs))
	}
}

func TestParseCommitLogInvalidDate(t *testing.T) {
	input := `abc123|Alice|not-a-date|feat: test
`
	logs := parseCommitLog(input)
	if len(logs) != 1 {
		t.Fatalf("got %d commits, want 1", len(logs))
	}
	if !logs[0].Date.IsZero() {
		t.Errorf("expected zero date for invalid input, got %v", logs[0].Date)
	}
}

// --- Integration tests (require git) ---

// gitInit creates a temporary git repository and returns the path.
func gitInit(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}

	// Configure user for commits in this repo
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

// gitWriteFile creates a file in the git repo and stages it.
func gitWriteFile(t *testing.T, dir, path, content string) {
	t.Helper()

	fullPath := filepath.Join(dir, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

// gitAdd stages all files.
func gitAdd(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
}

// gitCommit creates a commit.
func gitCommit(t *testing.T, dir, msg string) {
	t.Helper()
	cmd := exec.Command("git", "commit", "-m", msg)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
}

// gitTag creates a tag.
func gitTag(t *testing.T, dir, tag string) {
	t.Helper()
	cmd := exec.Command("git", "tag", tag)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git tag: %v\n%s", err, out)
	}
}

func TestExecClient_DiffCached(t *testing.T) {
	dir := gitInit(t)
	client := NewExecClient(dir)
	ctx := context.Background()

	// No staged changes yet
	change, err := client.DiffCached(ctx)
	if err != nil {
		t.Fatalf("DiffCached on empty repo: %v", err)
	}
	if !change.IsEmpty() {
		t.Errorf("expected empty change, got files=%v diff=%q", change.Files, change.Diff[:min(50, len(change.Diff))])
	}

	// Create and stage a file
	gitWriteFile(t, dir, "main.go", "package main\n\nfunc main() {}\n")
	gitAdd(t, dir)

	change, err = client.DiffCached(ctx)
	if err != nil {
		t.Fatalf("DiffCached after staging: %v", err)
	}
	if change.IsEmpty() {
		t.Fatal("expected non-empty change after staging")
	}
	if len(change.Files) != 1 || change.Files[0] != "main.go" {
		t.Errorf("files = %v, want [main.go]", change.Files)
	}
	if change.DiffStat == "" {
		t.Error("expected non-empty diff stat")
	}
}

func TestExecClient_Status(t *testing.T) {
	dir := gitInit(t)
	client := NewExecClient(dir)
	ctx := context.Background()

	// No staged changes
	status, err := client.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !status.IsEmpty {
		t.Error("expected empty status")
	}

	// Stage a file
	gitWriteFile(t, dir, "auth.go", "package auth\n")
	gitAdd(t, dir)

	status, err = client.Status(ctx)
	if err != nil {
		t.Fatalf("Status after staging: %v", err)
	}
	if status.IsEmpty {
		t.Fatal("expected non-empty status after staging")
	}
	if len(status.Files) != 1 || status.Files[0] != "auth.go" {
		t.Errorf("files = %v, want [auth.go]", status.Files)
	}
}

func TestExecClient_Commit(t *testing.T) {
	dir := gitInit(t)
	client := NewExecClient(dir)
	ctx := context.Background()

	// Create and stage a file
	gitWriteFile(t, dir, "main.go", "package main\n")
	gitAdd(t, dir)

	// Commit
	err := client.Commit(ctx, domain.CommitMessage{
		Title: "feat: initial commit",
		Body:  "Add main.go",
		Style: "conventional",
	})
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Verify the commit exists
	status, _ := client.Status(ctx)
	if !status.IsEmpty {
		t.Error("expected clean status after commit")
	}
}

func TestExecClient_CommitEmptyTitle(t *testing.T) {
	client := NewExecClient(t.TempDir())
	err := client.Commit(context.Background(), domain.CommitMessage{Title: ""})
	if err == nil {
		t.Fatal("expected error for empty commit title")
	}
}

func TestExecClient_Log(t *testing.T) {
	dir := gitInit(t)
	client := NewExecClient(dir)
	ctx := context.Background()

	// Create initial commit on main
	gitWriteFile(t, dir, "a.go", "package a\n")
	gitAdd(t, dir)
	gitCommit(t, dir, "feat: add a")
	gitTag(t, dir, "v1.0.0")

	// Second commit
	gitWriteFile(t, dir, "b.go", "package b\n")
	gitAdd(t, dir)
	gitCommit(t, dir, "feat: add b")

	// Log from v1.0.0 to HEAD
	logs, err := client.Log(ctx, "v1.0.0", "HEAD")
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(logs))
	}
	if logs[0].Message != "feat: add b" {
		t.Errorf("message = %q, want %q", logs[0].Message, "feat: add b")
	}
	if logs[0].Author != "GitX Test" {
		t.Errorf("author = %q, want %q", logs[0].Author, "GitX Test")
	}
	if logs[0].Hash == "" {
		t.Error("expected non-empty hash")
	}
	if logs[0].Date.IsZero() {
		t.Error("expected non-zero date")
	}
}

func TestExecClient_Tags(t *testing.T) {
	dir := gitInit(t)
	client := NewExecClient(dir)
	ctx := context.Background()

	// No tags yet
	tags, err := client.Tags(ctx)
	if err != nil {
		t.Fatalf("Tags: %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("expected no tags, got %v", tags)
	}

	// Create a commit and tag it
	gitWriteFile(t, dir, "main.go", "package main\n")
	gitAdd(t, dir)
	gitCommit(t, dir, "initial")
	gitTag(t, dir, "v1.0.0")
	gitTag(t, dir, "v2.0.0")

	tags, err = client.Tags(ctx)
	if err != nil {
		t.Fatalf("Tags: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %d: %v", len(tags), tags)
	}
}

func TestExecClient_RepoInfo(t *testing.T) {
	dir := gitInit(t)
	client := NewExecClient(dir)
	ctx := context.Background()

	// Need at least one commit for HEAD to resolve
	gitWriteFile(t, dir, "main.go", "package main\n")
	gitAdd(t, dir)
	gitCommit(t, dir, "initial")

	info, err := client.RepoInfo(ctx)
	if err != nil {
		t.Fatalf("RepoInfo: %v", err)
	}

	if info.CurrentBranch != "main" && info.CurrentBranch != "master" {
		t.Errorf("unexpected branch %q", info.CurrentBranch)
	}
	if !info.IsClean {
		t.Error("expected clean repo")
	}
}

func TestExecClient_UnstageAll_NothingStaged(t *testing.T) {
	dir := gitInit(t)
	client := NewExecClient(dir)
	ctx := context.Background()

	// No commits, no staged files — UnstageAll should be a no-op.
	// This previously failed with "pathspec '.' did not match any files".
	err := client.UnstageAll(ctx)
	if err != nil {
		t.Fatalf("UnstageAll on empty repo (no commits, nothing staged): %v", err)
	}
}

func TestExecClient_UnstageAll_StagedWithoutCommit(t *testing.T) {
	dir := gitInit(t)
	client := NewExecClient(dir)
	ctx := context.Background()

	// Stage files without making a commit (no HEAD yet)
	gitWriteFile(t, dir, "main.go", "package main\n")
	gitAdd(t, dir)

	// Verify files are staged
	status, err := client.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.IsEmpty {
		t.Fatal("expected files to be staged")
	}

	// Unstage them
	err = client.UnstageAll(ctx)
	if err != nil {
		t.Fatalf("UnstageAll with staged files (no HEAD): %v", err)
	}

	// Verify files are unstaged
	status, err = client.Status(ctx)
	if err != nil {
		t.Fatalf("Status after unstage: %v", err)
	}
	if !status.IsEmpty {
		t.Errorf("expected clean status after UnstageAll, got %v", status.Files)
	}
}

func TestExecClient_UnstageAll_StagedWithCommit(t *testing.T) {
	dir := gitInit(t)
	client := NewExecClient(dir)
	ctx := context.Background()

	// Make an initial commit (creates HEAD)
	gitWriteFile(t, dir, "a.go", "package a\n")
	gitAdd(t, dir)
	gitCommit(t, dir, "initial")

	// Make more changes and stage them
	gitWriteFile(t, dir, "b.go", "package b\n")
	gitAdd(t, dir)

	status, err := client.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.IsEmpty {
		t.Fatal("expected files to be staged")
	}

	// Unstage via HEAD reset
	err = client.UnstageAll(ctx)
	if err != nil {
		t.Fatalf("UnstageAll with HEAD: %v", err)
	}

	status, err = client.Status(ctx)
	if err != nil {
		t.Fatalf("Status after unstage: %v", err)
	}
	if !status.IsEmpty {
		t.Errorf("expected clean status after UnstageAll, got %v", status.Files)
	}
}

func TestExecClient_Stage(t *testing.T) {
	dir := gitInit(t)
	client := NewExecClient(dir)
	ctx := context.Background()

	gitWriteFile(t, dir, "a.go", "package a\n")
	gitWriteFile(t, dir, "sub/b.go", "package b\n")

	// Stage specific files
	err := client.Stage(ctx, []string{"a.go"})
	if err != nil {
		t.Fatalf("Stage: %v", err)
	}

	status, err := client.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(status.Files) != 1 || status.Files[0] != "a.go" {
		t.Errorf("expected only a.go to be staged, got %v", status.Files)
	}

	// Stage additional files
	err = client.Stage(ctx, []string{"sub/b.go"})
	if err != nil {
		t.Fatalf("Stage second file: %v", err)
	}

	status, err = client.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(status.Files) != 2 {
		t.Errorf("expected 2 staged files, got %v", status.Files)
	}
}

func TestExecClient_Stage_EmptyFiles(t *testing.T) {
	dir := gitInit(t)
	client := NewExecClient(dir)
	ctx := context.Background()

	// Stage with empty file list should be a no-op
	err := client.Stage(ctx, []string{})
	if err != nil {
		t.Fatalf("Stage with empty list: %v", err)
	}
}

func TestExecClient_Diff(t *testing.T) {
	dir := gitInit(t)
	client := NewExecClient(dir)
	ctx := context.Background()

	// Initial commit on main
	gitWriteFile(t, dir, "main.go", "package main\n")
	gitAdd(t, dir)
	gitCommit(t, dir, "initial")

	// Create a new branch and make changes
	exec.Command("git", "-C", dir, "checkout", "-b", "feature").Run()
	gitWriteFile(t, dir, "feature.go", "package feature\n")
	gitAdd(t, dir)
	gitCommit(t, dir, "feat: add feature")

	// Diff feature against main
	change, err := client.Diff(ctx, "main")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if change.IsEmpty() {
		t.Fatal("expected non-empty diff between branches")
	}
	// Could also be "feature.go" if git diff main...HEAD shows the right files
	if len(change.Files) < 1 {
		t.Errorf("expected at least 1 file, got %v", change.Files)
	}
}

func TestExecClient_Status_UnstagedOnly(t *testing.T) {
	dir := gitInit(t)
	client := NewExecClient(dir)
	ctx := context.Background()

	// Create a commit so we have HEAD
	gitWriteFile(t, dir, "main.go", "package main\n")
	gitAdd(t, dir)
	gitCommit(t, dir, "initial")

	// Modify the file WITHOUT staging the change — should be " M main.go"
	gitWriteFile(t, dir, "main.go", "package main\n\nfunc main() {}\n")

	// Status should NOT see it as staged (only column 2 is modified)
	status, err := client.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !status.IsEmpty {
		t.Errorf("expected no staged files, got %v", status.Files)
	}

	// UnstagedStatus SHOULD see it
	unstaged, err := client.UnstagedStatus(ctx)
	if err != nil {
		t.Fatalf("UnstagedStatus: %v", err)
	}
	if unstaged.IsEmpty {
		t.Fatal("expected unstaged changes after file modification")
	}
	if len(unstaged.Files) != 1 || unstaged.Files[0] != "main.go" {
		t.Errorf("unstaged files = %v, want [main.go]", unstaged.Files)
	}
}

func TestExecClient_Status_MixedStagedAndUnstaged(t *testing.T) {
	dir := gitInit(t)
	client := NewExecClient(dir)
	ctx := context.Background()

	// Create a commit so we have HEAD
	gitWriteFile(t, dir, "a.go", "package a\n")
	gitAdd(t, dir)
	gitCommit(t, dir, "initial")

	// Stage a change to a.go — "M  a.go"
	gitWriteFile(t, dir, "a.go", "package a\n\nfunc A() {}\n")
	gitAdd(t, dir)

	// Modify a.go again without staging — creates "MM a.go" (staged + unstaged)
	gitWriteFile(t, dir, "a.go", "package a\n\n// comment\nfunc A() {}\n")

	// Status: MM is counted as staged (first column is M)
	status, err := client.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.IsEmpty {
		t.Fatal("expected staged files for MM entry")
	}
	if len(status.Files) != 1 || status.Files[0] != "a.go" {
		t.Fatalf("expected [a.go], got %v", status.Files)
	}

	// UnstagedStatus: MM is also counted as unstaged (second column is M)
	unstaged, err := client.UnstagedStatus(ctx)
	if err != nil {
		t.Fatalf("UnstagedStatus: %v", err)
	}
	if unstaged.IsEmpty {
		t.Fatal("expected unstaged changes for MM entry")
	}
	if len(unstaged.Files) != 1 || unstaged.Files[0] != "a.go" {
		t.Errorf("unstaged files = %v, want [a.go]", unstaged.Files)
	}
}

func TestExecClient_StageAll(t *testing.T) {
	dir := gitInit(t)
	client := NewExecClient(dir)
	ctx := context.Background()

	// Create a commit so we have HEAD
	gitWriteFile(t, dir, "main.go", "package main\n")
	gitAdd(t, dir)
	gitCommit(t, dir, "initial")

	// Modify tracked file and add untracked file
	gitWriteFile(t, dir, "main.go", "package main\n\nfunc main() {}\n")
	gitWriteFile(t, dir, "new.go", "package new\n")

	// Nothing staged yet
	status, err := client.Status(ctx)
	if err != nil {
		t.Fatalf("Status before StageAll: %v", err)
	}
	if !status.IsEmpty {
		t.Fatal("expected clean status before StageAll")
	}

	// StageAll should stage both the tracked modification and the untracked file
	err = client.StageAll(ctx)
	if err != nil {
		t.Fatalf("StageAll: %v", err)
	}

	status, err = client.Status(ctx)
	if err != nil {
		t.Fatalf("Status after StageAll: %v", err)
	}
	if status.IsEmpty {
		t.Fatal("expected files to be staged after StageAll")
	}
	if len(status.Files) != 2 {
		t.Fatalf("expected 2 staged files, got %v", status.Files)
	}
}

func TestExecClient_UnstagedStatus_Untracked(t *testing.T) {
	dir := gitInit(t)
	client := NewExecClient(dir)
	ctx := context.Background()

	// Create a commit so we have HEAD
	gitWriteFile(t, dir, "main.go", "package main\n")
	gitAdd(t, dir)
	gitCommit(t, dir, "initial")

	// Add an untracked file
	gitWriteFile(t, dir, "new.go", "package new\n")

	// UnstagedStatus should include the untracked file
	unstaged, err := client.UnstagedStatus(ctx)
	if err != nil {
		t.Fatalf("UnstagedStatus: %v", err)
	}
	if unstaged.IsEmpty {
		t.Fatal("expected untracked file in unstaged status")
	}
	if len(unstaged.Files) != 1 || unstaged.Files[0] != "new.go" {
		t.Errorf("unstaged files = %v, want [new.go]", unstaged.Files)
	}
}
