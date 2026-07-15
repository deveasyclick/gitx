package domain

import "errors"

var (
	ErrEmptyCommitTitle = errors.New("commit title is required")
	ErrNoStagedChanges  = errors.New("no staged changes found")
)
