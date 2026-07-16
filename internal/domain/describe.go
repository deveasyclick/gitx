package domain

import (
	"fmt"
	"strings"
)

// DescribeChanges represents a description of the current repository state.
type DescribeChanges struct {
	Overview        string   // high-level summary of all changes
	Commits         []string // descriptions of individual commits
	StagedChanges   []string // descriptions of staged file changes
	UnstagedChanges []string // descriptions of unstaged file changes
}

// IsEmpty returns true when no content was generated.
func (d DescribeChanges) IsEmpty() bool {
	return d.Overview == "" && len(d.Commits) == 0 &&
		len(d.StagedChanges) == 0 && len(d.UnstagedChanges) == 0
}

// String returns the full description as formatted markdown.
func (d DescribeChanges) String() string {
	var b strings.Builder

	if d.Overview != "" {
		fmt.Fprintf(&b, "## Overview\n%s\n\n", d.Overview)
	}

	if len(d.Commits) > 0 {
		b.WriteString("## Commits\n")
		for _, c := range d.Commits {
			fmt.Fprintf(&b, "- %s\n", c)
		}
		b.WriteString("\n")
	}

	if len(d.StagedChanges) > 0 {
		b.WriteString("## Staged Changes\n")
		for _, c := range d.StagedChanges {
			fmt.Fprintf(&b, "- %s\n", c)
		}
		b.WriteString("\n")
	}

	if len(d.UnstagedChanges) > 0 {
		b.WriteString("## Unstaged Changes\n")
		for _, c := range d.UnstagedChanges {
			fmt.Fprintf(&b, "- %s\n", c)
		}
		b.WriteString("\n")
	}

	return strings.TrimSpace(b.String())
}
