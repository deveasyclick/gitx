# Product Specification


# Commands


## gitx commit

Generate commit messages from staged changes.


Example:

git add .

gitx commit


Flow:

1. Check staged changes.
2. Read git diff.
3. Generate commit message.
4. Display suggestion.
5. User approves.
6. Execute git commit.


Requirements:

- Must never commit automatically.
- Must support conventional commits.
- Must support custom commit formats.
- Must handle empty staged changes.


---

# gitx commit --group

Generate multiple commits from one large change.


Example:

20 changed files.

AI identifies:

Commit 1:
Authentication

Commit 2:
Payments

Commit 3:
Tests


Flow:

1. Analyze changed files.
2. Suggest groups.
3. User approves.
4. Stage selected files.
5. Create commits.


Safety:

Never:

- delete files
- reset files
- discard changes


---

# gitx pr

Generate pull request description.


Input:

- current branch
- commit history
- diff against base branch


Output:


## Summary

## Changes

## Testing

## Risks

## Breaking Changes



---

# gitx changelog

Generate changelog.


Input:

Git tags and commits.


Support:

gitx changelog --from v1.0.0

gitx changelog --latest


Output:

CHANGELOG.md


---

# gitx review

Review current changes.


Detect:

- bugs
- security issues
- missing tests
- performance issues


Output:

Review comments.


---

# gitx explain

Explain a diff.


Example:

gitx explain


Output:

Human readable explanation.