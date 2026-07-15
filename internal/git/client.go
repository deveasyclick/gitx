package git

import (
	"context"

	"github.com/user/gitx/internal/domain"
)

// Client defines the git operations GitX uses.
// Implementations must not call AI or external services.
type Client interface {
	// DiffCached returns the staged diff (git diff --cached).
	DiffCached(ctx context.Context) (domain.Change, error)

	// DiffUnstaged returns the unstaged diff (git diff) for tracked files.
	DiffUnstaged(ctx context.Context) (domain.Change, error)

	// Diff returns the diff between the current state and a base branch.
	Diff(ctx context.Context, base string) (domain.Change, error)

	// Log returns commits between two refs (exclusive from, inclusive to).
	// Use "HEAD" for to when generating PR descriptions.
	Log(ctx context.Context, from, to string) ([]CommitLog, error)

	// Commit creates a commit with the given message.
	Commit(ctx context.Context, msg domain.CommitMessage) error

	// Status returns staged changes.
	Status(ctx context.Context) (StagedChanges, error)

	// UnstagedStatus returns unstaged changes (tracked modified but not staged).
	UnstagedStatus(ctx context.Context) (StagedChanges, error)

	// Tags returns all tags, sorted by version (newest first when possible).
	Tags(ctx context.Context) ([]string, error)

	// RepoInfo returns repository metadata.
	RepoInfo(ctx context.Context) (domain.RepoInfo, error)

	// Stage adds specific files to the staging area.
	Stage(ctx context.Context, files []string) error

	// UnstageAll removes all files from the staging area.
	UnstageAll(ctx context.Context) error
}
