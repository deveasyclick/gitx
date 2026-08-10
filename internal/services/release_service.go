package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/user/gitx/internal/ai"
	"github.com/user/gitx/internal/domain"
	"github.com/user/gitx/internal/git"
	"github.com/user/gitx/internal/prompts"
)

// ReleaseService orchestrates the release workflow.
type ReleaseService struct {
	git  git.Client
	ai   aiProvider
	pmpt *prompts.ReleaseBuilder
}

// NewReleaseService creates a new release service.
func NewReleaseService(gitClient git.Client, aiProvider aiProvider, prompt *prompts.ReleaseBuilder) *ReleaseService {
	return &ReleaseService{
		git:  gitClient,
		ai:   aiProvider,
		pmpt: prompt,
	}
}

// CompareResult holds the result of comparing two branches.
type CompareResult struct {
	Info domain.ReleaseInfo
	Diff domain.Change
}

// CompareBranches compares head branch against base and returns the differences.
func (s *ReleaseService) CompareBranches(ctx context.Context, base, head string) (*CompareResult, error) {
	// Get repo info
	repoInfo, err := s.git.RepoInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("repo info: %w", err)
	}

	// Resolve branches
	if base == "" {
		return nil, fmt.Errorf("base branch is required")
	}
	if head == "" {
		head = repoInfo.CurrentBranch
	}
	if head == base {
		return nil, fmt.Errorf("head branch %q is the same as base branch %q", head, base)
	}

	// Get commits in head that are not in base
	logs, err := s.git.Log(ctx, base, head)
	if err != nil {
		// Try the other direction
		return nil, fmt.Errorf("comparing %s..%s: %w", base, head, err)
	}

	// Get diff between base and head
	diff, err := s.git.Diff(ctx, base)
	if err != nil {
		// If diff fails, still proceed with just log info
		diff = domain.Change{}
	}

	// Determine current version from tags
	tags, err := s.git.Tags(ctx)
	if err != nil {
		tags = nil
	}

	currentVersion := ""
	var latestTag string
	if len(tags) > 0 {
		latestTag = tags[0]
		for _, t := range tags {
			currentVersion = t
			break
		}
	}

	// For the range, use base..head commits
	// Suggest next version based on latest tag
	nextVersion := SuggestNextVersion(currentVersion, "minor")

	// Convert git.CommitLog to domain.CommitSummary to avoid circular imports
	commitSummaries := toCommitSummaries(logs)

	info := domain.ReleaseInfo{
		BaseBranch:     base,
		HeadBranch:     head,
		CurrentVersion: currentVersion,
		NextVersion:    nextVersion,
		LatestTag:      latestTag,
		Commits:        commitSummaries,
		Diff:           diff,
	}

	return &CompareResult{
		Info: info,
		Diff: diff,
	}, nil
}

// SuggestVersion uses AI to suggest a version bump type.
func (s *ReleaseService) SuggestVersion(ctx context.Context, commits []domain.CommitSummary) (domain.VersionBump, error) {
	system, user, err := s.pmpt.BuildVersionSuggestion(commits)
	if err != nil {
		return "", fmt.Errorf("building version prompt: %w", err)
	}

	resp, err := s.ai.Generate(ctx, ai.Request{
		SystemPrompt: system,
		UserPrompt:   user,
	})
	if err != nil {
		return "", fmt.Errorf("AI suggestion: %w", err)
	}

	text := strings.TrimSpace(strings.ToLower(resp.Text))
	switch {
	case strings.Contains(text, "major"):
		return domain.BumpMajor, nil
	case strings.Contains(text, "minor"):
		return domain.BumpMinor, nil
	case strings.Contains(text, "patch"):
		return domain.BumpPatch, nil
	default:
		return domain.BumpMinor, nil
	}
}

// toCommitSummaries converts git.CommitLog entries to domain.CommitSummary.
func toCommitSummaries(logs []git.CommitLog) []domain.CommitSummary {
	commitSummaries := make([]domain.CommitSummary, len(logs))
	for i, l := range logs {
		commitSummaries[i] = domain.CommitSummary{
			Hash:    l.Hash,
			Author:  l.Author,
			Message: l.Message,
		}
	}
	return commitSummaries
}

