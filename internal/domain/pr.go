package domain

// PullRequest represents a generated pull request description.
type PullRequest struct {
	BaseBranch    string   // target branch, e.g. "main"
	HeadBranch    string   // source branch, e.g. "feat/payment-retry"
	Summary       string   // high-level summary
	Changes       []string // bullet-point list of changes
	Testing       string   // testing notes
	Risks         string   // identified risks
	BreakingNotes string   // breaking change notes
}

// IsEmpty returns true when no content was generated.
func (p PullRequest) IsEmpty() bool {
	return p.Summary == "" && len(p.Changes) == 0
}
