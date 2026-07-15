package domain

// CommitMessage represents a generated git commit message.
type CommitMessage struct {
	Title string // first line, e.g. "feat(auth): add refresh token support"
	Body  string // optional multi-line body
	Style string // "conventional", "gitmoji", or "custom"
}

// String returns the full commit message as formatted text.
func (m CommitMessage) String() string {
	if m.Body == "" {
		return m.Title
	}
	return m.Title + "\n\n" + m.Body
}

// Validate returns an error if the message is missing required fields.
func (m CommitMessage) Validate() error {
	if m.Title == "" {
		return ErrEmptyCommitTitle
	}
	return nil
}
