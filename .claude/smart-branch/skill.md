---
name: smart-branch
description: Create properly named feature branches with type detection, validation, and optional remote tracking setup.
---

**Dependencies:** Requires `git` to be installed.

## Workflow

Execute this workflow when starting new work:

### Phase 1: Context Gathering

#### 1.1 Check Current State

```bash
git status --porcelain
git branch --show-current
```

**Verify:**
- Not in detached HEAD state
- Working directory status (warn if dirty)

**If uncommitted changes exist:**
- Show changed files
- Options:
  1. Stash changes and proceed
  2. Commit changes first
  3. Continue anyway (with warning)

#### 1.2 Ask User Intent

**Prompt:** "What are you working on?"

**User provides description**, for example:
- "adding YAML parser support"
- "fixing bug with null values"
- "updating documentation"
- "refactoring authentication module"
- "improving database query performance"

### Phase 2: Intelligent Type Detection

Analyze the user's description to determine branch type:

**Detection patterns:**

| Type | Keywords | Example Input |
|------|----------|---------------|
| `feat` | add, adding, new, implement, create | "adding YAML parser" |
| `fix` | fix, bug, issue, resolve, patch | "fixing null value bug" |
| `docs` | document, documentation, readme, guide | "updating documentation" |
| `refactor` | refactor, restructure, reorganize, cleanup | "refactoring auth module" |
| `perf` | performance, optimize, speed, improve speed | "improving query performance" |
| `test` | test, testing, coverage | "adding test coverage" |
| `chore` | dependency, deps, update, upgrade, maintain | "updating dependencies" |
| `ci` | ci, pipeline, workflow, github actions | "adding CI pipeline" |
| `style` | style, format, lint | "fixing code formatting" |

**Default to `feat` if unclear**

### Phase 3: Branch Name Generation

#### 3.1 Generate Suggestion

**Format:** `<type>/<descriptive-name>`

**Rules:**
- Use detected type as prefix
- Convert description to kebab-case
- Maximum 50 characters
- Use imperative mood
- Remove articles (a, an, the)
- Remove common words (with, for, to, from)
- Keep meaningful keywords only

**Examples:**

| Input | Suggested Branch |
|-------|-----------------|
| "adding YAML parser support" | `feat/add-yaml-parser` |
| "fixing bug with null values" | `fix/null-values` |
| "updating documentation for API" | `docs/update-api-docs` |
| "refactoring authentication module" | `refactor/authentication` |
| "improving database query performance" | `perf/database-queries` |
| "adding test coverage for validators" | `test/validator-coverage` |

#### 3.2 Present Suggestion

**Display:**
```
Detected type: feat
Suggested branch: feat/add-yaml-parser

Use this branch name? (yes/no/custom)
```

**Options:**
- `yes`: Use suggested name
- `no`: Provide alternative suggestion
- `custom`: User provides custom name

**If custom:**
- Prompt: "Enter branch name:"
- Validate format (no spaces, special chars, etc.)
- Confirm with user

### Phase 4: Branch Validation

#### 4.1 Check Local Branches

```bash
git branch --list <branch-name>
```

**If exists locally:**
- Option 1: Checkout existing branch
- Option 2: Create with suffix (e.g., `-v2`, `-alt`)
- Option 3: Enter different name

#### 4.2 Check Remote Branches

```bash
git ls-remote --heads origin <branch-name>
```

**If exists remotely:**
- Option 1: Checkout and track remote branch
- Option 2: Create with suffix
- Option 3: Enter different name

#### 4.3 Validate Name Format

**Rules:**
- No spaces
- No special characters except `-`, `/`, `_`
- Must start with letter
- Cannot end with `-` or `/`
- Maximum 60 characters

### Phase 5: Branch Creation

#### 5.1 Ensure Up-to-Date Base

```bash
git fetch origin
git checkout main  # or default branch
git pull --ff-only origin main
```

**If fast-forward fails:**
- Warn user about diverged main
- Options:
  1. Force pull (destructive)
  2. Continue anyway
  3. Abort

#### 5.2 Create and Checkout Branch

```bash
git checkout -b <branch-name>
```

#### 5.3 Set Up Tracking (Optional)

**Prompt:** "Push branch and set up tracking now? (yes/no)"

**If yes:**
```bash
git push -u origin <branch-name>
```

**Benefits:**
- Immediate backup
- Enables collaboration
- Shows branch in GitHub

### Phase 6: Optional Initial Commit

**Prompt:** "Create initial commit with task description? (yes/no)"

**If yes:**

Create `.task.md` file:
```markdown
# Task: <User Description>

**Type:** <type>
**Branch:** <branch-name>
**Started:** <timestamp>

## Description
<User's description>

## Goals
- [ ] 

## Notes
```

**Commit:**
```bash
git add .task.md
git commit -m "chore: initialize task - <description>"
```

**If tracking enabled, push:**
```bash
git push
```

### Phase 7: Summary

**Display:**
```
✅ Branch created successfully!

Branch: feat/add-yaml-parser
Type: feat
Description: adding YAML parser support
Tracking: Yes (pushed to origin)
Initial commit: Yes (.task.md created)

You're ready to start working!

Tips:
- Use 'smart-commit' to organize your commits
- Use 'smart-pull-request' when ready to create a PR
- Use 'smart-merge' to merge back to main
```

## Rules

### Never:
- Create branch without user confirmation
- Overwrite existing branches without warning
- Create from dirty working tree without acknowledgment
- Use spaces or invalid characters in branch names
- Create from detached HEAD state

### Always:
- Detect type from user description
- Suggest conventional branch names
- Check for existing branches (local and remote)
- Validate branch name format
- Update from main/default branch first
- Provide clear summary of what was created

### Prefer:
- Kebab-case for branch names
- Descriptive but concise names
- Conventional type prefixes
- Tracking setup for remote backup
- Optional task file for context

## Branch Naming Conventions

### Format
```
<type>/<short-description>
```

### Valid Types
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `refactor`: Code restructuring
- `perf`: Performance improvement
- `test`: Testing
- `chore`: Maintenance
- `ci`: CI/CD changes
- `style`: Code style/formatting

### Examples of Good Names
- `feat/user-authentication`
- `fix/memory-leak`
- `docs/api-guide`
- `refactor/database-layer`
- `perf/query-optimization`
- `test/integration-tests`
- `chore/update-dependencies`
- `ci/automated-testing`

### Examples of Bad Names
- `my-branch` (no type)
- `feat/fix something` (spaces)
- `feat/this-is-a-very-long-branch-name-that-should-be-shorter` (too long)
- `feature/add-stuff` (use feat, not feature; too vague)

## Edge Cases

- **Already on feature branch**: Offer to stay or create new branch
- **Detached HEAD**: Error and require checkout of branch first
- **Diverged main**: Warn and offer options
- **No network connection**: Skip remote checks
- **No git repository**: Error and suggest `git init`
- **Uncommitted changes**: Offer to stash or commit
- **Branch name conflicts**: Suggest alternatives with suffixes
- **Multiple remotes**: Ask which remote to use for tracking
- **No default branch set**: Prompt for base branch

## Configuration

Check for `.smartbranch.json` (optional):
```json
{
  "autoTrack": true,
  "createTaskFile": true,
  "baseBaranch": "main",
  "namingPattern": "<type>/<description>",
  "maxLength": 50,
  "defaultType": "feat",
  "autoUpdate": true
}
```

## Invocation

User can trigger with:
- "smart branch"
- "start new work"
- "create a branch"
- "new feature branch"
- "start working on"

**With description:**
- "smart branch - adding YAML support"
- "start new work on fixing bug"
