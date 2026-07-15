package git

import "time"

// CommitLog represents one commit in git log output.
type CommitLog struct {
	Hash    string
	Author  string
	Message string
	Date    time.Time
}

// StagedChanges summarizes what is currently staged.
type StagedChanges struct {
	Files   []string // staged file paths
	IsEmpty bool     // true when nothing is staged
}
