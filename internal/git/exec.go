package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/user/gitx/internal/domain"
)

// ExecClient implements Client using the system git binary.
type ExecClient struct {
	repoPath string // working directory for git commands
}

// NewExecClient creates a client that runs git in the given directory.
func NewExecClient(repoPath string) *ExecClient {
	return &ExecClient{repoPath: repoPath}
}

// run executes a git command and returns stdout.
func (c *ExecClient) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = c.repoPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, msg)
	}

	return stdout.String(), nil
}

// DiffCached returns staged changes (git diff --cached).
func (c *ExecClient) DiffCached(ctx context.Context) (domain.Change, error) {
	diff, err := c.run(ctx, "diff", "--cached")
	if err != nil {
		return domain.Change{}, err
	}

	stat, err := c.run(ctx, "diff", "--cached", "--stat")
	if err != nil {
		return domain.Change{}, err
	}

	files := parseFilesFromDiff(diff)

	return domain.Change{
		Files:    files,
		Diff:     diff,
		DiffStat: strings.TrimSpace(stat),
	}, nil
}

// DiffUnstaged returns unstaged changes for tracked and untracked files.
// Uses git diff for tracked changes and builds synthetic diffs for untracked files.
func (c *ExecClient) DiffUnstaged(ctx context.Context) (domain.Change, error) {
	// Tracked unstaged changes
	trackedDiff, err := c.run(ctx, "diff")
	if err != nil {
		return domain.Change{}, err
	}

	trackedStat, err := c.run(ctx, "diff", "--stat")
	if err != nil {
		return domain.Change{}, err
	}

	// Untracked files
	untrackedOut, err := c.run(ctx, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return domain.Change{}, err
	}

	var untrackedDiffs []string
	var untrackedFiles []string
	for _, f := range strings.Split(strings.TrimSpace(untrackedOut), "\n") {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		untrackedFiles = append(untrackedFiles, f)

		content, err := os.ReadFile(filepath.Join(c.repoPath, f))
		if err != nil {
			continue
		}
		lines := strings.Split(string(content), "\n")
		// Drop trailing empty line from split
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}

		var b strings.Builder
		fmt.Fprintf(&b, "diff --git a/%s b/%s\n", f, f)
		fmt.Fprintf(&b, "new file mode 100644\n")
		fmt.Fprintf(&b, "--- /dev/null\n")
		fmt.Fprintf(&b, "+++ b/%s\n", f)
		fmt.Fprintf(&b, "@@ -0,0 +1,%d @@\n", len(lines))
		for _, l := range lines {
			fmt.Fprintf(&b, "+%s\n", l)
		}
		untrackedDiffs = append(untrackedDiffs, b.String())
	}

	// Merge tracked and untracked
	allDiff := trackedDiff
	if len(untrackedDiffs) > 0 {
		if allDiff != "" && !strings.HasSuffix(allDiff, "\n") {
			allDiff += "\n"
		}
		allDiff += strings.Join(untrackedDiffs, "\n")
	}

	allFiles := parseFilesFromDiff(trackedDiff)
	allFiles = append(allFiles, untrackedFiles...)

	allStat := strings.TrimSpace(trackedStat)
	if len(untrackedFiles) > 0 {
		if allStat != "" {
			allStat += "\n"
		}
		allStat += fmt.Sprintf("%d untracked files", len(untrackedFiles))
	}

	return domain.Change{
		Files:    allFiles,
		Diff:     allDiff,
		DiffStat: allStat,
	}, nil
}

// Diff returns the diff between HEAD and the given base branch.
func (c *ExecClient) Diff(ctx context.Context, base string) (domain.Change, error) {
	diff, err := c.run(ctx, "diff", base+"...HEAD")
	if err != nil {
		return domain.Change{}, err
	}

	stat, err := c.run(ctx, "diff", base+"...HEAD", "--stat")
	if err != nil {
		return domain.Change{}, err
	}

	files := parseFilesFromDiff(diff)

	return domain.Change{
		Files:    files,
		Diff:     diff,
		DiffStat: strings.TrimSpace(stat),
	}, nil
}

// Log returns commits between two refs.
func (c *ExecClient) Log(ctx context.Context, from, to string) ([]CommitLog, error) {
	rangeSpec := from + ".." + to
	out, err := c.run(ctx, "log", rangeSpec, "--oneline", "--format=%H|%an|%ai|%s")
	if err != nil {
		return nil, err
	}

	return parseCommitLog(out), nil
}

// RecentCommits returns the last n commits from the given ref.
// Uses --max-count to avoid errors when the ref has fewer than n commits.
func (c *ExecClient) RecentCommits(ctx context.Context, n int, ref string) ([]CommitLog, error) {
	out, err := c.run(ctx, "log", fmt.Sprintf("--max-count=%d", n), ref,
		"--format=%H|%an|%ai|%s")
	if err != nil {
		return nil, err
	}
	return parseCommitLog(out), nil
}

// Commit creates a commit using the given message.
func (c *ExecClient) Commit(ctx context.Context, msg domain.CommitMessage) error {
	if err := msg.Validate(); err != nil {
		return err
	}

	args := []string{"commit", "-m", msg.Title}
	if msg.Body != "" {
		args = append(args, "-m", msg.Body)
	}

	_, err := c.run(ctx, args...)
	return err
}

