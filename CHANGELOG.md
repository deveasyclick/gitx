# Changelog

All notable changes to this project will be documented in this file.

## v0.2.0

### Added
- Add `gitx release` command to orchestrate the full release workflow: compare branches, generate changelogs, suggest and bump versions, and create PR descriptions.
- Add AI-powered release changelog generation with an interactive Accept / Regenerate / Copy / Edit workflow.
- Add version bumping support for `package.json` and `package-lock.json`.
- Add `gitx update` command to self-update to the latest release, with sha256 checksum verification and atomic binary swap.
- Add automatic update checks on startup (opt out with `GITX_NO_UPDATE_CHECK=1`).
- Add `gitx version` command to print the build version.
- Add AI-based logical commit grouping to `gitx commit --group`.
- Add GitHub Pages deployment workflow.
- Support user-local installation without sudo via `GITX_INSTALL_DIR`.

### Changed
- Update golangci-lint action in CI to v9.
- Update install and uninstall documentation to cover both sudo and user-local installs.
- Add Taskfile with common development tasks.
