package domain_test

import (
	"testing"

	"github.com/user/gitx/internal/domain"
)

func TestPullRequestIsEmpty(t *testing.T) {
	tests := []struct {
		name  string
		input domain.PullRequest
		want  bool
	}{
		{
			name:  "empty PR",
			input: domain.PullRequest{},
			want:  true,
		},
		{
			name: "PR with summary only",
			input: domain.PullRequest{
				Summary: "Adds payment retry support.",
			},
			want: false,
		},
		{
			name: "PR with changes only",
			input: domain.PullRequest{
				Changes: []string{"Added retry service", "Improved provider fallback"},
			},
			want: false,
		},
		{
			name: "fully populated PR",
			input: domain.PullRequest{
				BaseBranch:    "main",
				HeadBranch:    "feat/payment-retry",
				Summary:       "Adds payment retry support.",
				Changes:       []string{"Added retry service"},
				Testing:       "Unit tests added",
				Risks:         "None identified",
				BreakingNotes: "",
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
