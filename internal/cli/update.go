package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/user/gitx/internal/ui"
	"github.com/user/gitx/internal/update"
)

func newUpdateCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update gitx to the latest version",
		Long: `Download and install the latest gitx release.

The release archive matching your OS and architecture is downloaded
from GitHub, verified against the published sha256 checksums, and
swapped in atomically: the previous binary is kept as a backup and
restored if anything fails.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return updateRun(cmd, yes)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "update without asking for confirmation")

	return cmd
}

func updateRun(cmd *cobra.Command, yes bool) error {
	if currentVersion == "" || currentVersion == "dev" {
		ui.PrintInfo("You're running a dev build — nothing to update.")
		return nil
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 90*time.Second)
	defer cancel()

	ui.PrintInfo("Checking for updates...")
	release, err := update.LatestRelease(ctx, updateOptions())
	if err != nil {
		ui.PrintError(err.Error())
		return fmt.Errorf("update: checking latest release")
	}

	if !update.IsNewer(release.TagName, currentVersion) {
		ui.PrintSuccess(fmt.Sprintf("Already up to date (%s)", currentVersion))
		return nil
	}

	ui.PrintInfo(fmt.Sprintf("New version available: %s (you have %s)", release.TagName, currentVersion))

	if !yes && ui.IsInteractive() {
		if !askUpdateConfirm() {
			ui.PrintInfo("Update cancelled.")
			return nil
		}
	}

	// Locate the asset for this platform and the checksum file.
	asset := update.FindAsset(release.Assets, release.TagName, runtime.GOOS, runtime.GOARCH)
	if asset == nil {
		ui.PrintError(fmt.Sprintf("No release asset for %s/%s. Available assets:\n  %s",
			runtime.GOOS, runtime.GOARCH, strings.Join(update.AssetNames(release.Assets), "\n  ")))
		return fmt.Errorf("update: unsupported platform")
	}
	checksums := update.FindAssetByName(release.Assets, "checksums.txt")
	if checksums == nil {
		ui.PrintError("checksums.txt not found in the release")
		return fmt.Errorf("update: missing checksums")
	}

	// Download and verify before touching anything.
	ui.PrintInfo(fmt.Sprintf("Downloading %s...", asset.Name))
	archive, err := update.Download(ctx, asset.URL)
	if err != nil {
		ui.PrintError(err.Error())
		return fmt.Errorf("update: download")
	}

	expected, err := update.ReleaseChecksum(ctx, checksums.URL, asset.Name)
	if err != nil {
		ui.PrintError(err.Error())
		return fmt.Errorf("update: checksums")
	}
	if err := update.VerifyChecksum(archive, expected); err != nil {
		ui.PrintError(fmt.Sprintf("Download failed verification: %s", err))
		return fmt.Errorf("update: checksum")
	}
	ui.PrintInfo("Checksum verified ✓")

	binary, err := update.ExtractBinary(archive, runtime.GOOS)
	if err != nil {
		ui.PrintError(err.Error())
		return fmt.Errorf("update: extract")
	}

	exe, err := os.Executable()
	if err != nil {
		ui.PrintError(fmt.Sprintf("Cannot locate the current binary: %s", err))
		return fmt.Errorf("update: locate binary")
	}

	if err := update.ReplaceBinary(exe, binary); err != nil {
		ui.PrintError(err.Error())
		ui.PrintInfo(fmt.Sprintf("Reinstall manually: curl -fsSL https://raw.githubusercontent.com/%s/main/scripts/install.sh | sh", gitxRepo))
		return fmt.Errorf("update: install")
	}

	update.CacheLatestVersion(updateCacheDir(), release.TagName)
	ui.PrintSuccess(fmt.Sprintf("Updated gitx to %s ✓", release.TagName))
	ui.PrintInfo("Restart any open shells to use the new version.")
	return nil
}

// updateOptions returns the update check options. GITX_UPDATE_BASE_URL can
// point the check at a mirror or a local test server.
func updateOptions() update.Options {
	opts := update.Options{Repo: gitxRepo}
	if base := os.Getenv("GITX_UPDATE_BASE_URL"); base != "" {
		opts.BaseURL = base
	}
	return opts
}

func askUpdateConfirm() bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Update now? [y/N] ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		switch input {
		case "y", "yes":
			return true
		case "n", "no", "":
			return false
		}
	}
}
