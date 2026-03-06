---
name: smart-merge
description: Safely merge the current branch into main with pre-flight checks, conflict handling, and optional push.
---

# Smart Merge Skill

**Dependencies:** Requires `git` to be installed.

An intelligent merge workflow with automatic branch detection, comprehensive safety checks, flexible merge strategies, and conflict resolution guidance.

## Workflow

Execute this workflow with optional parameters:

### Parameters

- `target=<branch>`: Target branch (auto-detected if not provided)
- `strategy=<merge|rebase|squash>`: Merge strategy (default: merge)
- `allow_dirty=<true|false>`: Allow merge with uncommitted changes (default: false)
- `run_tests=<true|false>`: Run tests after merge (default: false)
- `run_lint=<true|false>`: Run linting after merge (default: false)

### Phase 1: Intelligent Branch Detection

#### 1.1 Detect Current Branch

```bash
git branch --show-current
```

#### 1.2 Determine Target Branch

**Priority order:**

1. **User specified**: Use provided `target` parameter
2. **Upstream tracking**: Check for upstream branch
   ```bash
   git rev-parse --abbrev-ref --symbolic-full-name @{upstream}
   ```
3. **Common parent patterns**: Look for tracking branch
   ```bash
   # Check if parent/<current-branch> exists remotely
   git ls-remote --heads origin parent/<current-branch>
   ```
4. **Default branches**: Check for main/master
   ```bash
   git remote show origin | grep "HEAD branch"
   ```
5. **Prompt user**: If still unclear, show options and ask

**If remote-only parent exists:**
- Create local tracking branch: `git checkout -b parent/<current> origin/parent/<current>`
- Use as target

