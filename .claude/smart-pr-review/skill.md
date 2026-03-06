---
name: smart-pr-review
description: Review pull requests locally with testing, feedback, and approve/request-changes workflow.
---

# Smart PR Review Skill

An intelligent pull request review workflow that helps you review PRs locally, run tests, leave feedback, and approve or request changes.

**Dependencies:** Requires `git` and `gh` CLI (GitHub CLI) to be installed and authenticated.
- Install: https://cli.github.com/
- Authenticate: `gh auth login`

## Workflow

Execute this workflow to review pull requests:

### Phase 1: Select PR to Review

#### 1.1 List Open PRs

```bash
gh pr list
```

**Display:**
```
📋 OPEN PULL REQUESTS

1. #123 feat: add YAML parsing support
   Author: @user
   Branch: feat/add-yaml-parser
   Status: ✅ All checks passed
   Reviews: 0 approved, 0 changes requested
   Updated: 2 hours ago

2. #120 fix: memory leak in parser
   Author: @contributor
   Branch: fix/memory-leak
   Status: ⏳ CI running
   Reviews: 1 approved, 0 changes requested
   Updated: 1 day ago

Select PR number to review (or 'exit'):
```

#### 1.2 Show PR Details

**Once user selects PR:**

```bash
gh pr view <number>
```

**Display:**
```
📝 PR #123: feat: add YAML parsing support

Author: @user
Branch: feat/add-yaml-parser → main
Status: ✅ Open
Checks: ✅ All passed

Description:
  Adds YAML parsing support with validation
  - Implements YAML parser
  - Adds validation logic
  - Updates documentation
  - Adds comprehensive tests

Files changed: 8 (+450, -120)
Commits: 3

Continue with review? (yes/no)
```

### Phase 2: Checkout and Inspect

#### 2.1 Handle Current State

**If uncommitted changes:**
```
⚠️  You have uncommitted changes.

Options:
  1. Save with smart-save
  2. Commit with smart-commit
  3. Stash changes
  4. Cancel review

Choose option:
```

#### 2.2 Checkout PR Branch

```bash
gh pr checkout <number>
```

**After checkout:**
```
✅ Checked out PR #123: feat/add-yaml-parser

Branch: feat/add-yaml-parser
Base: main
Files changed: 8
```

#### 2.3 Show Changed Files

```bash
gh pr diff <number> --name-only
```

**Display:**
```
📂 CHANGED FILES (8)

Source Code:
  M  src/parser.ts (+145, -30)
  A  src/yaml-handler.ts (+89, -0)
  M  src/types.ts (+23, -5)
  M  src/index.ts (+12, -8)

Tests:
  M  tests/parser.test.ts (+78, -15)
  A  tests/yaml.test.ts (+95, -0)

Documentation:
  M  README.md (+6, -2)
  M  docs/api.md (+2, -0)
```

### Phase 3: Review Options Menu

**Present review actions:**

```
🔍 REVIEW ACTIONS

Code Review:
  1. View full diff
  2. View file-by-file diff
  3. View specific file
  4. Show commits

Testing:
  5. Run tests locally
  6. Run linting
  7. Run type checking
  8. Run build

Feedback:
  9. Leave general comment
  10. Comment on specific file
  11. View existing comments

Decision:
  12. Approve PR
  13. Request changes
  14. Comment only
  15. Merge PR (if approved)

Navigation:
  16. Return to branch selection
  0. Exit review

Choose action (0-16):
```

### Phase 4: Code Review Actions

#### 4.1 View Diffs

**Full diff:**
```bash
gh pr diff <number>
```

**File-by-file:**
```bash
# Show list, let user select
gh pr diff <number> <file>
```

**Display with syntax highlighting and context**

#### 4.2 View Commits

```bash
git log main..HEAD --oneline --graph
```

**Show:**
- Commit messages
- Conventional commit compliance
- Logical organization

#### 4.3 View Specific File

**Interactive file selection:**
```
Select file to review:
  1. src/parser.ts
  2. src/yaml-handler.ts
  3. src/types.ts
  ...

Enter number:
```

