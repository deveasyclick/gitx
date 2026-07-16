# GitX - AI Powered Git Assistant

## Overview

GitX is an AI-powered command-line Git assistant designed to improve developer productivity.

It extends Git with intelligent workflows for:

- generating commit messages
- splitting changes into logical commits
- describing repository state
- generating changelogs
- reviewing changes
- explaining diffs

GitX does not replace Git.

Git remains the source of truth.

GitX provides intelligence and automation around existing Git workflows.

---

# Vision

Make Git workflows faster, clearer, and more consistent by combining:

- Git metadata
- developer intent
- AI models
- automation

---

# Goals

## Primary Goals

1. Build a production-quality CLI tool.
2. Support multiple AI providers.
3. Provide excellent developer experience.
4. Maintain user control over all Git operations.
5. Be safe by default.

---

# Non Goals

GitX will NOT:

- replace Git
- host repositories
- automatically commit without approval
- automatically push code
- modify source code without user permission
- store user repositories remotely

---

# Target Users

Software engineers who:

- use Git daily
- work with pull requests
- maintain open-source projects
- want better commit history
- want faster documentation generation

---

# Technology

Language:

Go

Version:

Go 1.26+

Distribution:

Single binary CLI

Supported platforms:

- Linux
- macOS
- Windows