**Smart defaults:**
- Feature branches (feat/*, feature/*) → develop or main
- Hotfix branches (hotfix/*) → main
- Release branches (release/*) → main
- Other → main

### Phase 2: Safety Checks

#### 2.1 Check Working Directory

```bash
git status --porcelain
```

**If dirty (uncommitted changes):**
- Show changed files
- If `allow_dirty=false` (default): Stop with error
  - Message: "Uncommitted changes detected. Commit or stash them first, or use allow_dirty=true"
- If `allow_dirty=true`: Warn and proceed

#### 2.2 Protected Branch Warning

**If target is main or master:**
- Show warning: "⚠️  You are merging into protected branch: {target}"
- Show current branch commits: `git log {target}..HEAD --oneline`
- Require confirmation: "Continue? (yes/no)"
- If no: Abort

#### 2.3 Check Branch Status

```bash
git fetch origin
git rev-list --left-right --count {target}...HEAD
```

**Show status:**
- X commits ahead of {target}
- Y commits behind {target}

**If behind target:**
- Warn: "Target branch has new commits. Syncing first."
- Proceed to sync

#### 2.4 Sync Target Branch

```bash
git checkout {target}
git pull --ff-only origin {target}
```

**If fast-forward fails:**
- Error: "Target branch has diverged. Manual intervention required."
- Show: `git log origin/{target}..{target} --oneline`
- Stop and require user to resolve

**After sync:**
```bash
git checkout {source-branch}
```

### Phase 3: Execute Merge Strategy

User chooses strategy based on project needs.

#### Strategy 1: Merge (Default)

Creates a clear merge commit in history.

```bash
git merge --no-ff {target} -m "Merge {target} into {source-branch}"
```

**Benefits:**
- Preserves complete history
- Clear feature branch boundaries
- Easy to revert entire feature

**When to use:**
- Long-lived feature branches
- Multiple contributors on branch
- Want to preserve discussion context

#### Strategy 2: Rebase

Creates linear history by replaying commits.

```bash
git rebase {target}
```

**Benefits:**
- Clean linear history
- No merge commits
- Easier to follow chronologically

**When to use:**
- Short-lived feature branches
- Solo contributor
- Project prefers linear history

**Note:** Show warning if branch has been pushed (requires force push)

#### Strategy 3: Squash

Combines all commits into single commit.

```bash
git merge --squash {target}
```

**Generate conventional commit message:**

1. Analyze commits:
   ```bash
   git log {target}..HEAD --pretty=format:"%s"
   ```

2. Determine primary type:
   - Count commit types (feat, fix, refactor, etc.)
   - Use most common type
   - Default to `feat` if mixed

3. Generate scope:
   - Extract from commit scopes if consistent
   - Use changed directory as scope
   - Leave empty if unclear

4. Generate description:
   - Summarize feature from commit messages
   - Use imperative mood
   - Keep under 72 characters

5. Generate body:
   - List all original commits
   - Include breaking changes
   - Add co-authors if multiple contributors

**Example squash commit:**
```
feat(parser): add YAML support

- Add YAML parser implementation
- Add validation for YAML documents
- Update documentation for YAML usage
- Add comprehensive test coverage

Original commits:
- feat(parser): implement YAML parsing
- feat(parser): add YAML validation
- docs: update parser documentation
- test(parser): add YAML tests

Co-authored-by: User <user@example.com>
```

**Commit:**
```bash
git commit -m "<generated-message>"
```

**Benefits:**
- Clean single commit
- Conventional format
- Preserves attribution

**When to use:**
- Many small commits
- Experimental/WIP commits
- Want clean main branch history

### Phase 4: Conflict Resolution

**If conflicts occur:**

#### 4.1 Detect Conflicts

```bash
git status --porcelain | grep "^UU\|^AA\|^DD\|^AU\|^UA\|^DU\|^UD"
```

#### 4.2 Show Conflict Information

**Display:**
```
⚠️  Merge conflicts detected in 3 files:

1. src/parser.ts
   - Both modified
   - Lines 45-67

2. tests/parser.test.ts
   - Both modified
   - Lines 123-145

3. README.md
   - Both modified
   - Lines 12-15

Conflict markers (<<<<<<, =======, >>>>>>>) have been added to files.
```

#### 4.3 Provide Options

**Interactive menu:**
```
Choose an option:
1. Open conflicted files for manual resolution
2. Abort merge and return to previous state
3. Show conflict details with diff
4. Continue after resolving (mark as resolved)
5. Use their changes (accept target branch)
6. Use our changes (accept current branch)
```

#### 4.4 Guide Resolution

**For option 1 (manual resolution):**
- Open each conflicted file
- Show conflict context:
  ```bash
  git diff --ours --theirs {file}
  ```
- Wait for user to resolve
- Verify conflicts resolved: `git diff --check`
- Stage resolved files: `git add {file1} {file2} ...`
- Complete merge: `git commit` or `git rebase --continue`

**For option 2 (abort):**
```bash
# For merge
git merge --abort

# For rebase
git rebase --abort
```

**For option 5/6 (automated resolution):**
```bash
# Use their changes
git checkout --theirs {file}

# Use our changes
git checkout --ours {file}
```

Then stage and continue.

#### 4.5 Verify Resolution

After resolution:
```bash
# Check no conflicts remain
git diff --check

# Verify files stage correctly
git status

# Show summary
echo "✅ All conflicts resolved"
```

### Phase 5: Quality Gates (Optional)

#### 5.1 Run Tests (if run_tests=true)

**Follow STACK guidance - use scoped tests:**

**For Python:**
```bash
# Run tests only for changed files
pytest $(git diff --name-only {target}..HEAD | grep "\.py$" | sed 's/\.py$/.py/; s/^src/tests/')

# Or smart scope detection
pytest tests/ -k "$(git log {target}..HEAD --pretty=format:'%s' | grep -o 'test[A-Za-z]*' | sort -u | paste -sd '|')"
```

**For TypeScript/JavaScript:**
```bash
npm test -- --changedSince={target}
```

**If tests fail:**
- Show failure details
- Options:
  1. Fix tests and re-run
  2. Abort merge
  3. Continue anyway (with warning)

**If tests pass:**
- Show summary: "✅ X tests passed"
- Track coverage if available

#### 5.2 Run Linting (if run_lint=true)

**For Python:**
```bash
ruff check $(git diff --name-only {target}..HEAD | grep "\.py$")
```

**For TypeScript/JavaScript:**
```bash
npm run lint -- $(git diff --name-only {target}..HEAD | grep "\.\(ts\|js\|tsx\|jsx\)$")
```

**If linting issues:**
- Show issues
- Offer to auto-fix: `ruff check --fix` or `npm run lint -- --fix`
- If auto-fixed, commit: `git commit -m "style: fix linting issues after merge"`

### Phase 6: Final Merge to Target

If merging into target (not just sync from target):

```bash
git checkout {target}
git merge {source-branch}
```

**Prefer fast-forward when possible:**
```bash
# Check if fast-forward possible
if git merge-base --is-ancestor {target} {source-branch}; then
    git merge --ff-only {source-branch}
else
    git merge --no-ff {source-branch}
fi
```

### Phase 7: Report Summary

**Show complete summary:**
```
✅ Merge completed successfully!

Branch: feat/add-yaml-support → main
Strategy: merge (--no-ff)
Commits merged: 5
Files changed: 12 (+450, -120)

Changes:
  • src/parser.ts
  • src/yaml-handler.ts
  • tests/parser.test.ts
  • README.md

Quality gates:
  ✅ Tests passed (23 tests)
  ✅ Linting clean

Next steps:
  • Push changes: git push origin main
  • Delete feature branch: git branch -d feat/add-yaml-support
  • Delete remote branch: git push origin --delete feat/add-yaml-support
```

## Rules

### Never:
- Merge with failing tests (unless explicitly overridden)
- Use `git add .` - stage files explicitly
- Force push without warning
- Proceed with conflicts unresolved
- Merge into protected branch without confirmation
- Leave working directory in conflicted state

### Always:
- Detect target branch intelligently
- Sync target before merge
- Show ahead/behind status
- Provide clear conflict information
- Use conventional commits for squash
- Stage files explicitly
- Report what happened
- Offer next steps

### Prefer:
- Fast-forward when possible
- Scoped tests over full suite
- Interactive conflict resolution
- Clear user confirmations for risky operations

## Strategy Selection Guide

Help users choose the right strategy:

**Use `merge` when:**
- Multiple contributors on feature branch
- Want to preserve complete history
- Long-lived feature branch
- Need to maintain discussion context

**Use `rebase` when:**
- Solo contributor
- Short-lived feature
- Project requires linear history
- Branch hasn't been pushed/shared

**Use `squash` when:**
- Many small/WIP commits
- Want single commit in main branch
- Clean up experimental commits
- Conventional commit format required

## Configuration

Check for `.smartmerge.json` (optional):
```json
{
  "defaultStrategy": "merge",
  "protectedBranches": ["main", "master", "production"],
  "requireTests": true,
  "requireLint": false,
  "allowDirty": false,
  "autoSync": true,
  "defaultTarget": "main"
}
```

## Edge Cases

- **No upstream set**: Use smart detection or prompt
- **Remote-only parent**: Create local tracking branch
- **Diverged branches**: Require manual reconciliation
- **Protected branch**: Extra confirmation required
- **Detached HEAD**: Error and require branch checkout
- **Empty merge**: Detect and inform (already up-to-date)
- **Binary conflicts**: Show special handling instructions
- **Submodule conflicts**: Provide submodule-specific guidance
- **Large merge**: Warn if >100 files or >5000 lines changed
- **Merge already in progress**: Detect and offer to continue or abort

## Invocation

User can trigger with:
- "smart merge"
- "merge this branch"
- "merge into main"
- "rebase on develop"
- "squash and merge"

**With parameters:**
- "smart merge target=develop"
- "smart merge strategy=squash"
- "smart merge run_tests=true"
- "smart merge target=main strategy=squash run_tests=true"
