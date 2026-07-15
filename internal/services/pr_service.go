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

// PRService orchestrates pull request description generation.
type PRService struct {
	git  git.Client
	ai   aiProvider
	pmpt prompts.Builder[prompts.PRPromptInput]
	scan bool
}

// NewPRService creates a new PR service.
func NewPRService(gitClient git.Client, aiProvider aiProvider, prompt prompts.Builder[prompts.PRPromptInput]) *PRService {
	return &PRService{
		git:  gitClient,
		ai:   aiProvider,
		pmpt: prompt,
		scan: true,
	}
}

// PRResult contains the result of PR description generation.
type PRResult struct {
	Description  domain.PullRequest
	Provider     string
	InputTokens  int
	OutputTokens int
}

// Generate generates a PR description for the current branch against base.
func (s *PRService) Generate(ctx context.Context, base string) (*PRResult, error) {
	// 1. Get repo info
	info, err := s.git.RepoInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading repo info: %w", err)
	}

	// 2. Get diff against base
	change, err := s.git.Diff(ctx, base)
	if err != nil {
		return nil, fmt.Errorf("reading diff against %s: %w", base, err)
	}

	// 3. Scan for secrets
	if s.scan {
		cleaned, found := security.Scan(change.Diff)
		if found {
			change.Diff = cleaned
		}
	}

	// 4. Get commit log
	logs, err := s.git.Log(ctx, base, "HEAD")
	if err != nil {
		return nil, fmt.Errorf("reading commit log: %w", err)
	}

	// 5. Build prompt
	system, userPrompt, err := s.pmpt.Build(prompts.PRPromptInput{
		Diff:    change,
		Branch:  info.CurrentBranch,
		Base:    base,
		Commits: logs,
	})
	if err != nil {
		return nil, fmt.Errorf("building prompt: %w", err)
	}

	// 6. Generate
	resp, err := s.ai.Generate(ctx, ai.Request{
		SystemPrompt: system,
		UserPrompt:   userPrompt,
	})
	if err != nil {
		return nil, fmt.Errorf("AI generation: %w", err)
	}

	// 7. Parse response
	pr := parsePRDescription(resp.Text)

	return &PRResult{
		Description:  pr,
		Provider:     s.ai.Name(),
		InputTokens:  resp.Usage.InputTokens,
		OutputTokens: resp.Usage.OutputTokens,
	}, nil
}

// parsePRDescription parses an AI response into a PullRequest.
// This is a basic parser; the AI is instructed to follow the section format.
func parsePRDescription(text string) domain.PullRequest {
	text = cleanResponse(text)

	pr := domain.PullRequest{
		Summary: extractSection(text, "Summary"),
		Testing: extractSection(text, "Testing"),
		Risks:   extractSection(text, "Risks"),
	}
	if br := extractSection(text, "Breaking Changes"); br != "" {
		pr.BreakingNotes = br
	}

	// Extract bullet points from Changes section
	pr.Changes = extractBullets(text, "Changes")

	return pr
}

// extractSection extracts the content of a section from the response.
// Sections are markdown headers like ## Summary.
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
