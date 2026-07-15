package domain_test

import (
	"testing"

	"github.com/user/gitx/internal/domain"
)

func TestChangeIsEmpty(t *testing.T) {
	tests := []struct {
		name  string
		input domain.Change
		want  bool
	}{
		{
			name:  "empty change",
			input: domain.Change{},
			want:  true,
		},
		{
			name: "change with files but no diff",
			input: domain.Change{
				Files: []string{"main.go"},
			},
			want: false,
		},
		{
			name: "change with diff but no files",
			input: domain.Change{
				Diff: "--- a/main.go\n+++ b/main.go\n@@ -1 +1 @@\n-foo\n+bar",
			},
			want: false,
		},
		{
			name: "fully populated change",
			input: domain.Change{
				Files:    []string{"main.go", "auth.go"},
				Diff:     "diff --git a/main.go b/main.go\nindex abc..def 100644\n--- a/main.go\n+++ b/main.go\n@@ -1 +1 @@\n-foo\n+bar",
				DiffStat: "2 files changed, 10 insertions(+), 2 deletions(-)",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.IsEmpty()
			if got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}