**Then show:**
```bash
gh pr diff <number> <selected-file>
```

### Phase 5: Testing Actions

#### 5.1 Run Tests

**Detect test framework:**
```bash
# Python
if (Test-Path pytest.ini) { pytest }
if (Test-Path setup.py) { python -m pytest }

# Node.js
if (Test-Path package.json) { npm test }

# Other
# Check for common test commands
```

**Show results:**
```
🧪 Running tests...

✅ 45 tests passed
❌ 2 tests failed
⚠️  1 test skipped

Failed tests:
  - test_yaml_parser_null_handling
  - test_yaml_validation_edge_case

Run 'npm test -- --verbose' for details

Add to review feedback? (yes/no)
```

#### 5.2 Run Linting

```bash
# Python
ruff check .

# Node.js
npm run lint
```

**Show results and offer to add to feedback**

#### 5.3 Run Type Checking

```bash
# Python
mypy . || pyright .

# TypeScript
tsc --noEmit
```

#### 5.4 Run Build

```bash
# Python
python -m build

# Node.js
npm run build
```

**Track all results for final feedback**

### Phase 6: Leave Feedback

#### 6.1 General Comment

**Prompt for comment:**
```
Enter general comment about the PR:
(Type your feedback, press Enter twice when done)

>
```

**Post comment:**
```bash
gh pr comment <number> --body "<comment>"
```

#### 6.2 File-Specific Comment

**Select file and line:**
```
Select file for comment:
  1. src/parser.ts
  2. src/yaml-handler.ts
  ...

Enter file number:
```

**View file with line numbers, then:**
```
Enter line number for comment:
Enter your comment:

>
```

**Post comment:**
```bash
gh pr comment <number> --body "<comment>" --file <file> --line <line>
```

#### 6.3 Review Comments

**Show existing comments:**
```bash
gh pr view <number> --comments
```

**Display with context**

### Phase 7: Make Decision

#### 7.1 Approve PR

**Collect feedback:**
```
✅ APPROVE PR

Summary of your review:
  - Tests: ✅ Passed (45/47)
  - Linting: ✅ Clean
  - Build: ✅ Successful

Optional approval comment:
>
```

**Submit approval:**
```bash
gh pr review <number> --approve --body "<comment>"
```

**Confirmation:**
```
✅ PR #123 approved!

Next steps:
  - Merge PR (action 15)
  - Select another PR to review
  - Exit review
```

#### 7.2 Request Changes

**Collect issues:**
```
⚠️  REQUEST CHANGES

Issues found:
  - 2 test failures
  - Type error in yaml-handler.ts
  - Missing edge case handling

Summary comment:
>
```

**Submit review:**
```bash
gh pr review <number> --request-changes --body "<comment>"
```

**Confirmation:**
```
⚠️  Changes requested on PR #123

Feedback sent to author.
Author will be notified to address issues.
```

#### 7.3 Comment Only

**For non-blocking feedback:**
```bash
gh pr review <number> --comment --body "<comment>"
```

### Phase 8: Merge PR (if approved)

**Pre-merge checks:**
```
🔄 MERGE PR #123

Checks:
  ✅ All CI checks passed
  ✅ Required reviews: 1/1
  ✅ No conflicts
  ✅ Branch is up to date

Merge strategy:
  1. Merge commit (default)
  2. Squash and merge
  3. Rebase and merge

Choose strategy (1-3):
```

**Execute merge:**
```bash
# Merge commit
gh pr merge <number> --merge

# Squash
gh pr merge <number> --squash

# Rebase
gh pr merge <number> --rebase
```

**After merge:**
```
✅ PR #123 merged successfully!

Branch feat/add-yaml-parser merged into main

Next steps:
  - Delete branch: gh pr close <number> --delete-branch
  - Update your local main: git checkout main && git pull
  - Run smart-cleanup to clean up local branches
```

### Phase 9: Cleanup and Return

#### 9.1 Return to Original Branch

```bash
git checkout <original-branch>
```

#### 9.2 Offer Cleanup

**If PR was merged:**
```
Clean up PR branch locally? (yes/no)
```

