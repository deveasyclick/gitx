package prompts

import (
	"fmt"
	"strings"

	"github.com/user/gitx/internal/domain"
	"github.com/user/gitx/internal/git"
)

// DescribePromptInput contains the information needed to describe changes.
type DescribePromptInput struct {
	Branch   string
	Base     string
	Staged   domain.Change // staged changes (empty if not included)
	Unstaged domain.Change // unstaged changes (empty if not included)
	Commits  []git.CommitLog
}

// DescribeBuilder generates prompts for describing repository changes.
type DescribeBuilder struct{}

// NewDescribeBuilder creates a new describe prompt builder.
func NewDescribeBuilder() *DescribeBuilder {
	return &DescribeBuilder{}
}

// Build returns system and user prompts for describing changes.
func (b *DescribeBuilder) Build(input DescribePromptInput) (string, string, error) {
	if len(input.Commits) == 0 && input.Staged.IsEmpty() && input.Unstaged.IsEmpty() {
		return "", "", fmt.Errorf("nothing to describe: no commits, staged, or unstaged changes")
	}

	system := `You are an expert software engineer describing the current state of a codebase.

Generate a structured markdown description with the following sections:

## Overview
A high-level summary of what's happening in the current state. Synthesize the key themes from the commits and changes.

## Commits
Bullet-point list of what each commit does, focusing on the meaningful change (not just the commit subject line).

## Staged Changes
If present, describe what the staged files change. Focus on the intent and impact.

## Unstaged Changes
If present, describe what the unstaged files change. Focus on what's still in progress.

Rules:
- Be concise but informative
- Focus on intent and impact, not just file names
- Only include sections for which data was provided
- Use present tense`

	// Build commit summary
	var commitSummary string
	if len(input.Commits) > 0 {
		// Determine range label
		var rangeLabel string
		if input.Base != "" {
			rangeLabel = fmt.Sprintf("Commits since %s (%d commits)", input.Base, len(input.Commits))
		} else {
			rangeLabel = fmt.Sprintf("Recent commits (%d)", len(input.Commits))
		}

		var b strings.Builder
		b.WriteString(rangeLabel + ":\n")
		for _, c := range input.Commits {
			hash := c.Hash
			if len(hash) > 7 {
				hash = hash[:7]
			}
			fmt.Fprintf(&b, "  - %s %s\n", hash, c.Message)
		}
		commitSummary = b.String()
	}

	// Build staged diff section
	var stagedSection string
	if !input.Staged.IsEmpty() {
		stagedSection = "Staged changes diff:\n" + strings.TrimSpace(input.Staged.Diff)
	}

	// Build unstaged diff section
	var unstagedSection string
	if !input.Unstaged.IsEmpty() {
		unstagedSection = "Unstaged changes diff:\n" + strings.TrimSpace(input.Unstaged.Diff)
	}

	// Build user prompt
	var parts []string
	parts = append(parts, fmt.Sprintf("Describe the current state of this repository."))
	parts = append(parts, fmt.Sprintf("Branch: %s", input.Branch))

	if commitSummary != "" {
		parts = append(parts, commitSummary)
	}
	if stagedSection != "" {
		parts = append(parts, stagedSection)
	}
	if unstagedSection != "" {
		parts = append(parts, unstagedSection)
	}

	user := strings.Join(parts, "\n\n")

	return system, user, nil
}
