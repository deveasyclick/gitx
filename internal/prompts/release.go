package prompts

import (
	"fmt"
	"strings"

	"github.com/user/gitx/internal/domain"
)

// ReleaseChangelogInput contains the information needed to generate a release changelog.
type ReleaseChangelogInput struct {
	Version       string
	PreviousTag   string
	CurrentTag    string
	Commits       []domain.CommitSummary
	CommitDetails string // optional: git diff summary line
	Diff          string // optional: unified diff text (primary source of changes)
}

// maxChangelogDiffChars caps how much diff text is included in changelog prompts.
const maxChangelogDiffChars = 50000

// formatDiffSection formats diff text for a changelog prompt, capping its size
// so huge diffs don't blow up the model context.
func formatDiffSection(diff string) string {
	diff = strings.TrimSpace(diff)
	if diff == "" {
		return ""
	}
	note := ""
	if len(diff) > maxChangelogDiffChars {
		diff = diff[:maxChangelogDiffChars]
		note = fmt.Sprintf("\n\n(diff truncated after %d characters — the commit list below covers the remaining changes)\n", maxChangelogDiffChars)
	}
	return "Diff:\n" + diff + note
}

// ReleasePRInput contains the information needed to generate a PR description.
type ReleasePRInput struct {
	Version       string
	BaseBranch    string
	HeadBranch    string
	ChangelogText string
	Commits       []domain.CommitSummary
	FilesChanged  []string
	DiffStat      string
}

// ReleaseBuilder generates prompts for release-related AI tasks.
type ReleaseBuilder struct{}

// NewReleaseBuilder creates a new release prompt builder.
func NewReleaseBuilder() *ReleaseBuilder {
	return &ReleaseBuilder{}
}

// BuildReleaseChangelog builds prompts for generating release changelog content.
func (b *ReleaseBuilder) BuildReleaseChangelog(input ReleaseChangelogInput) (system, user string, err error) {
	if len(input.Commits) == 0 && strings.TrimSpace(input.Diff) == "" {
		return "", "", fmt.Errorf("no changes: nothing to generate a changelog from")
	}

	system = `You are an expert software engineer generating a release changelog entry.

Categorize the changes into these sections:

### Added
For new features.

### Fixed
For bug fixes.

### Changed
For changes in existing functionality.

### Removed
For now-removed features.

Rules:
- Base the changelog on the actual code changes in the diff; the diff is the source of truth
- Treat commit messages as supplementary context only — they can be incomplete or misleading (e.g. "wip", "in progress")
- Group related changes together into concise bullet points
- Use present-tense descriptions (e.g., "Add", "Fix", "Update")
- Do not invent changes that are not in the diff
- Keep descriptions concise and user-focused
- If a commit message follows conventional commits format (feat:, fix:, chore:, etc.), use it only to help with categorization of the corresponding diff
- Output ONLY the section content — no preamble, no commentary`

	var rangeInfo string
	if input.PreviousTag != "" {
		rangeInfo = fmt.Sprintf("Previous tag: %s\nCurrent tag: %s\n", input.PreviousTag, input.CurrentTag)
	} else {
		rangeInfo = fmt.Sprintf("Release: %s\n", input.CurrentTag)
	}

	diffSection := formatDiffSection(input.Diff)

	var commitLines strings.Builder
	for _, c := range input.Commits {
		hash := c.Hash
		if len(hash) > 7 {
			hash = hash[:7]
		}
		fmt.Fprintf(&commitLines, "  - %s %s\n", hash, c.Message)
	}

	user = fmt.Sprintf(`Generate a changelog entry for version %s.

%s
%s
Commits:
%s`,
		input.Version,
		rangeInfo,
		diffSection,
		commitLines.String(),
	)

	return system, user, nil
}

// BuildPRDescription builds prompts for generating a release PR description.
func (b *ReleaseBuilder) BuildPRDescription(input ReleasePRInput) (system, user string, err error) {
	system = `You are an expert software engineer generating a pull request description for a release.

Generate a clear, structured PR description with:

1. A brief title for the release PR
2. A summary paragraph explaining what this release contains
3. A changelog section summarizing key changes (grouped by type)
4. A testing notes section
5. A checklist for the release process

Format the output as markdown. Be concise but informative.`

	commitList := make([]string, len(input.Commits))
	for i, c := range input.Commits {
		hash := c.Hash
		if len(hash) > 7 {
			hash = hash[:7]
		}
		commitList[i] = fmt.Sprintf("- %s %s", hash, c.Message)
	}

	var fileList string
	if len(input.FilesChanged) > 0 {
		fileList = "Files changed: " + strings.Join(input.FilesChanged, ", ")
	}

	user = fmt.Sprintf(`Create a release pull request description.

Version: %s
Branch: %s → %s

%s

%s

Changelog:
%s`,
		input.Version,
		input.HeadBranch,
		input.BaseBranch,
		fileList,
		input.DiffStat,
		input.ChangelogText,
	)

	if len(commitList) > 0 {
		user += "\n\nAll commits:\n" + strings.Join(commitList, "\n")
	}

	return system, user, nil
}

// BuildVersionSuggestion builds prompts for suggesting a version bump.
func (b *ReleaseBuilder) BuildVersionSuggestion(commits []domain.CommitSummary) (system, user string, err error) {
	system = `You are an expert at semantic versioning. Based on the commits provided, determine if this release should be a:

- major: Breaking changes or major feature additions
- minor: New features, non-breaking changes
- patch: Bug fixes, small improvements, documentation

Reply with exactly one word: major, minor, or patch.`

	var commitLines strings.Builder
	for _, c := range commits {
		hash := c.Hash
		if len(hash) > 7 {
			hash = hash[:7]
		}
		fmt.Fprintf(&commitLines, "  - %s %s\n", hash, c.Message)
	}

	user = fmt.Sprintf("Based on these commits, what version bump type is appropriate?\n\n%s", commitLines.String())

	return system, user, nil
}