**If yes:**
```bash
git branch -d feat/add-yaml-parser
```

#### 9.3 Summary

```
📊 REVIEW SESSION SUMMARY

Reviewed: 1 PR
  - #123: Approved and merged

Time spent: 15 minutes

Actions taken:
  - Ran tests locally
  - Left 2 comments
  - Approved PR
  - Merged with squash strategy

Want to review another PR? (yes/no)
```

## Advanced Features

### Batch Review Mode

**For multiple PRs:**
```bash
smart-pr-review --batch
```

- Review multiple PRs in sequence
- Track which ones reviewed
- Summary at end

### Quick Review Mode

**For simple PRs:**
```bash
smart-pr-review --quick <pr-number>
```

- Checkout PR
- Show diff
- Run tests
- Quick approve/reject

### Review Checklist

**Customizable checklist:**
```
📋 REVIEW CHECKLIST

Code Quality:
  [ ] Code follows project style
  [ ] No obvious bugs
  [ ] Proper error handling
  [ ] Clear variable names

Testing:
  [ ] Tests pass locally
  [ ] New tests added
  [ ] Edge cases covered

Documentation:
  [ ] README updated if needed
  [ ] Code comments clear
  [ ] API docs updated

Check items as you review
```

## Rules

### Never:
- Approve without reviewing changes
- Merge with failing tests
- Delete branches without confirmation
- Leave vague feedback
- Skip testing for non-trivial changes

### Always:
- Show PR context and details
- Run tests locally for code changes
- Provide specific, actionable feedback
- Confirm before merging
- Return to original branch after review
- Track review session

### Prefer:
- Testing locally over just reading code
- Specific line comments over general feedback
- Constructive suggestions over just criticism
- Approving quickly when appropriate
- Thorough review for complex changes

## Configuration

Check for `.smartreview.json` (optional):
```json
{
  "autoRunTests": true,
  "autoRunLint": false,
  "requireTestsPass": true,
  "defaultMergeStrategy": "squash",
  "checklist": [
    "Tests pass",
    "Code style consistent",
    "Documentation updated"
  ],
  "autoCleanupBranch": true
}
```

## Integration with Workflow

### Review Cycle
```bash
# Someone creates PR
smart-pull-request  # From their side

# You review
smart-pr-review  # From your side
→ Checkout PR
→ Run tests
→ Leave feedback

# They update
# (notified automatically)

# You re-review
smart-pr-review
→ Check updates
→ Approve
→ Merge

# Cleanup
smart-cleanup
```

### Team Workflow
```bash
# Morning routine
smart-status  # Check state
smart-pr-review --batch  # Review pending PRs
smart-branch  # Start new work
```

## Edge Cases

- **PR already checked out**: Offer to stay or switch
- **Conflicts detected**: Show conflict info, suggest author rebase
- **CI still running**: Warn, offer to wait or proceed
- **Required reviews not met**: Block merge, show requirements
- **No gh CLI**: Error with installation instructions
- **Not authenticated**: Prompt for `gh auth login`
- **Draft PR**: Show draft status, limited actions
- **Closed PR**: Show as closed, can't review
- **Force push during review**: Detect and warn
- **Large PR**: Warn if >500 lines, suggest splitting

## Output Examples

### Quick Approval
```
📋 PR #125: docs: fix typo in README

✅ Simple documentation fix
📂 Files: 1 (README.md)
✅ All checks passed

Quick approve? (yes/no): yes

✅ Approved and ready to merge!
```

### Request Changes
```
📋 PR #126: feat: add caching layer

⚠️  Issues found:

Tests:
  ❌ 3 tests failing
  - test_cache_invalidation
  - test_cache_ttl
  - test_concurrent_access

Code Review:
  ⚠️  Potential race condition in cache.ts:45
  ⚠️  Missing error handling in getCached()

📝 Requesting changes with detailed feedback...
✅ Feedback sent to author
```

## Invocation

User can trigger with:
- "smart pr review"
- "review pr"
- "review pull request"
- "check pr"

**With PR number:**
- "smart pr review 123"
- "review pr #123"
