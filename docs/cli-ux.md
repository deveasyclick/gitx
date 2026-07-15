# GitX CLI UX Specification

## Purpose

This document defines the command-line interface behavior for GitX.

It specifies:

- commands
- flags
- arguments
- output format
- interactive flows
- error handling
- exit codes

The CLI should be:

- predictable
- safe
- fast
- easy to understand

---

# General Principles

## Safety First

GitX must never perform destructive actions without confirmation.

The following require confirmation:

- git commit
- staging files
- creating multiple commits
- modifying CHANGELOG.md


---

## Output Style

GitX uses three output levels:

### Normal

Human-readable output.


Example:

```

Analyzing changes...

Generated commit:

feat(auth): add refresh token support

* Add refresh endpoint
* Improve token validation

```


---

### Verbose

Enabled with:

```

--verbose

```


Shows:

- git commands executed
- AI provider
- model used
- timing information


---

### JSON

Enabled with:

```

--json

```


Used for:

- scripts
- CI pipelines
- integrations


Example:

```json
{
  "command": "commit",
  "message": "feat(auth): add refresh token support"
}
```

---

# Global Flags

Available on every command:

```
--verbose

--json

--no-color

--version

--help
```

---

# Command Structure

```
gitx <command> [arguments] [flags]
```

Available commands:

```
gitx commit

gitx pr

gitx changelog

gitx config

gitx doctor
```

Future:

```
gitx review

gitx explain

gitx release
```

---

# Command: gitx commit

## Purpose

Generate a commit message from staged changes.

Usage:

```
gitx commit
```

---

## Preconditions

Required:

* Git repository
* staged changes

If no staged changes:

Output:

```
No staged changes found.

Run:

git add <files>

then retry.
```

Exit:

```
1
```

---

## Flow

### Step 1

Read staged diff.

Output:

```
Analyzing staged changes...
```

---

### Step 2

Generate message.

Output:

```
Generated commit:

feat(payment): add transaction retry logic


Changes:
- Add retry handler
- Improve provider fallback
```

---

### Step 3

Confirmation

Prompt:

```
Commit this change?

[Y] Yes
[N] No
[E] Edit
[R] Regenerate
```

---

## Edit Flow

User selects:

```
E
```

Open editor:

```
$EDITOR
```

After save:

Commit.

---

## Regenerate Flow

User selects:

```
R
```

Generate a new message.

Maximum:

```
3 attempts
```

---

## Flags

### --dry-run

Generate but do not commit.

Example:

```
gitx commit --dry-run
```

Output:

```
Generated commit:

feat(auth): add OAuth support
```

---

### --provider

Override AI provider.

Example:

```
gitx commit --provider ollama
```

---

### --model

Override model.

Example:

```
gitx commit --model llama3
```

---

# Command: gitx pr

## Purpose

Generate pull request description.

Usage:

```
gitx pr
```

---

## Input

GitX collects:

* current branch
* commit history
* diff against base branch

---

## Output

```
Pull Request Description


## Summary

Adds payment retry support.


## Changes

- Added retry service
- Improved provider handling


## Testing

- Unit tests added


## Risks

None identified
```

---

## Flags

### --base

Specify base branch.

Example:

```
gitx pr --base develop
```

Default:

```
main
```

---

### --output

Write to file.

Example:

```
gitx pr --output pr.md
```

---

# Command: gitx changelog

## Purpose

Generate changelog entries.

Usage:

```
gitx changelog
```

---

## Flow

Collect:

```
git tags

git commits

commit messages
```

Generate:

```
## v1.2.0


### Added

- Payment retries


### Fixed

- Token refresh issue
```

---

## Flags

```
--from

--to

--output

--latest
```

Examples:

```
gitx changelog --latest
```

```
gitx changelog --from v1.0.0 --to v1.1.0
```

---

# Command: gitx config

## Purpose

Manage configuration.

Usage:

```
gitx config
```

---

## Set

Example:

```
gitx config set ai.provider openai
```

---

## Get

Example:

```
gitx config get ai.provider
```

Output:

```
openai
```

---

## List

Example:

```
gitx config list
```

Output:

```
ai.provider=openai

ai.model=gpt-5-mini

commit.style=conventional
```

---

# Command: gitx doctor

## Purpose

Diagnose installation.

Usage:

```
gitx doctor
```

Checks:

```
✓ Git installed

✓ Repository detected

✓ Config found

✓ AI provider configured
```

Failure:

```
✗ OpenAI API key missing

Run:

gitx config set ai.api_key <key>
```

---

# Error Handling

Errors should be:

* actionable
* short
* explain next step

Bad:

```
Error 500
```

Good:

```
Unable to generate commit message.

Reason:
OpenAI API key missing.

Fix:

gitx config set ai.api_key <key>
```

---

# Exit Codes

```
0
Success


1
User error


2
Configuration error


3
Git error


4
AI provider error


5
Unexpected error
```

---

# Interactive UI Rules

Prompts must:

* explain the action
* show available options
* provide safe defaults

Example:

```
Commit generated.

Commit this change?

[Y] Yes
[N] No


(default: No)
```

---

# Colors

Color usage:

Green:

Success

Yellow:

Warning

Red:

Error

Blue:

Information

Colors can be disabled:

```
--no-color
```

---

# Future Commands

Reserved:

```
gitx review

gitx explain

gitx release

gitx history
```

Do not implement until core workflow is stable.

```

---

This document gives the LLM enough constraints to implement the CLI consistently.

The next document after this should be **TASKS.md**, because now we have:

- architecture boundaries ✅
- user-facing behavior ✅

The task breakdown can now map directly to commands:
