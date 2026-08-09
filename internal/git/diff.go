package git

import (
	"path"
	"strings"

	"github.com/user/gitx/internal/domain"
)

// GroupedDiff represents a group of changed files with a combined diff.
// A group may be a logical group (named by an AI partition) or a directory group.
type GroupedDiff struct {
	Name  string   // group name (e.g. a logical description or directory name)
	Files []string // files in this group
	Diff  string   // combined diff for this group
}

// FileGroup is a named partition of files, e.g. produced by AI-based grouping.
type FileGroup struct {
	Name  string   // logical group name
	Files []string // files assigned to this group
}

// GroupDiffsByDir splits a unified diff into groups by top-level directory.
// It is used as a fallback when logical (AI) grouping is unavailable.
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
			groups[dir] = &GroupedDiff{Name: dir}
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

// GroupDiffsByFileGroups recombines the per-file diff sections of change into
// named groups according to the given partition. Files that appear in a group
// but have no diff section are skipped; files not mentioned by any group are
// simply not included (callers should use FilesInDiff to verify full coverage).
func GroupDiffsByFileGroups(change domain.Change, groups []FileGroup) []GroupedDiff {
	sections := splitDiffSections(change.Diff)
	byFile := make(map[string]string, len(sections))
	for _, sec := range sections {
		file := extractFilePath(sec.header)
		if file == "" {
			continue
		}
		byFile[file] = sec.header + "\n" + sec.body
	}

	result := make([]GroupedDiff, 0, len(groups))
	seen := make(map[string]bool)
	for _, fg := range groups {
		gd := GroupedDiff{Name: fg.Name}
		for _, f := range fg.Files {
			if seen[f] {
				continue
			}
			d, ok := byFile[f]
			if !ok {
				continue
			}
			seen[f] = true
			gd.Files = append(gd.Files, f)
			if gd.Diff != "" {
				gd.Diff += "\n"
			}
			gd.Diff += d
		}
		if len(gd.Files) > 0 {
			result = append(result, gd)
		}
	}
	return result
}

// FilesInDiff returns the file paths referenced by a unified diff, in order of
// first appearance. It is used to verify that a grouping covers every change.
func FilesInDiff(change domain.Change) []string {
	var files []string
	seen := make(map[string]bool)
	for _, sec := range splitDiffSections(change.Diff) {
		file := extractFilePath(sec.header)
		if file == "" || seen[file] {
			continue
		}
		seen[file] = true
		files = append(files, file)
	}
	return files
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
