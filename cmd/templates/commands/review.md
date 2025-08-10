---
allowed-tools: Bash, Read, Write, Edit, MultiEdit, Grep, Glob, LS
description: "Review and merge a Pull Request"
---

# Review Plan

As a QA engineer, your task is to review the Pull Request (PR) associated with the specified Issue and evaluate whether it meets all quality standards.

## Context

- Specification Driven Development: @.claude/osoba/docs/spacification_driven_development.md
- Target Issue number: $ARGUMENTS


## Workflow

### 1. Check the Issue

- Run `GH_PAGER= gh issue view <issue number>` to understand the context and requirements
- Identify the corresponding PR number

- Run `GH_PAGER= gh issue view <issue number>`  
  → Confirm the **issue content and requirements**

- Run `GH_PAGER= gh issue view <issue number> --comments`  
  → Review the **design document and task list**

⚠️ **Note**: When using `--comments`, the issue body may not be displayed correctly.  
Be sure to run the version *without* `--comments` first to understand the requirements.

### 2. Check the PR

- Run `GH_PAGER= gh pr view <PR number>` to review the purpose, changes, and description of the PR
- Ensure the implementation satisfies the original requirements

### 3. Review Code Changes

- Run `GH_PAGER= gh pr diff <PR number>` to check the code diff
- Evaluate the changes with the following criteria:
  - Compliance with coding standards
  - Presence and adequacy of test cases
  - Security concerns and potential vulnerabilities
  - Unnecessary diffs (e.g., debug code, commented-out lines)

### 4. Check CI Results

- Run `GH_PAGER= gh pr checks <PR number>` to verify CI status
  - All checks must ✅ pass
  - If checks are still running, wait and retry until completed

### 5. Post Review Result

- Post the review result using:
  `GH_PAGER= gh pr comment <PR number> --body "$(cat ./.tmp/review-result-<issue number>.md)"`
- Use the following template for `./.tmp/review-result-<issue number>.md`:

```markdown
## Review Result

- Issue: #<issue number>
- PR: #<PR number>

### ✅ Verdict
- [ ] Approved (LGTM)
- [ ] Requires changes

### 👍 Positive Notes
- [List of strengths in the implementation]

### 🛠 Suggestions for Improvement
- [List of specific recommendations]

### 🔍 Additional Notes
- [Optional remarks if any]
```

## Basic Rules

- Ensure compliance with coding conventions
- Confirm the implementation fully meets the issue requirements
- Check for any potential security issues
- All tests and CI checks must pass
- Review comments must be clear and constructive
