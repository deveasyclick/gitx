package prompts

import (
	"fmt"
	"strings"

	"github.com/user/gitx/internal/git"
)

// ChangelogPromptInput contains the information needed to generate a changelog entry.
type ChangelogPromptInput struct {
	From    string // start tag (empty if none)
	To      string // end tag (empty for HEAD)
	Commits []git.CommitLog
	Tags    []string
	Diff    string // unified diff text (primary source of changes)
}

// ChangelogBuilder generates prompts for changelog generation.
type ChangelogBuilder struct{}

// NewChangelogBuilder creates a new changelog prompt builder.
func NewChangelogBuilder() *ChangelogBuilder {
	return &ChangelogBuilder{}
}

// Build returns system and user prompts for changelog generation.
func (b *ChangelogBuilder) Build(input ChangelogPromptInput) (string, string, error) {
	if len(input.Commits) == 0 {
		return "", "", fmt.Errorf("no commits: nothing to generate a changelog from")
	}

	system := `You are an expert software engineer generating changelog entries.

Categorize the changes into:

## Added
For new features.

## Fixed
For bug fixes.

## Changed
For changes in existing functionality.

## Removed
For now-removed features.

Rules:
- Base the changelog on the actual code changes in the diff; the diff is the source of truth
- Treat commit messages as supplementary context only — they can be incomplete or misleading (e.g. "wip", "in progress")
- Group related commits together
- Use bullet points with present-tense descriptions
- Do not invent changes that are not in the diff
- Keep descriptions concise`

	var tagInfo string
	if input.From != "" && input.To != "" {
		tagInfo = fmt.Sprintf("From: %s\nTo: %s\n", input.From, input.To)
	} else if input.From != "" {
		tagInfo = fmt.Sprintf("Since: %s\n", input.From)
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

	user := fmt.Sprintf(`Generate a changelog entry for the following commits.

%s
%s
Commits:
%s`,
		tagInfo,
		diffSection,
		commitLines.String(),
	)

	return system, user, nil
}
