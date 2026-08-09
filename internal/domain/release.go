package domain

// ReleaseInfo holds the context for a release operation.
type ReleaseInfo struct {
	BaseBranch     string
	HeadBranch     string
	CurrentVersion string // e.g. "v1.2.0" — "" if no tags exist
	NextVersion    string // suggested version for this release
	LatestTag      string // most recent tag before head (for range comparison)
	Commits        []CommitSummary
	Diff           Change
}

// CommitSummary is a lightweight commit representation for domain types.
// Avoids importing git package to prevent circular dependencies.
type CommitSummary struct {
	Hash    string
	Author  string
	Message string
}

// VersionBump describes the type of version increment.
type VersionBump string

const (
	BumpMajor VersionBump = "major"
	BumpMinor VersionBump = "minor"
	BumpPatch VersionBump = "patch"
)

// ReleaseAction tracks what the user decided to do.
type ReleaseAction struct {
	UseReleaseBranch bool   // create a release/vX.Y.Z branch?
	ReleaseBranch    string // e.g. "release/v1.3.0"
	CommitChanges    bool   // commit the version/changelog changes?
	CreateTag        bool   // create a git tag?
	TagName          string // e.g. "v1.3.0"
}

// ChangelogSection is one section in a changelog.
type ChangelogSection struct {
	Version string
	Added   []string
	Fixed   []string
	Changed []string
	Removed []string
}

// String formats a changelog section as markdown.
func (s ChangelogSection) String() string {
	var result string
	if s.Version != "" {
		result += "## " + s.Version + "\n\n"
	}
	writeSection := func(name string, items []string) {
		if len(items) == 0 {
			return
		}
		result += "### " + name + "\n"
		for _, item := range items {
			result += "- " + item + "\n"
		}
		result += "\n"
	}
	writeSection("Added", s.Added)
	writeSection("Fixed", s.Fixed)
	writeSection("Changed", s.Changed)
	writeSection("Removed", s.Removed)
	return result
}
