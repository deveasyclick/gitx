package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindAsset(t *testing.T) {
	assets := []Asset{
		{Name: "gitx_1.2.3_linux_amd64.tar.gz", URL: "u1"},
		{Name: "gitx_1.2.3_darwin_arm64.tar.gz", URL: "u2"},
		{Name: "gitx_1.2.3_windows_amd64.zip", URL: "u3"},
		{Name: "checksums.txt", URL: "u4"},
	}

	if got := FindAsset(assets, "v1.2.3", "linux", "amd64"); got == nil || got.URL != "u1" {
		t.Errorf("FindAsset(linux/amd64) = %+v, want u1", got)
	}
	if got := FindAsset(assets, "v1.2.3", "windows", "amd64"); got == nil || got.URL != "u3" {
		t.Errorf("FindAsset(windows/amd64) = %+v, want u3", got)
	}
	// Unbuilt platform → nil.
	if got := FindAsset(assets, "v1.2.3", "linux", "386"); got != nil {
		t.Errorf("FindAsset(linux/386) = %+v, want nil", got)
	}
	if got := FindAssetByName(assets, "checksums.txt"); got == nil || got.URL != "u4" {
		t.Errorf("FindAssetByName(checksums.txt) = %+v, want u4", got)
	}
}

func TestParseChecksum(t *testing.T) {
	checksums := "deadbeef  gitx_1.2.3_linux_amd64.tar.gz\ncafebabe  gitx_1.2.3_darwin_arm64.tar.gz\n"
	got, err := parseChecksum(checksums, "gitx_1.2.3_linux_amd64.tar.gz")
	if err != nil {
		t.Fatalf("parseChecksum: %v", err)
	}
	if got != "deadbeef" {
		t.Errorf("got %q, want deadbeef", got)
	}
	if _, err := parseChecksum(checksums, "missing.tar.gz"); err == nil {
		t.Error("expected error for missing asset")
	}
}

func TestVerifyChecksum(t *testing.T) {
	data := []byte("hello gitx")
	sum := sha256.Sum256(data)
	want := hex.EncodeToString(sum[:])

	if err := VerifyChecksum(data, want); err != nil {
		t.Errorf("VerifyChecksum(valid) = %v, want nil", err)
	}
	if err := VerifyChecksum(append(data, 'x'), want); err == nil {
		t.Error("VerifyChecksum(tampered) should fail")
	}
}

func TestExtractBinary_TarGz(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	content := []byte("#!/bin/sh\nfake gitx binary\n")
	if err := tw.WriteHeader(&tar.Header{Name: "gitx", Mode: 0o755, Size: int64(len(content))}); err != nil {
		t.Fatalf("WriteHeader: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("Write: %v", err)
	}
	_ = tw.Close()
	_ = gz.Close()

	got, err := ExtractBinary(buf.Bytes(), "linux")
	if err != nil {
		t.Fatalf("ExtractBinary: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("extracted %q, want %q", got, content)
	}
}

func TestExtractBinary_Zip(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	content := []byte("fake gitx.exe")
	w, err := zw.Create("gitx.exe")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := w.Write(content); err != nil {
		t.Fatalf("Write: %v", err)
	}
	_ = zw.Close()

	got, err := ExtractBinary(buf.Bytes(), "windows")
	if err != nil {
		t.Fatalf("ExtractBinary(windows): %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("extracted %q, want %q", got, content)
	}
}

func TestExtractBinary_Missing(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	content := []byte("some other file")
	if err := tw.WriteHeader(&tar.Header{Name: "README.md", Size: int64(len(content))}); err != nil {
		t.Fatalf("WriteHeader: %v", err)
	}
	_, _ = tw.Write(content)
	_ = tw.Close()
	_ = gz.Close()

	if _, err := ExtractBinary(buf.Bytes(), "linux"); err == nil {
		t.Error("expected error when the archive has no gitx binary")
	}
}

func TestDownload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "archive-bytes")
	}))
	defer srv.Close()

	got, err := Download(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if string(got) != "archive-bytes" {
		t.Errorf("got %q", got)
	}

	srv404 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv404.Close()
	if _, err := Download(context.Background(), srv404.URL); err == nil {
		t.Error("expected error on non-200")
	}
}

func TestReplaceBinary(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "gitx")
	old := []byte("old binary")
	if err := os.WriteFile(target, old, 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	new := []byte("new binary v1.2.3")
	if err := ReplaceBinary(target, new); err != nil {
		t.Fatalf("ReplaceBinary: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(got, new) {
		t.Errorf("target = %q, want %q", got, new)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Errorf("binary should be executable, got %v", info.Mode().Perm())
	}
	// No leftover backup or temp files.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.Contains(e.Name(), ".bak") || strings.Contains(e.Name(), ".gitx-update-") {
			t.Errorf("leftover file: %s", e.Name())
		}
	}
}

func TestReplaceBinary_ReadOnlyDir(t *testing.T) {
	// ReplaceBinary must fail cleanly (and leave the original intact) when the
	// directory is not writable.
	dir := t.TempDir()
	target := filepath.Join(dir, "gitx")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	if err := ReplaceBinary(target, []byte("new")); err == nil {
		t.Error("expected error for read-only directory")
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "old" {
		t.Errorf("original binary should be intact, got %q, err %v", got, err)
	}
}
