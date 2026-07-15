package prompts

import (
	"fmt"
	"strings"

	"github.com/user/gitx/internal/domain"
	"github.com/user/gitx/internal/git"
)

// PRPromptInput contains the information needed to generate a PR description.
type PRPromptInput struct {
	Diff     domain.Change
	Branch   string
	Base     string
	Commits  []git.CommitLog
}

// PRBuilder generates prompts for pull request description generation.
type PRBuilder struct{}

// NewPRBuilder creates a new PR prompt builder.
func NewPRBuilder() *PRBuilder {
	return &PRBuilder{}
}

// Build returns system and user prompts for PR description generation.
func (b *PRBuilder) Build(input PRPromptInput) (string, string, error) {
	if input.Diff.IsEmpty() && len(input.Commits) == 0 {
		return "", "", fmt.Errorf("empty diff and no commits: nothing to generate a PR from")
	}

	system := `You are an expert software engineer generating pull request descriptions.

Generate a structured PR description with the following sections:

## Summary
A high-level summary of what this PR does.

## Changes
Bullet-point list of specific changes.

## Testing
How the changes have been tested, or notes on what testing is needed.

## Risks
Any risks or concerns.

## Breaking Changes
Note any breaking changes, or "None" if there are none.

Rules:
- Be concise but thorough
- Focus on what changed and why
- Do not invent changes that are not in the diff or commits`

	// Build commit summary
	var commitSummary string
	if len(input.Commits) > 0 {
		var b strings.Builder
		b.WriteString("Commits:\n")
		for _, c := range input.Commits {
			hash := c.Hash
			if len(hash) > 7 {
				hash = hash[:7]
			}
			fmt.Fprintf(&b, "  - %s %s\n", hash, c.Message)
		}
		commitSummary = b.String()
	}

	user := fmt.Sprintf(`Generate a pull request description.

Branch: %s
Base: %s

%s
Diff:
%s`,
		input.Branch,
		input.Base,
		commitSummary,
		strings.TrimSpace(input.Diff.Diff),
	)

	return system, user, nil
}
