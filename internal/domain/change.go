package domain

// Change represents a set of file modifications.
// It is the core model: commit messages, PR descriptions,
// and changelog entries are all different presentations of a Change.
type Change struct {
	Files    []string // changed file paths
	Diff     string   // unified diff text
	DiffStat string   // summary line, e.g. "10 files changed, 200 insertions(+), 50 deletions(-)"
}

// IsEmpty returns true when no files were changed.
func (c Change) IsEmpty() bool {
	return len(c.Files) == 0 && c.Diff == ""
}
