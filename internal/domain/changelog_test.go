package domain_test

import (
	"testing"

	"github.com/user/gitx/internal/domain"
)

func TestChangelogEntryIsEmpty(t *testing.T) {
	tests := []struct {
		name  string
		input domain.ChangelogEntry
		want  bool
	}{
		{
			name:  "empty entry",
			input: domain.ChangelogEntry{},
			want:  true,
		},
		{
			name: "version only",
			input: domain.ChangelogEntry{
				Version: "v1.2.0",
			},
			want: false,
		},
		{
			name: "added features only",
			input: domain.ChangelogEntry{
				Added: []string{"Payment retries"},
			},
			want: false,
		},
		{
			name: "fully populated entry",
			input: domain.ChangelogEntry{
				Version: "v1.2.0",
				Added:   []string{"Payment retries", "OAuth support"},
				Fixed:   []string{"Token refresh issue"},
				Changed: []string{"Updated dependencies"},
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
