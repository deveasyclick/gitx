package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FindAsset returns the release asset for the given OS/arch, or nil. Asset
// names follow goreleaser's template "gitx_<version>_<os>_<arch>.(tar.gz|zip)".
func FindAsset(assets []Asset, version, goos, goarch string) *Asset {
	prefix := fmt.Sprintf("gitx_%s_%s_%s.", strings.TrimPrefix(version, "v"), goos, goarch)
	for i := range assets {
		if strings.HasPrefix(assets[i].Name, prefix) {
			return &assets[i]
		}
	}
	return nil
}

// FindAssetByName returns the asset with the exact given name, or nil.
func FindAssetByName(assets []Asset, name string) *Asset {
	for i := range assets {
		if assets[i].Name == name {
			return &assets[i]
		}
	}
	return nil
}

// AssetNames returns the names of all assets, for error messages.
func AssetNames(assets []Asset) []string {
	names := make([]string, len(assets))
	for i, a := range assets {
		names[i] = a.Name
	}
	return names
}

// Download fetches the file at url and returns its contents.
func Download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("update: creating download request: %w", err)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("update: downloading %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("update: downloading %s: unexpected status %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// ReleaseChecksum downloads a goreleaser checksums.txt and returns the sha256
// expected for the named asset.
func ReleaseChecksum(ctx context.Context, checksumsURL, assetName string) (string, error) {
	data, err := Download(ctx, checksumsURL)
	if err != nil {
		return "", err
	}
	return parseChecksum(string(data), assetName)
}

// parseChecksum finds the "<sha256>  <filename>" line for assetName.
func parseChecksum(checksums, assetName string) (string, error) {
	for _, line := range strings.Split(checksums, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == assetName {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("update: checksum for %s not found", assetName)
}

// VerifyChecksum checks data against an expected hex sha256.
func VerifyChecksum(data []byte, expected string) error {
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if !strings.EqualFold(got, strings.TrimSpace(expected)) {
		return fmt.Errorf("sha256 mismatch: got %s, want %s", got, expected)
	}
	return nil
}

// ExtractBinary pulls the gitx executable out of a release archive (tar.gz,
// or zip on Windows).
func ExtractBinary(archive []byte, goos string) ([]byte, error) {
	if goos == "windows" {
		return extractFromZip(archive)
	}
	return extractFromTarGz(archive)
}

func extractFromTarGz(data []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("update: opening gzip archive: %w", err)
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("update: reading tar archive: %w", err)
		}
		if hdr.Typeflag == tar.TypeReg && filepath.Base(hdr.Name) == "gitx" {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("update: gitx binary not found in archive")
}

func extractFromZip(data []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("update: opening zip archive: %w", err)
	}
	for _, f := range zr.File {
		if filepath.Base(f.Name) == "gitx.exe" {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("update: opening %s: %w", f.Name, err)
			}
			out, readErr := io.ReadAll(rc)
			_ = rc.Close()
			if readErr != nil {
				return nil, fmt.Errorf("update: reading %s: %w", f.Name, readErr)
			}
			return out, nil
		}
	}
	return nil, fmt.Errorf("update: gitx.exe not found in archive")
}

// ReplaceBinary atomically swaps the file at currentPath with newBinary. The
// current binary is moved aside first and restored if the swap fails, so a
// failed update never leaves a broken installation.
func ReplaceBinary(currentPath string, newBinary []byte) error {
	dir := filepath.Dir(currentPath)

	// Write the new binary to a temp file in the same directory so the final
	// rename stays on one filesystem.
	tmp, err := os.CreateTemp(dir, ".gitx-update-*")
	if err != nil {
		return fmt.Errorf("update: creating temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if err := os.Chmod(tmpName, 0o755); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("update: chmod temp file: %w", err)
	}
	if _, err := tmp.Write(newBinary); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("update: writing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("update: closing temp file: %w", err)
	}

	backup := currentPath + ".bak"
	_ = os.Remove(backup)

	if err := os.Rename(currentPath, backup); err != nil {
		return fmt.Errorf("update: backing up current binary (is %s writable?): %w", currentPath, err)
	}
	if err := os.Rename(tmpName, currentPath); err != nil {
		// Roll back to the previous binary.
		_ = os.Rename(backup, currentPath)
		return fmt.Errorf("update: installing new binary: %w", err)
	}
	_ = os.Remove(backup)
	return nil
}
