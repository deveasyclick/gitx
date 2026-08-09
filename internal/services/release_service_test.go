package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/user/gitx/internal/domain"
	"github.com/user/gitx/internal/git"
	"github.com/user/gitx/internal/prompts"
)

// --- Mocks (reuse the mocks from commit_service_test.go) ---

func TestReleaseService_CompareBranches_Success(t *testing.T) {
	svc := NewReleaseService(
		&mockGit{
			repo: domain.RepoInfo{
				CurrentBranch: "feature/login",
				Remote:        "origin",
				IsClean:       true,
			},
			log: []git.CommitLog{
				{Hash: "abc1234", Message: "feat: add login flow"},
				{Hash: "def5678", Message: "fix: handle edge case"},
			},
			diff: domain.Change{
				Files:    []string{"login.go", "auth.go"},
				Diff:     "diff --git a/login.go b/login.go\n...",
				DiffStat: "2 files changed, 50 insertions(+)",
			},
			tags: []string{"v1.0.0", "v0.9.0"},
		},
		&mockAI{},
		prompts.NewReleaseBuilder(),
	)

	result, err := svc.CompareBranches(context.Background(), "main", "feature/login")
	if err != nil {
		t.Fatalf("CompareBranches: %v", err)
	}

	info := result.Info
	if info.BaseBranch != "main" {
		t.Errorf("BaseBranch = %q, want %q", info.BaseBranch, "main")
	}
	if info.HeadBranch != "feature/login" {
		t.Errorf("HeadBranch = %q, want %q", info.HeadBranch, "feature/login")
	}
	if info.CurrentVersion != "v1.0.0" {
		t.Errorf("CurrentVersion = %q, want %q", info.CurrentVersion, "v1.0.0")
	}
	if info.NextVersion == "" {
		t.Error("NextVersion should not be empty")
	}
	if len(info.Commits) != 2 {
		t.Errorf("expected 2 commits, got %d", len(info.Commits))
	}
	if info.Diff.DiffStat != "2 files changed, 50 insertions(+)" {
		t.Errorf("DiffStat = %q", info.Diff.DiffStat)
	}
}

func TestReleaseService_CompareBranches_NoTags(t *testing.T) {
	svc := NewReleaseService(
		&mockGit{
			repo: domain.RepoInfo{
				CurrentBranch: "feature/init",
			},
			log: []git.CommitLog{
				{Hash: "abc123", Message: "feat: initial commit"},
			},
			diff: domain.Change{
				Files:    []string{"main.go"},
				DiffStat: "1 file changed, 100 insertions(+)",
			},
			tags: []string{},
		},
		&mockAI{},
		prompts.NewReleaseBuilder(),
	)

	result, err := svc.CompareBranches(context.Background(), "main", "feature/init")
	if err != nil {
		t.Fatalf("CompareBranches: %v", err)
	}

	if result.Info.CurrentVersion != "" {
		t.Errorf("CurrentVersion = %q, want empty", result.Info.CurrentVersion)
	}
	if result.Info.NextVersion != "v0.1.0" {
		t.Errorf("NextVersion = %q, want %q", result.Info.NextVersion, "v0.1.0")
	}
}

func TestReleaseService_CompareBranches_DefaultHead(t *testing.T) {
	svc := NewReleaseService(
		&mockGit{
			repo: domain.RepoInfo{
				CurrentBranch: "feature/auth",
			},
			log: []git.CommitLog{
				{Hash: "abc123", Message: "feat: add auth"},
			},
			diff: domain.Change{
				Files: []string{"auth.go"},
			},
			tags: []string{"v1.0.0"},
		},
		&mockAI{},
		prompts.NewReleaseBuilder(),
	)

	// head = "" should default to current branch
	result, err := svc.CompareBranches(context.Background(), "main", "")
	if err != nil {
		t.Fatalf("CompareBranches: %v", err)
	}

	if result.Info.HeadBranch != "feature/auth" {
		t.Errorf("HeadBranch = %q, want %q", result.Info.HeadBranch, "feature/auth")
	}
}

