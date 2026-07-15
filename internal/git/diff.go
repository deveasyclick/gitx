package git

import (
	"path"
	"strings"

	"github.com/user/gitx/internal/domain"
)

// GroupedDiff represents a diff grouped by directory.
type GroupedDiff struct {
	Dir   string         // directory name (e.g. "api", "ui", or "." for root)
	Files []string       // files in this group
	Diff  string         // combined diff for this group
}

// GroupDiffsByDir splits a unified diff into groups by top-level directory.
func GroupDiffsByDir(change domain.Change) []GroupedDiff {
	sections := splitDiffSections(change.Diff)
	groups := make(map[string]*GroupedDiff)

	// Preserve order of first occurrence
	var order []string

	for _, sec := range sections {
		file := extractFilePath(sec.header)
		if file == "" {
			continue
		}
		dir := topLevelDir(file)

		if _, ok := groups[dir]; !ok {
			order = append(order, dir)
			groups[dir] = &GroupedDiff{Dir: dir}
		}
		g := groups[dir]
		g.Files = append(g.Files, file)
		if g.Diff != "" {
			g.Diff += "\n"
		}
		g.Diff += sec.header + "\n" + sec.body
	}

	result := make([]GroupedDiff, 0, len(order))
	for _, dir := range order {
		result = append(result, *groups[dir])
	}
	return result
}

// diffSection is a single file entry in a unified diff.
type diffSection struct {
	header string // e.g. "diff --git a/x.go b/x.go\nindex abc..def\n--- a/x.go\n+++ b/x.go"
	body   string // the @@ ... @@ hunk and content lines
}

// splitDiffSections splits a unified diff into per-file sections.
func splitDiffSections(diff string) []diffSection {
	lines := strings.Split(diff, "\n")
	var sections []diffSection
	var current *diffSection
	inHeader := false

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git ") {
			if current != nil {
				sections = append(sections, *current)
			}
			current = &diffSection{header: line}
			inHeader = true
			continue
		}
		if current == nil {
			continue
		}

		if strings.HasPrefix(line, "@@") {
			inHeader = false
		}

		if inHeader {
			if current.header != "" {
				current.header += "\n"
			}
			current.header += line
		} else {
			if current.body != "" {
				current.body += "\n"
			}
			current.body += line
		}
	}

	if current != nil {
		sections = append(sections, *current)
	}

	return sections
}

// extractFilePath extracts the file path from a diff header line.
// Handles "diff --git a/path b/path" → returns "path".
func extractFilePath(header string) string {
	// Find the first line
	first := header
	if idx := strings.Index(header, "\n"); idx >= 0 {
		first = header[:idx]
	}
	// "diff --git a/path b/path"
	parts := strings.Split(first, " ")
	if len(parts) >= 4 {
		return strings.TrimPrefix(parts[3], "b/")
	}
	return ""
}

// topLevelDir returns the top-level directory of a file path.
// "api/handler.go" → "api", "main.go" → "."
func topLevelDir(filePath string) string {
	dir := path.Dir(filePath)
	if dir == "." {
		return "."
	}
	parts := strings.SplitN(dir, "/", 2)
	return parts[0]
}