// Status returns information about staged changes.
func (c *ExecClient) Status(ctx context.Context) (StagedChanges, error) {
	out, err := c.run(ctx, "status", "--porcelain")
	if err != nil {
		return StagedChanges{}, err
	}

	var files []string
	for _, line := range strings.Split(out, "\n") {
		// Format: "XY filename" — X=index status, Y=worktree status, then space, then filename
		// "M  main.go"  — staged only
		// " M main.go"  — unstaged only (not matched here)
		// "MM main.go"  — both staged and unstaged
		// "?? file.go"  — untracked (not matched)
		if len(line) < 4 || line[2] != ' ' {
			continue
		}
		x := line[0]
		if x != ' ' && x != '?' {
			files = append(files, strings.TrimSpace(line[3:]))
		}
	}

	return StagedChanges{
		Files:   files,
		IsEmpty: len(files) == 0,
	}, nil
}

// UnstagedStatus returns unstaged changes (tracked modified but not staged).
// Handles patterns: " M" (unstaged only), "MM" (both staged and unstaged).
func (c *ExecClient) UnstagedStatus(ctx context.Context) (StagedChanges, error) {
	out, err := c.run(ctx, "status", "--porcelain")
	if err != nil {
		return StagedChanges{}, err
	}

	var files []string
	for _, line := range strings.Split(out, "\n") {
		// Format: "XY filename" — X=index status, Y=worktree status, then space, then filename
		// " M main.go"  — unstaged only
		// "MM main.go"  — both staged and unstaged
		// " D old.go"   — unstaged delete
		// "?? file.go"  — untracked (not matched here)
		if len(line) < 4 || line[2] != ' ' {
			continue
		}
		x, y := line[0], line[1]
		if y != ' ' && x != '?' {
			files = append(files, strings.TrimSpace(line[3:]))
		}
	}

	// Also check for untracked files separately since "git diff" won't cover them.
	for _, line := range strings.Split(out, "\n") {
		if len(line) >= 4 && line[0] == '?' && line[1] == '?' && line[2] == ' ' {
			files = append(files, strings.TrimSpace(line[3:]))
		}
	}

	return StagedChanges{
		Files:   files,
		IsEmpty: len(files) == 0,
	}, nil
}

// Tags returns all tags sorted by creation date (newest first).
func (c *ExecClient) Tags(ctx context.Context) ([]string, error) {
	out, err := c.run(ctx, "tag", "--sort=-creatordate")
	if err != nil {
		return nil, err
	}

	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}

	return strings.Split(out, "\n"), nil
}

// RepoInfo returns metadata about the current repository.
func (c *ExecClient) RepoInfo(ctx context.Context) (domain.RepoInfo, error) {
	branch, err := c.run(ctx, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		// Fallback: try rev-parse for detached HEAD
		branch, err = c.run(ctx, "rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			return domain.RepoInfo{}, err
		}
	}

	remote, _ := c.run(ctx, "remote", "get-url", "origin")

	status, err := c.Status(ctx)
	if err != nil {
		return domain.RepoInfo{}, err
	}

	tags, err := c.Tags(ctx)
	if err != nil {
		return domain.RepoInfo{}, err
	}

	return domain.RepoInfo{
		CurrentBranch: strings.TrimSpace(branch),
		Remote:        strings.TrimSpace(remote),
		Tags:          tags,
		IsClean:       status.IsEmpty,
	}, nil
}

// Stage adds specific files to the staging area.
func (c *ExecClient) Stage(ctx context.Context, files []string) error {
	if len(files) == 0 {
		return nil
	}
	args := append([]string{"add", "--"}, files...)
	_, err := c.run(ctx, args...)
	return err
}

// StageAll stages all changes (tracked modified, deleted, and untracked files).
func (c *ExecClient) StageAll(ctx context.Context) error {
	_, err := c.run(ctx, "add", "-A")
	return err
}

// UnstageAll removes all files from the staging area.
func (c *ExecClient) UnstageAll(ctx context.Context) error {
	_, err := c.run(ctx, "reset", "HEAD", "--quiet")
	if err != nil {
		// If HEAD doesn't exist (no commits yet), reset fails.
		// Use rm --cached instead to unstage everything.
		_, err2 := c.run(ctx, "rm", "--cached", "-r", ".")
		if err2 != nil {
			// If nothing was staged, rm --cached fails with
			// "pathspec '.' did not match any files".
			// This is fine — nothing to unstage.
			if strings.Contains(err2.Error(), "did not match any files") {
				return nil
			}
			return err2
		}
	}
	return nil
}

// parseFilesFromDiff extracts file paths from a unified diff.
func parseFilesFromDiff(diff string) []string {
	var files []string
	seen := make(map[string]bool)

	for _, line := range strings.Split(diff, "\n") {
		if !strings.HasPrefix(line, "diff --git a/") {
			continue
		}
		// "diff --git a/path b/path" → extract second path
		parts := strings.Split(line, " ")
		if len(parts) < 4 {
			continue
		}
		path := strings.TrimPrefix(parts[3], "b/")
		if !seen[path] {
			files = append(files, path)
			seen[path] = true
		}
	}

	return files
}

// parseCommitLog parses the output of our custom git log format.
func parseCommitLog(out string) []CommitLog {
	var logs []CommitLog
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// format: hash|author|date|message
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}

		entry := CommitLog{
			Hash:    parts[0],
			Author:  parts[1],
			Message: parts[3],
		}

		if t, err := time.Parse("2006-01-02T15:04:05-07:00", parts[2]); err == nil {
			entry.Date = t
		} else if t, err := time.Parse("2006-01-02 15:04:05 -0700", parts[2]); err == nil {
			entry.Date = t
		}

		logs = append(logs, entry)
	}

	return logs
}