func TestReleaseService_CompareBranches_SameBranch(t *testing.T) {
	svc := NewReleaseService(
		&mockGit{
			repo: domain.RepoInfo{
				CurrentBranch: "main",
			},
		},
		&mockAI{},
		prompts.NewReleaseBuilder(),
	)

	_, err := svc.CompareBranches(context.Background(), "main", "main")
	if err == nil {
		t.Fatal("expected error for same branch")
	}
}

func TestReleaseService_SuggestVersion_Major(t *testing.T) {
	svc := NewReleaseService(nil, &mockAI{text: "major"}, prompts.NewReleaseBuilder())
	bump, err := svc.SuggestVersion(context.Background(), []domain.CommitSummary{
		{Message: "feat: breaking change"},
	})
	if err != nil {
		t.Fatalf("SuggestVersion: %v", err)
	}
	if bump != domain.BumpMajor {
		t.Errorf("bump = %q, want %q", bump, domain.BumpMajor)
	}
}

func TestReleaseService_SuggestVersion_Minor(t *testing.T) {
	svc := NewReleaseService(nil, &mockAI{text: "minor"}, prompts.NewReleaseBuilder())
	bump, err := svc.SuggestVersion(context.Background(), []domain.CommitSummary{
		{Message: "feat: new feature"},
	})
	if err != nil {
		t.Fatalf("SuggestVersion: %v", err)
	}
	if bump != domain.BumpMinor {
		t.Errorf("bump = %q, want %q", bump, domain.BumpMinor)
	}
}

func TestReleaseService_SuggestVersion_Patch(t *testing.T) {
	svc := NewReleaseService(nil, &mockAI{text: "patch"}, prompts.NewReleaseBuilder())
	bump, err := svc.SuggestVersion(context.Background(), []domain.CommitSummary{
		{Message: "fix: bug fix"},
	})
	if err != nil {
		t.Fatalf("SuggestVersion: %v", err)
	}
	if bump != domain.BumpPatch {
		t.Errorf("bump = %q, want %q", bump, domain.BumpPatch)
	}
}

func TestReleaseService_SuggestVersion_Fallback(t *testing.T) {
	// When AI returns something unexpected, fallback to minor
	svc := NewReleaseService(nil, &mockAI{text: "unknown"}, prompts.NewReleaseBuilder())
	bump, err := svc.SuggestVersion(context.Background(), []domain.CommitSummary{
		{Message: "chore: cleanup"},
	})
	if err != nil {
		t.Fatalf("SuggestVersion: %v", err)
	}
	if bump != domain.BumpMinor {
		t.Errorf("bump = %q, want %q (fallback)", bump, domain.BumpMinor)
	}
}

func TestReleaseService_GenerateReleaseChangelog_Success(t *testing.T) {
	svc := NewReleaseService(
		nil,
		&mockAI{text: "### Added\n- Login flow\n\n### Fixed\n- Edge case bug"},
		prompts.NewReleaseBuilder(),
	)

	info := domain.ReleaseInfo{
		BaseBranch: "main",
		HeadBranch: "feature/login",
		Commits: []domain.CommitSummary{
			{Hash: "abc", Message: "feat: login"},
			{Hash: "def", Message: "fix: edge case"},
		},
	}

	changelog, err := svc.GenerateReleaseChangelog(context.Background(), info, "v1.1.0")
	if err != nil {
		t.Fatalf("GenerateReleaseChangelog: %v", err)
	}

	if !strings.Contains(changelog, "Login flow") {
		t.Errorf("changelog should contain 'Login flow', got: %s", changelog)
	}
	if !strings.Contains(changelog, "Edge case bug") {
		t.Errorf("changelog should contain 'Edge case bug', got: %s", changelog)
	}
}

func TestReleaseService_GenerateReleaseChangelog_AIError(t *testing.T) {
	svc := NewReleaseService(
		nil,
		&mockAI{err: assertError("AI failure")},
		prompts.NewReleaseBuilder(),
	)

	info := domain.ReleaseInfo{
		Commits: []domain.CommitSummary{{Hash: "abc", Message: "fix: bug"}},
	}

	_, err := svc.GenerateReleaseChangelog(context.Background(), info, "v1.0.1")
	if err == nil {
		t.Fatal("expected error from AI failure")
	}
}