// GenerateReleaseChangelog generates changelog text for a release.
func (s *ReleaseService) GenerateReleaseChangelog(ctx context.Context, info domain.ReleaseInfo, version string) (string, error) {
	// A release changelog should cover everything since the last release tag,
	// not just commits ahead of the base branch (base..head). base..head can
	// miss unreleased commits that landed on the base branch after the last
	// tag, which would otherwise silently drop real changes from the changelog.
	commits := info.Commits
	diff := info.Diff
	if s.git != nil && info.LatestTag != "" && info.HeadBranch != "" {
		if sinceTag, err := s.git.Log(ctx, info.LatestTag, info.HeadBranch); err == nil && len(sinceTag) > 0 {
			commits = toCommitSummaries(sinceTag)
		}
		// The diff is the source of truth for the changelog: use the
		// cumulative diff since the last release tag, not the base..head diff.
		if sinceTagDiff, err := s.git.DiffRange(ctx, info.LatestTag, info.HeadBranch); err == nil && !sinceTagDiff.IsEmpty() {
			diff = sinceTagDiff
		}
	}

	input := prompts.ReleaseChangelogInput{
		Version:       version,
		PreviousTag:   info.LatestTag,
		CurrentTag:    version,
		Commits:       commits,
		CommitDetails: diff.DiffStat,
		Diff:          diff.Diff,
	}

	system, user, err := s.pmpt.BuildReleaseChangelog(input)
	if err != nil {
		return "", fmt.Errorf("building changelog prompt: %w", err)
	}

	resp, err := s.ai.Generate(ctx, ai.Request{
		SystemPrompt: system,
		UserPrompt:   user,
	})
	if err != nil {
		return "", fmt.Errorf("AI changelog generation: %w", err)
	}

	return strings.TrimSpace(resp.Text), nil
}

// GeneratePRDescription generates a PR description for the release.
func (s *ReleaseService) GeneratePRDescription(ctx context.Context, info domain.ReleaseInfo, version, changelogText string) (string, error) {
	input := prompts.ReleasePRInput{
		Version:       version,
		BaseBranch:    info.BaseBranch,
		HeadBranch:    info.HeadBranch,
		ChangelogText: changelogText,
		Commits:       info.Commits,
		FilesChanged:  info.Diff.Files,
		DiffStat:      info.Diff.DiffStat,
	}

	system, user, err := s.pmpt.BuildPRDescription(input)
	if err != nil {
		return "", fmt.Errorf("building PR prompt: %w", err)
	}

	resp, err := s.ai.Generate(ctx, ai.Request{
		SystemPrompt: system,
		UserPrompt:   user,
	})
	if err != nil {
		return "", fmt.Errorf("AI PR description generation: %w", err)
	}

	return strings.TrimSpace(resp.Text), nil
}

// UpdateChangelogFile prepends or creates a CHANGELOG.md with the new entry.
// If an entry for the same version already exists (e.g. because the release
// command was run more than once), it is replaced in place instead of being
// duplicated.
func (s *ReleaseService) UpdateChangelogFile(changelogPath string, version string, newEntry string) error {
	dir := filepath.Dir(changelogPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	header := "# Changelog\n\nAll notable changes to this project will be documented in this file.\n\n"
	// Normalize the entry: "## <version>\n\n<entry>\n"
	entryBlock := fmt.Sprintf("## %s\n\n%s\n", version, strings.TrimSpace(newEntry))

	// Check if file exists
	existing, err := os.ReadFile(changelogPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("reading %s: %w", changelogPath, err)
		}
		// File doesn't exist — create it
		return os.WriteFile(changelogPath, []byte(header+entryBlock), 0o644)
	}

	text := string(existing)

	// If an entry for this version already exists, replace it instead of
	// prepending a duplicate.
	if updated, ok := replaceChangelogEntry(text, version, entryBlock); ok {
		return os.WriteFile(changelogPath, []byte(updated), 0o644)
	}

	// File exists without an entry for this version — prepend after the header.
	var content string
	if strings.HasPrefix(text, "# Changelog") {
		// Find the first ## or end of file after header
		headerEnd := strings.Index(text, "\n## ")
		if headerEnd < 0 {
			// No existing entries — insert after the header block
			lines := strings.SplitN(text, "\n", 4)
			if len(lines) >= 3 {
				content = lines[0] + "\n" + lines[1] + "\n" + lines[2] + "\n\n" + entryBlock + strings.Join(lines[3:], "\n")
			} else {
				content = text + "\n" + entryBlock
			}
		} else {
			// Insert after header but before first existing entry
			content = text[:headerEnd] + "\n" + entryBlock + text[headerEnd:]
		}
	} else {
		// No standard header — just prepend
		content = header + entryBlock + "\n" + text
	}

	return os.WriteFile(changelogPath, []byte(content), 0o644)
}

