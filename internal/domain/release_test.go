package domain_test

import (
	"testing"

	"github.com/user/gitx/internal/domain"
)

func TestChangelogSection_String(t *testing.T) {
	tests := []struct {
		name  string
		input domain.ChangelogSection
		want  string
	}{
		{
			name: "full section",
			input: domain.ChangelogSection{
				Version: "v1.2.0",
				Added:   []string{"New feature A", "New feature B"},
				Fixed:   []string{"Bug fix"},
				Changed: []string{"Updated module"},
				Removed: []string{"Deprecated API"},
			},
			want: "## v1.2.0\n\n### Added\n- New feature A\n- New feature B\n\n### Fixed\n- Bug fix\n\n### Changed\n- Updated module\n\n### Removed\n- Deprecated API\n\n",
		},
		{
			name: "no version",
			input: domain.ChangelogSection{
				Added: []string{"Feature"},
			},
			want: "### Added\n- Feature\n\n",
		},
		{
			name:  "empty section",
			input: domain.ChangelogSection{},
			want:  "",
		},
		{
			name: "only some sections",
			input: domain.ChangelogSection{
				Version: "v2.0.0",
				Added:   []string{"Major feature"},
				Fixed:   []string{"Critical fix"},
			},
			want: "## v2.0.0\n\n### Added\n- Major feature\n\n### Fixed\n- Critical fix\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVersionBumpConstants(t *testing.T) {
	if domain.BumpMajor != "major" {
		t.Errorf("BumpMajor = %q, want %q", domain.BumpMajor, "major")
	}
	if domain.BumpMinor != "minor" {
		t.Errorf("BumpMinor = %q, want %q", domain.BumpMinor, "minor")
	}
	if domain.BumpPatch != "patch" {
		t.Errorf("BumpPatch = %q, want %q", domain.BumpPatch, "patch")
	}
}

func TestReleaseActionDefaults(t *testing.T) {
	action := domain.ReleaseAction{}
	if action.UseReleaseBranch {
		t.Error("UseReleaseBranch should default to false")
	}
	if action.ReleaseBranch != "" {
		t.Errorf("ReleaseBranch = %q, want empty", action.ReleaseBranch)
	}
	if action.CommitChanges {
		t.Error("CommitChanges should default to false")
	}
	if action.CreateTag {
		t.Error("CreateTag should default to false")
	}
	if action.TagName != "" {
		t.Errorf("TagName = %q, want empty", action.TagName)
	}
}

func TestCommitSummaryConversion(t *testing.T) {
	// CommitSummary is used to avoid circular imports between domain and git.
	// Verify it carries the expected fields.
	summary := domain.CommitSummary{
		Hash:    "abc123def456",
		Author:  "test-user",
		Message: "feat: add payment retries",
	}

	if summary.Hash != "abc123def456" {
		t.Errorf("Hash = %q", summary.Hash)
	}
	if summary.Author != "test-user" {
		t.Errorf("Author = %q", summary.Author)
	}
	if summary.Message != "feat: add payment retries" {
		t.Errorf("Message = %q", summary.Message)
	}
}
