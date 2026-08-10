// Package update checks the latest gitx release on GitHub and compares it
// against the installed version. Callers must fail open: an update check
// should never block or fail a user command, so network and parse errors are
// returned as errors and expected to be ignored (or surfaced as a verbose
// note at most).
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Options configures an update check.
type Options struct {
	Repo     string        // GitHub repository, e.g. "deveasyclick/gitx"
	BaseURL  string        // API base URL; defaults to https://api.github.com
	CacheDir string        // directory for the check cache; empty disables caching
	TTL      time.Duration // how long a cached check stays valid; defaults to 24h
	Timeout  time.Duration // HTTP request timeout; defaults to 5s
}

// Check returns the latest release version when it is strictly newer than
// current, and "" otherwise. A fresh cached result avoids the network.
func Check(ctx context.Context, current string, opts Options) (string, error) {
	if latest, ok := readCache(opts.CacheDir, opts.TTL); ok {
		if IsNewer(latest, current) {
			return latest, nil
		}
		return "", nil
	}

	latest, err := LatestVersion(ctx, opts)
	if err != nil {
		return "", err
	}
	_ = writeCache(opts.CacheDir, latest)

	if IsNewer(latest, current) {
		return latest, nil
	}
	return "", nil
}

// Release is a GitHub release with its download assets.
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// Asset is a downloadable file attached to a release.
type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

// LatestRelease fetches the latest non-prerelease release from the GitHub
// Releases API, including its assets.
func LatestRelease(ctx context.Context, opts Options) (Release, error) {
	base := opts.BaseURL
	if base == "" {
		base = "https://api.github.com"
	}
	if opts.Repo == "" {
		return Release{}, fmt.Errorf("update: repo is required")
	}
	url := fmt.Sprintf("%s/repos/%s/releases/latest", base, opts.Repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Release{}, fmt.Errorf("update: creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	client := &http.Client{Timeout: timeout}

	resp, err := client.Do(req)
	if err != nil {
		return Release{}, fmt.Errorf("update: fetching latest release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("update: unexpected status %s", resp.Status)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return Release{}, fmt.Errorf("update: decoding response: %w", err)
	}
	return release, nil
}

// LatestVersion fetches the tag of the latest non-prerelease release.
func LatestVersion(ctx context.Context, opts Options) (string, error) {
	release, err := LatestRelease(ctx, opts)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(release.TagName), nil
}

// CacheLatestVersion records the latest known version so subsequent checks
// skip the network until the entry expires.
func CacheLatestVersion(cacheDir, latest string) {
	_ = writeCache(cacheDir, latest)
}

// IsNewer reports whether latest is strictly newer than current. Both are
// compared as (major, minor, patch); a prerelease of the same version number
// is older than its stable release. Returns false when either side cannot be
// parsed as a plain numeric version.
func IsNewer(latest, current string) bool {
	lm, lmi, lp, lpr, lok := parseVersion(latest)
	cm, ci, cp, cpr, cok := parseVersion(current)
	if !lok || !cok {
		return false
	}
	if lm != cm {
		return lm > cm
	}
	if lmi != ci {
		return lmi > ci
	}
	if lp != cp {
		return lp > cp
	}
	// Same (major, minor, patch): a stable release is newer than a prerelease.
	if lpr == cpr {
		return false
	}
	return !lpr
}

// parseVersion parses "v1.2.3" (optionally with a prerelease suffix such as
// "-beta" or "-rc.1") into (1, 2, 3, isPrerelease).
func parseVersion(v string) (major, minor, patch int, prerelease bool, ok bool) {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")

	// A suffix like "-beta" or "-rc.1" marks a prerelease.
	if idx := strings.IndexAny(v, "-+"); idx >= 0 {
		v = v[:idx]
		prerelease = true
	}

	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return 0, 0, 0, false, false
	}

	var nums [3]int
	for i, p := range parts {
		if p == "" {
			return 0, 0, 0, false, false
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return 0, 0, 0, false, false
		}
		nums[i] = n
	}

	return nums[0], nums[1], nums[2], prerelease, true
}

// ── Cache ────────────────────────────────────────────────────────────────

// cacheFileName stores the last update-check result.
const cacheFileName = "update.json"

type cacheEntry struct {
	CheckedAt time.Time `json:"checked_at"`
	Latest    string    `json:"latest"`
}

func cachePath(cacheDir string) string {
	if cacheDir == "" {
		return ""
	}
	return filepath.Join(cacheDir, cacheFileName)
}

// readCache returns the cached latest version when it is still fresh.
func readCache(cacheDir string, ttl time.Duration) (string, bool) {
	path := cachePath(cacheDir)
	if path == "" {
		return "", false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return "", false
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	if entry.Latest == "" || time.Since(entry.CheckedAt) > ttl {
		return "", false
	}
	return entry.Latest, true
}

func writeCache(cacheDir, latest string) error {
	path := cachePath(cacheDir)
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}
	entry := cacheEntry{CheckedAt: time.Now(), Latest: latest}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