// replaceChangelogEntry finds every existing "## <version>" section in text
// and swaps them for the single entryBlock, returning the updated text and
// true. If the version has no entry yet, it returns ok=false with text
// unchanged. Replacing in place (rather than always prepending) keeps
// re-running the release command from duplicating the entry, and repairs
// duplicates already present in the file.
func replaceChangelogEntry(text, version, entryBlock string) (string, bool) {
	re := regexp.MustCompile(`(?m)^##\s+` + regexp.QuoteMeta(version) + `\s*$`)

	// Each "## <version>" section runs from its header line to the next
	// "## " header (or the end of the file).
	var firstStart, lastEnd int
	found := false
	searchFrom := 0
	for {
		loc := re.FindStringIndex(text[searchFrom:])
		if loc == nil {
			break
		}
		start := searchFrom + loc[0]
		bodyStart := searchFrom + loc[1]
		end := len(text)
		if next := strings.Index(text[bodyStart:], "\n## "); next >= 0 {
			end = bodyStart + next
			// Drop the blank line left between this entry and the next header.
			for end > start && text[end-1] == '\n' {
				end--
			}
		}
		if !found {
			firstStart = start
			found = true
		}
		lastEnd = end
		searchFrom = end
	}

	if !found {
		return text, false
	}

	updated := text[:firstStart] + strings.TrimRight(entryBlock, "\n") + "\n" + text[lastEnd:]
	return updated, true
}

// BumpVersionInFiles updates version strings in known files.
// Currently supports: package.json, package-lock.json
// Returns list of files modified.
func (s *ReleaseService) BumpVersionInFiles(version string) ([]string, error) {
	version = strings.TrimPrefix(version, "v")
	var modified []string

	// package.json
	if _, err := os.Stat("package.json"); err == nil {
		if err := bumpJSONVersion("package.json", version); err != nil {
			return modified, fmt.Errorf("bumping package.json: %w", err)
		}
		modified = append(modified, "package.json")
	}

	// package-lock.json
	if _, err := os.Stat("package-lock.json"); err == nil {
		if err := bumpJSONVersion("package-lock.json", version); err != nil {
			return modified, fmt.Errorf("bumping package-lock.json: %w", err)
		}
		modified = append(modified, "package-lock.json")
	}

	return modified, nil
}

// bumpJSONVersion reads a JSON file, updates the "version" field, and writes it back.
func bumpJSONVersion(path, version string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Simple regex-based version replacement to avoid importing a JSON library
	// Match "version": "anything" and replace
	re := regexp.MustCompile(`"version"\s*:\s*"[^"]+"`)
	newData := re.ReplaceAll(data, []byte(fmt.Sprintf(`"version": "%s"`, version)))

	if string(data) == string(newData) {
		return fmt.Errorf("no version field found in %s", path)
	}

	return os.WriteFile(path, newData, 0o644)
}

// SuggestNextVersion takes a current version (e.g., "v1.2.0" or "1.2.0") and a bump type,
// and returns the next version string with "v" prefix.
func SuggestNextVersion(current string, bump string) string {
	if current == "" {
		return "v0.1.0"
	}

	current = strings.TrimPrefix(current, "v")
	parts := strings.Split(current, ".")
	if len(parts) != 3 {
		return "v" + current
	}

	major, minor, patch := parts[0], parts[1], parts[2]

	switch bump {
	case "major":
		return fmt.Sprintf("v%s.%s.%s", nextNum(major), "0", "0")
	case "minor":
		return fmt.Sprintf("v%s.%s.%s", major, nextNum(minor), "0")
	case "patch":
		return fmt.Sprintf("v%s.%s.%s", major, minor, nextNum(patch))
	default:
		return fmt.Sprintf("v%s.%s.%s", major, nextNum(minor), "0")
	}
}

func nextNum(s string) string {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return "1" // no leading number — treat the segment as 0
	}
	return fmt.Sprintf("%d", n+1)
}

// ComputeDiffStat returns a human-readable stat from the diff.
func ComputeDiffStat(diff domain.Change) string {
	if diff.DiffStat != "" {
		return diff.DiffStat
	}
	return fmt.Sprintf("%d file(s) changed", len(diff.Files))
}
