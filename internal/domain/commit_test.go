package domain_test

import (
	"testing"

	"github.com/user/gitx/internal/domain"
)

func TestCommitMessageString(t *testing.T) {
	tests := []struct {
		name  string
		input domain.CommitMessage
		want  string
	}{
		{
			name: "title only",
			input: domain.CommitMessage{
				Title: "feat(auth): add refresh token support",
				Style: "conventional",
			},
			want: "feat(auth): add refresh token support",
		},
		{
			name: "title and body",
			input: domain.CommitMessage{
				Title: "feat(payment): add transaction retry logic",
				Body:  "Add retry handler\nImprove provider fallback",
				Style: "conventional",
			},
			want: "feat(payment): add transaction retry logic\n\nAdd retry handler\nImprove provider fallback",
		},
		{
			name: "empty message",
			input: domain.CommitMessage{
				Style: "conventional",
			},
			want: "",
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

func TestCommitMessageValidate(t *testing.T) {
	tests := []struct {
		name    string
		input   domain.CommitMessage
		wantErr error
	}{
		{
			name: "valid conventional commit",
			input: domain.CommitMessage{
				Title: "feat(auth): add refresh token support",
				Style: "conventional",
			},
			wantErr: nil,
		},
		{
			name: "valid title and body",
			input: domain.CommitMessage{
				Title: "fix: resolve nil pointer in parser",
				Body:  "The parser panicked when receiving empty input.",
				Style: "conventional",
			},
			wantErr: nil,
		},
		{
			name: "empty title",
			input: domain.CommitMessage{
				Title: "",
				Body:  "some body text",
				Style: "conventional",
			},
			wantErr: domain.ErrEmptyCommitTitle,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
