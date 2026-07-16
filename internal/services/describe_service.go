package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/user/gitx/internal/ai"
	"github.com/user/gitx/internal/domain"
	"github.com/user/gitx/internal/git"
	"github.com/user/gitx/internal/prompts"
	"github.com/user/gitx/internal/security"
)

// DescribeService orchestrates describing repository changes.
type DescribeService struct {
	git  git.Client
	ai   aiProvider
	pmpt prompts.Builder[prompts.DescribePromptInput]
	scan bool
}

// NewDescribeService creates a new describe service.
func NewDescribeService(gitClient git.Client, aiProvider aiProvider, prompt prompts.Builder[prompts.DescribePromptInput]) *DescribeService {
	return &DescribeService{
		git:  gitClient,
		ai:   aiProvider,
		pmpt: prompt,
		scan: true,
	}
}

// DescribeOptions controls what data to include in the description.
type DescribeOptions struct {
	Base     string // empty = last N commits
	Commits  int    // number of recent commits (default 10; ignored when Base is set)
	Staged   bool   // include staged changes
	Unstaged bool   // include unstaged changes
}

// DescribeResult contains the result of describing changes.
type DescribeResult struct {
	Description  domain.DescribeChanges
	Provider     string
	InputTokens  int
	OutputTokens int
}

// Generate generates a description of the current repository state.
func (s *DescribeService) Generate(ctx context.Context, opts DescribeOptions) (*DescribeResult, error) {
	// 1. Get repo info
	info, err := s.git.RepoInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading repo info: %w", err)
	}

	// Determine what to include based on flags.
	// Default (no flags): commits only.
	// Explicit flags opt in to specific data sources instead.
	includeCommits := !opts.Staged && !opts.Unstaged

	// 2. Get commits (only when no staged/unstaged flags are set)
	var logs []git.CommitLog
	if includeCommits {
		if opts.Base != "" {
			logs, err = s.git.Log(ctx, opts.Base, "HEAD")
			if err != nil {
				return nil, fmt.Errorf("reading commits since %s: %w", opts.Base, err)
			}
		} else {
			n := opts.Commits
			if n <= 0 {
				n = 10
			}
			logs, err = s.git.RecentCommits(ctx, n, "HEAD")
			if err != nil {
				return nil, fmt.Errorf("reading recent commits: %w", err)
			}
		}
	}

	// 3. Get staged diff if requested
	var stagedChange domain.Change
	if opts.Staged {
		stagedChange, err = s.git.DiffCached(ctx)
		if err != nil {
			return nil, fmt.Errorf("reading staged diff: %w", err)
		}
	}

	// 4. Get unstaged diff if requested
	var unstagedChange domain.Change
	if opts.Unstaged {
		unstagedChange, err = s.git.DiffUnstaged(ctx)
		if err != nil {
			return nil, fmt.Errorf("reading unstaged diff: %w", err)
		}
	}

	// 5. Scan for secrets
	if s.scan {
		if stagedDiff := stagedChange.Diff; stagedDiff != "" {
			cleaned, found := security.Scan(stagedDiff)
			if found {
				stagedChange.Diff = cleaned
			}
		}
		if unstagedDiff := unstagedChange.Diff; unstagedDiff != "" {
			cleaned, found := security.Scan(unstagedDiff)
			if found {
				unstagedChange.Diff = cleaned
			}
		}
	}

	// 6. Build prompt
	system, userPrompt, err := s.pmpt.Build(prompts.DescribePromptInput{
		Branch:   info.CurrentBranch,
		Base:     opts.Base,
		Staged:   stagedChange,
		Unstaged: unstagedChange,
		Commits:  logs,
	})
	if err != nil {
		return nil, fmt.Errorf("building prompt: %w", err)
	}

	// 7. Generate
	resp, err := s.ai.Generate(ctx, ai.Request{
		SystemPrompt: system,
		UserPrompt:   userPrompt,
	})
	if err != nil {
		return nil, fmt.Errorf("AI generation: %w", err)
	}

	// 8. Parse response
	desc := parseDescribeResponse(resp.Text)

	return &DescribeResult{
		Description:  desc,
		Provider:     s.ai.Name(),
		InputTokens:  resp.Usage.InputTokens,
		OutputTokens: resp.Usage.OutputTokens,
	}, nil
}

// parseDescribeResponse parses an AI response into DescribeChanges.
func parseDescribeResponse(text string) domain.DescribeChanges {
	text = cleanResponse(text)

	return domain.DescribeChanges{
		Overview:        extractSection(text, "Overview"),
		Commits:         extractBullets(text, "Commits"),
		StagedChanges:   extractBullets(text, "Staged Changes"),
		UnstagedChanges: extractBullets(text, "Unstaged Changes"),
	}
}

// extractSection extracts the content of a section from a markdown response.
// Sections are markdown headers like ## Overview.
func extractSection(text, name string) string {
	prefix := "## " + name
	idx := strings.Index(text, prefix)
	if idx < 0 {
		return ""
	}

	start := idx + len(prefix)
	// Skip to next line
	if rest := text[start:]; strings.HasPrefix(rest, "\n") {
		start++
	}

	// Find next section or end
	rest := text[start:]
	nextIdx := strings.Index(rest, "\n## ")
	if nextIdx >= 0 {
		rest = rest[:nextIdx]
	}

	return strings.TrimSpace(rest)
}

// extractBullets extracts bullet points from a section.
func extractBullets(text, sectionName string) []string {
	content := extractSection(text, sectionName)
	if content == "" {
		return nil
	}

	var bullets []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") {
			bullets = append(bullets, strings.TrimPrefix(line, "- "))
		} else if strings.HasPrefix(line, "* ") {
			bullets = append(bullets, strings.TrimPrefix(line, "* "))
		}
	}

	return bullets
}