func TestReleaseService_GenerateReleaseChangelog_UsesSinceTagCommitsAndDiff(t *testing.T) {
	mock := &mockGit{
		log: []git.CommitLog{
			{Hash: "111", Message: "feat: shipped after tag"},
		},
		diff: domain.Change{
			Diff:     "diff --git a/feature.go b/feature.go\n@@ -1 +1 @@\n-old\n+new feature",
			DiffStat: "1 file changed, 1 insertion(+)",
		},
	}
	aiMock := &mockAI{text: "### Added\n- shipped after tag"}
	svc := NewReleaseService(mock, aiMock, prompts.NewReleaseBuilder())

	info := domain.ReleaseInfo{
		BaseBranch: "main",
		HeadBranch: "feature/release",
		LatestTag:  "v0.1.0",
		Commits: []domain.CommitSummary{ // base..head — must NOT be used
			{Hash: "222", Message: "in progress"},
		},
	}

	if _, err := svc.GenerateReleaseChangelog(context.Background(), info, "v0.1.1"); err != nil {
		t.Fatalf("GenerateReleaseChangelog: %v", err)
	}

	if mock.logFrom != "v0.1.0" || mock.logTo != "feature/release" {
		t.Errorf("expected log range v0.1.0..feature/release, got %s..%s", mock.logFrom, mock.logTo)
	}
	if mock.diffFrom != "v0.1.0" || mock.diffTo != "feature/release" {
		t.Errorf("expected diff range v0.1.0..feature/release, got %s..%s", mock.diffFrom, mock.diffTo)
	}
	if !strings.Contains(aiMock.lastUserPrompt, "shipped after tag") {
		t.Errorf("prompt should include since-tag commits, got: %s", aiMock.lastUserPrompt)
	}
	if strings.Contains(aiMock.lastUserPrompt, "in progress") {
		t.Errorf("prompt should not include base..head commits, got: %s", aiMock.lastUserPrompt)
	}
	if !strings.Contains(aiMock.lastUserPrompt, "feature.go") {
		t.Errorf("prompt should include the since-tag diff, got: %s", aiMock.lastUserPrompt)
	}
}

func TestReleaseService_GenerateReleaseChangelog_FallsBackWithoutTag(t *testing.T) {
	aiMock := &mockAI{text: "### Added\n- base commit"}
	svc := NewReleaseService(nil, aiMock, prompts.NewReleaseBuilder())

	info := domain.ReleaseInfo{
		Commits: []domain.CommitSummary{{Hash: "abc", Message: "fix: base commit"}},
	}

	if _, err := svc.GenerateReleaseChangelog(context.Background(), info, "v1.0.0"); err != nil {
		t.Fatalf("GenerateReleaseChangelog: %v", err)
	}
	if !strings.Contains(aiMock.lastUserPrompt, "fix: base commit") {
		t.Errorf("expected fallback to info.Commits, got: %s", aiMock.lastUserPrompt)
	}
}

func TestReleaseService_GeneratePRDescription_Success(t *testing.T) {
	svc := NewReleaseService(
		nil,
		&mockAI{text: "# Release v1.2.0\n\n## Summary\nThis release adds payment retries."},
		prompts.NewReleaseBuilder(),
	)

	info := domain.ReleaseInfo{
		BaseBranch: "main",
		HeadBranch: "release/v1.2.0",
		Diff: domain.Change{
			Files:    []string{"payment.go", "handler.go"},
			DiffStat: "2 files changed",
		},
		Commits: []domain.CommitSummary{
			{Hash: "abc", Message: "feat: payment retries"},
		},
	}

	prDesc, err := svc.GeneratePRDescription(context.Background(), info, "v1.2.0", "### Added\n- Payment retries")
	if err != nil {
		t.Fatalf("GeneratePRDescription: %v", err)
	}

	if !strings.Contains(prDesc, "v1.2.0") {
		t.Errorf("PR description should contain version, got: %s", prDesc)
	}
	if !strings.Contains(prDesc, "payment retries") {
		t.Errorf("PR description should reference changelog content, got: %s", prDesc)
	}
}

