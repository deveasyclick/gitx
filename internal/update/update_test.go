package update

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIsNewer(t *testing.T) {
	tests := []struct {
		latest  string
		current string
		want    bool
	}{
		{"v1.2.3", "v1.1.0", true},
		{"v1.2.3", "v1.2.3", false},
		{"v1.2.3", "v1.3.0", false},
		{"v2.0.0", "v1.9.9", true},
		{"1.2.3", "v1.2.2", true},  // no v prefix
		{"V1.2.3", "v1.2.2", true}, // uppercase V
		{"v1.2.3", "dev", false},   // unparseable current
		{"not-a-version", "v1.0.0", false},
		{"v1.2.3", "v1.2", false}, // two segments only
		{"v1.2.3", "", false},
		{"v1.2.3", "v1.2.3-beta", true},  // stable beats prerelease of same version
		{"v1.2.3-beta", "v1.2.3", false}, // prerelease never beats stable
		{"v1.3.0-beta", "v1.2.9", true},  // prerelease can beat an older stable
		{"v1.2.4", "v1.2.4-rc.1", true},  // stable beats rc of same version
		{"v1.2.4-rc.1", "v1.2.4", false}, // rc never beats stable
		{"v1.2.4-rc.1", "v1.2.3", true},  // rc can beat an older stable
	}
	for _, tt := range tests {
		if got := IsNewer(tt.latest, tt.current); got != tt.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", tt.latest, tt.current, got, tt.want)
		}
	}
}

func TestLatestVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/deveasyclick/gitx/releases/latest" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v1.2.3"})
	}))
	defer srv.Close()

	got, err := LatestVersion(context.Background(), Options{Repo: "deveasyclick/gitx", BaseURL: srv.URL})
	if err != nil {
		t.Fatalf("LatestVersion: %v", err)
	}
	if got != "v1.2.3" {
		t.Errorf("got %q, want v1.2.3", got)
	}
}

func TestLatestVersion_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	if _, err := LatestVersion(context.Background(), Options{Repo: "deveasyclick/gitx", BaseURL: srv.URL}); err == nil {
		t.Error("expected error for non-200 response")
	}
}

func TestLatestVersion_NoRepo(t *testing.T) {
	if _, err := LatestVersion(context.Background(), Options{}); err == nil {
		t.Error("expected error when repo is empty")
	}
}

func TestCheck_FetchesLatest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v1.2.3"})
	}))
	defer srv.Close()

	opts := Options{Repo: "deveasyclick/gitx", BaseURL: srv.URL}

	// Behind → returns the latest version.
	latest, err := Check(context.Background(), "v1.1.0", opts)
	if err != nil {
		t.Fatalf("Check (behind): %v", err)
	}
	if latest != "v1.2.3" {
		t.Errorf("got %q, want v1.2.3", latest)
	}

	// Up to date → empty.
	latest, err = Check(context.Background(), "v1.2.3", opts)
	if err != nil {
		t.Fatalf("Check (current): %v", err)
	}
	if latest != "" {
		t.Errorf("got %q, want empty when up to date", latest)
	}
}

func TestCheck_UsesCache(t *testing.T) {
	cacheDir := t.TempDir()
	_ = writeCache(cacheDir, "v9.9.9")

	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer srv.Close()

	latest, err := Check(context.Background(), "v1.1.0", Options{
		Repo:     "deveasyclick/gitx",
		BaseURL:  srv.URL,
		CacheDir: cacheDir,
		TTL:      time.Hour,
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if latest != "v9.9.9" {
		t.Errorf("got %q, want cached v9.9.9", latest)
	}
	if called {
		t.Error("a fresh cache hit should not hit the network")
	}
}

func TestCacheReadWrite(t *testing.T) {
	cacheDir := t.TempDir()

	if err := writeCache(cacheDir, "v1.2.3"); err != nil {
		t.Fatalf("writeCache: %v", err)
	}
	got, ok := readCache(cacheDir, time.Hour)
	if !ok || got != "v1.2.3" {
		t.Errorf("readCache = (%q, %v), want (v1.2.3, true)", got, ok)
	}

	// Expired TTL → treated as a miss.
	if _, ok := readCache(cacheDir, time.Nanosecond); ok {
		t.Error("expired cache entry should be a miss")
	}

	// Missing file → miss.
	if _, ok := readCache(t.TempDir(), time.Hour); ok {
		t.Error("missing cache file should be a miss")
	}
}
