package domain

// ChangelogEntry represents a single version's changelog section.
type ChangelogEntry struct {
	Version string   // e.g. "v1.2.0"
	Added   []string // new features
	Fixed   []string // bug fixes
	Changed []string // modifications
	Removed []string // deprecated removals
}

// IsEmpty returns true when no version or changes exist.
func (e ChangelogEntry) IsEmpty() bool {
	return e.Version == "" &&
		len(e.Added) == 0 &&
		len(e.Fixed) == 0 &&
		len(e.Changed) == 0 &&
		len(e.Removed) == 0
}