func TestReleaseService_GeneratePRDescription_AIError(t *testing.T) {
	svc := NewReleaseService(
		nil,
		&mockAI{err: assertError("AI failure")},
		prompts.NewReleaseBuilder(),
	)

	info := domain.ReleaseInfo{
		Commits: []domain.CommitSummary{{Hash: "abc", Message: "fix: bug"}},
	}

	_, err := svc.GeneratePRDescription(context.Background(), info, "v1.0.0", "### Fixed\n- Bug")
	if err == nil {
		t.Fatal("expected error from AI failure")
	}
}

func TestUpdateChangelogFile_CreateNew(t *testing.T) {
	svc := NewReleaseService(nil, nil, prompts.NewReleaseBuilder())
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "CHANGELOG.md")

	err := svc.UpdateChangelogFile(path, "v1.0.0", "### Added\n- Initial release")
	if err != nil {
		t.Fatalf("UpdateChangelogFile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "# Changelog") {
		t.Errorf("should contain changelog header, got: %s", content)
	}
	if !strings.Contains(content, "## v1.0.0") {
		t.Errorf("should contain version header, got: %s", content)
	}
	if !strings.Contains(content, "Initial release") {
		t.Errorf("should contain changelog entry, got: %s", content)
	}
}

func TestUpdateChangelogFile_PrependToExisting(t *testing.T) {
	svc := NewReleaseService(nil, nil, prompts.NewReleaseBuilder())
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "CHANGELOG.md")

	// Write existing changelog
	existing := "# Changelog\n\nAll notable changes will be documented here.\n\n## v1.0.0\n\n### Added\n- First release\n"
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Prepend new entry
	err := svc.UpdateChangelogFile(path, "v1.1.0", "### Added\n- Payment retries")
	if err != nil {
		t.Fatalf("UpdateChangelogFile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	content := string(data)
	// New entry should appear before old entry
	v110Idx := strings.Index(content, "## v1.1.0")
	v100Idx := strings.Index(content, "## v1.0.0")
	if v110Idx < 0 {
		t.Errorf("should contain new version v1.1.0")
	}
	if v100Idx < 0 {
		t.Errorf("should contain existing version v1.0.0")
	}
	if v110Idx > v100Idx {
		t.Errorf("new entry (v1.1.0) should appear before old entry (v1.0.0)")
	}
	if !strings.Contains(content, "Payment retries") {
		t.Errorf("should contain new changelog content")
	}
	if !strings.Contains(content, "First release") {
		t.Errorf("should preserve old changelog content")
	}
}

func TestUpdateChangelogFile_ExistingWithNoHeader(t *testing.T) {
	svc := NewReleaseService(nil, nil, prompts.NewReleaseBuilder())
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "CHANGELOG.md")

	// Existing file without standard header
	existing := "## v0.5.0\n\n### Added\n- Beta feature\n"
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	err := svc.UpdateChangelogFile(path, "v1.0.0", "### Added\n- Stable release")
	if err != nil {
		t.Fatalf("UpdateChangelogFile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "# Changelog") {
		t.Errorf("should add standard header, got: %s", content)
	}
	if !strings.Contains(content, "Stable release") {
		t.Errorf("should contain new content, got: %s", content)
	}
	if !strings.Contains(content, "Beta feature") {
		t.Errorf("should preserve old content, got: %s", content)
	}
}

func TestUpdateChangelogFile_ReplaceExistingVersion(t *testing.T) {
	svc := NewReleaseService(nil, nil, prompts.NewReleaseBuilder())
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "CHANGELOG.md")

	existing := "# Changelog\n\nAll notable changes will be documented here.\n\n## v1.1.0\n\n### Added\n- Payment retries\n\n## v1.0.0\n\n### Added\n- First release\n\n## v0.9.0\n\n### Added\n- Beta feature\n"
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	err := svc.UpdateChangelogFile(path, "v1.0.0", "### Fixed\n- Security fix")
	if err != nil {
		t.Fatalf("UpdateChangelogFile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)

	if got := strings.Count(content, "## v1.0.0"); got != 1 {
		t.Errorf("v1.0.0 should appear exactly once, got %d:\n%s", got, content)
	}
	if !strings.Contains(content, "Security fix") {
		t.Errorf("should contain the new entry, got: %s", content)
	}
	if strings.Contains(content, "First release") {
		t.Errorf("old v1.0.0 content should be replaced, got: %s", content)
	}
	// Order preserved: v1.1.0 before v1.0.0 before v0.9.0
	idx110 := strings.Index(content, "## v1.1.0")
	idx100 := strings.Index(content, "## v1.0.0")
	idx090 := strings.Index(content, "## v0.9.0")
	if !(idx110 < idx100 && idx100 < idx090) {
		t.Errorf("entry order should be preserved, got v1.1.0=%d v1.0.0=%d v0.9.0=%d", idx110, idx100, idx090)
	}
}

func TestUpdateChangelogFile_ReplaceExistingVersion_LastEntry(t *testing.T) {
	svc := NewReleaseService(nil, nil, prompts.NewReleaseBuilder())
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "CHANGELOG.md")

	existing := "# Changelog\n\nAll notable changes will be documented here.\n\n## v1.0.0\n\n### Added\n- First release\n"
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := svc.UpdateChangelogFile(path, "v1.0.0", "### Fixed\n- Hotfix"); err != nil {
		t.Fatalf("UpdateChangelogFile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if got := strings.Count(content, "## v1.0.0"); got != 1 {
		t.Errorf("v1.0.0 should appear exactly once, got %d:\n%s", got, content)
	}
	if !strings.Contains(content, "Hotfix") || strings.Contains(content, "First release") {
		t.Errorf("last entry should be replaced in place, got: %s", content)
	}
}

func TestUpdateChangelogFile_RunningTwice_NoDuplicate(t *testing.T) {
	// Regression: running the release command twice for the same version must
	// not duplicate the changelog entry.
	svc := NewReleaseService(nil, nil, prompts.NewReleaseBuilder())
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "CHANGELOG.md")

	if err := svc.UpdateChangelogFile(path, "v1.0.0", "### Added\n- First run"); err != nil {
		t.Fatalf("UpdateChangelogFile (first): %v", err)
	}
	if err := svc.UpdateChangelogFile(path, "v1.0.0", "### Added\n- Second run"); err != nil {
		t.Fatalf("UpdateChangelogFile (second): %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if got := strings.Count(content, "## v1.0.0"); got != 1 {
		t.Errorf("v1.0.0 should appear exactly once after two runs, got %d:\n%s", got, content)
	}
	if !strings.Contains(content, "Second run") {
		t.Errorf("the latest entry should win, got: %s", content)
	}
	if strings.Contains(content, "First run") {
		t.Errorf("the stale first-run entry should be replaced, got: %s", content)
	}
}

func TestUpdateChangelogFile_RepairsExistingDuplicates(t *testing.T) {
	// A file already corrupted with duplicate entries (from running the release
	// command before this fix) should be repaired on the next update.
	svc := NewReleaseService(nil, nil, prompts.NewReleaseBuilder())
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "CHANGELOG.md")

	existing := "# Changelog\n\nAll notable changes will be documented here.\n\n## v1.0.0\n\n### Added\n- First version\n\n## v1.0.0\n\n### Added\n- Duplicated version\n\n## v0.9.0\n\n### Added\n- Beta feature\n"
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := svc.UpdateChangelogFile(path, "v1.0.0", "### Added\n- Fresh entry"); err != nil {
		t.Fatalf("UpdateChangelogFile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if got := strings.Count(content, "## v1.0.0"); got != 1 {
		t.Errorf("v1.0.0 should appear exactly once after repair, got %d:\n%s", got, content)
	}
	if strings.Contains(content, "First version") || strings.Contains(content, "Duplicated version") {
		t.Errorf("stale duplicate bodies should be removed, got: %s", content)
	}
	if !strings.Contains(content, "Fresh entry") || !strings.Contains(content, "Beta feature") {
		t.Errorf("new entry and untouched versions should remain, got: %s", content)
	}
}

func TestBumpVersionInFiles_PackageJSON(t *testing.T) {
	svc := NewReleaseService(nil, nil, prompts.NewReleaseBuilder())
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	// Create test package.json
	pkgJSON := `{
  "name": "test-pkg",
  "version": "1.0.0",
  "description": "A test package"
}`
	if err := os.WriteFile("package.json", []byte(pkgJSON), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	modified, err := svc.BumpVersionInFiles("v1.2.3")
	if err != nil {
		t.Fatalf("BumpVersionInFiles: %v", err)
	}

	if len(modified) != 1 || modified[0] != "package.json" {
		t.Errorf("modified = %v, want [package.json]", modified)
	}

	data, err := os.ReadFile("package.json")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if !strings.Contains(string(data), `"version": "1.2.3"`) {
		t.Errorf("package.json should have version 1.2.3, got: %s", data)
	}
}

func TestBumpVersionInFiles_BothFiles(t *testing.T) {
	svc := NewReleaseService(nil, nil, prompts.NewReleaseBuilder())
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	// Create both files
	pkgJSON := `{"name":"test","version":"1.0.0"}`
	lockJSON := `{"name":"test","version":"1.0.0","lockfileVersion":1}`
	os.WriteFile("package.json", []byte(pkgJSON), 0o644)
	os.WriteFile("package-lock.json", []byte(lockJSON), 0o644)

	modified, err := svc.BumpVersionInFiles("v2.0.0")
	if err != nil {
		t.Fatalf("BumpVersionInFiles: %v", err)
	}

	if len(modified) != 2 {
		t.Errorf("modified = %v, want [package.json package-lock.json]", modified)
	}

	// Verify both files updated
	for _, f := range []string{"package.json", "package-lock.json"} {
		data, _ := os.ReadFile(f)
		if !strings.Contains(string(data), `"version": "2.0.0"`) {
			t.Errorf("%s should have version 2.0.0, got: %s", f, data)
		}
	}
}

func TestBumpVersionInFiles_NoFiles(t *testing.T) {
	svc := NewReleaseService(nil, nil, prompts.NewReleaseBuilder())
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	modified, err := svc.BumpVersionInFiles("v1.0.0")
	if err != nil {
		t.Fatalf("BumpVersionInFiles: %v", err)
	}

	if len(modified) != 0 {
		t.Errorf("modified = %v, want empty slice", modified)
	}
}

func TestSuggestNextVersion(t *testing.T) {
	tests := []struct {
		current string
		bump    string
		want    string
	}{
		{current: "", bump: "minor", want: "v0.1.0"},
		{current: "v1.0.0", bump: "major", want: "v2.0.0"},
		{current: "v1.0.0", bump: "minor", want: "v1.1.0"},
		{current: "v1.0.0", bump: "patch", want: "v1.0.1"},
		{current: "v2.5.3", bump: "major", want: "v3.0.0"},
		{current: "v2.5.3", bump: "minor", want: "v2.6.0"},
		{current: "v2.5.3", bump: "patch", want: "v2.5.4"},
		{current: "1.0.0", bump: "minor", want: "v1.1.0"},
		{current: "v0.9.0", bump: "minor", want: "v0.10.0"},
		{current: "v10.0.0", bump: "major", want: "v11.0.0"},
		// Non-semver fallback
		{current: "v1.0", bump: "minor", want: "v1.0"},
		{current: "v1", bump: "patch", want: "v1"},
	}

	for _, tt := range tests {
		t.Run(tt.current+"_"+tt.bump, func(t *testing.T) {
			got := SuggestNextVersion(tt.current, tt.bump)
			if got != tt.want {
				t.Errorf("SuggestNextVersion(%q, %q) = %q, want %q", tt.current, tt.bump, got, tt.want)
			}
		})
	}
}

func TestComputeDiffStat(t *testing.T) {
	tests := []struct {
		name string
		diff domain.Change
		want string
	}{
		{
			name: "with diff stat",
			diff: domain.Change{DiffStat: "10 files changed, 200 insertions(+)", Files: []string{"a.go"}},
			want: "10 files changed, 200 insertions(+)",
		},
		{
			name: "without diff stat",
			diff: domain.Change{Files: []string{"a.go", "b.go"}},
			want: "2 file(s) changed",
		},
		{
			name: "empty change",
			diff: domain.Change{},
			want: "0 file(s) changed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeDiffStat(tt.diff)
			if got != tt.want {
				t.Errorf("ComputeDiffStat() = %q, want %q", got, tt.want)
			}
		})
	}
}

// assertError returns an error with the given message for testing.
type assertError string

func (e assertError) Error() string { return string(e) }
