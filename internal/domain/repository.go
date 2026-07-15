package domain

// RepoInfo describes the current state of the git repository.
type RepoInfo struct {
	CurrentBranch string   // active branch name
	Remote        string   // origin URL (if any)
	Tags          []string // all tags sorted
	IsClean       bool     // true when working tree has no uncommitted changes
}
