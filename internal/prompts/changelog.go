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

Categorize commits into:

## Added
For new features.

## Fixed
For bug fixes.

## Changed
For changes in existing functionality.

## Removed
For now-removed features.

Rules:
- Group related commits together
- Use bullet points with present-tense descriptions
- Do not invent changes that are not in the commits
- Keep descriptions concise`

	var tagInfo string
	if input.From != "" && input.To != "" {
		tagInfo = fmt.Sprintf("From: %s\nTo: %s\n", input.From, input.To)
	} else if input.From != "" {
		tagInfo = fmt.Sprintf("Since: %s\n", input.From)
	}

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
Commits:
%s`,
		tagInfo,
		commitLines.String(),
	)

	return system, user, nil
}
