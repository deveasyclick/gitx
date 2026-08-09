# Changelog

All notable changes to this project will be documented in this file.

## v0.1.1

### Added
- Add `gitx release` command to orchestrate the full release workflow: compare branches, generate changelogs, suggest and bump versions, and create PR descriptions.
- Add AI-powered release changelog generation with an interactive Accept / Regenerate / Copy / Edit workflow.
- Add version bumping support for `package.json` and `package-lock.json`.
- Add GitHub Pages deployment workflow.
- Support user-local installation without sudo via `GITX_INSTALL_DIR`.

### Changed
- Update golangci-lint action from v6 to v7 in CI.
- Update install and uninstall documentation to cover both sudo and user-local installs.
