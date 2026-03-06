---
name: smart-cleanup
description: Identify and remove merged, stale, and unnecessary branches both locally and remotely.
---

# Smart Cleanup Skill

**Dependencies:** Requires `git` to be installed.

An intelligent branch cleanup workflow that identifies and removes merged, stale, and unnecessary branches both locally and remotely. Keeps your repository clean and organized.

## Workflow

Execute this workflow after merges or periodically to maintain clean repository state:

### Phase 1: Safety Check & Discovery

#### 1.1 Verify Current State

```bash
git branch --show-current
git status --porcelain
```

**Verify:**
- Not in detached HEAD state
- Working directory is clean (warn if dirty)

**If on feature branch being cleaned:**
- Checkout main/default branch first
- Warn user about switch

#### 1.2 Update Remote Information

```bash
git fetch --prune origin
git remote prune origin
```

**This removes:**
- Stale remote tracking branches
- References to deleted remote branches

#### 1.3 Discover Branches

```bash
# Local branches
git branch

# Remote branches
git branch -r

# Merged branches (local)
git branch --merged main

# Merged branches (remote)
git branch -r --merged main
```

### Phase 2: Categorize Branches

Analyze and categorize all branches:

#### 2.1 Protected Branches (Never Delete)

- `main`, `master`
- `develop`, `development`
- `staging`, `production`
- Branches matching config patterns

#### 2.2 Merged Branches (Safe to Delete)

**Local merged:**
```bash
git branch --merged main | grep -v "main\|master\|develop"
```

**Remote merged:**
```bash
git branch -r --merged main | grep -v "main\|master\|develop"
```

**Criteria:**
- Fully merged into main/default branch
- Not a protected branch
- Commits are in main history

#### 2.3 Stale Branches (Candidate for Deletion)

**Criteria:**
- No commits in last 30 days (configurable)
- No open PR associated
- Not current branch

```bash
# Check last commit date
git for-each-ref --sort=-committerdate refs/heads/ --format='%(refname:short) %(committerdate:relative)'
```

#### 2.4 Active Branches (Keep)

- Current branch
- Branches with commits in last 30 days
- Branches with open PRs
- Recently created (<7 days)

#### 2.5 Gone Remote Branches (Local tracking deleted remote)

```bash
git branch -vv | grep ': gone]'
```

**These are local branches where remote was deleted**

### Phase 3: Present Cleanup Plan

#### 3.1 Show Categorized Report

```
🧹 Branch Cleanup Report

✅ MERGED BRANCHES (Safe to delete)
Local (3):
  ✓ feat/add-yaml-parser (merged 2 days ago)
  ✓ fix/null-handling (merged 1 week ago)
  ✓ docs/update-readme (merged 3 days ago)

Remote (2):
  ✓ origin/feat/add-yaml-parser (merged 2 days ago)
  ✓ origin/fix/null-handling (merged 1 week ago)

⚠️  STALE BRANCHES (No activity in 30+ days)
Local (1):
  ⚠️  experiment/new-approach (45 days old, no PR)

Remote (1):
  ⚠️  origin/old-feature (60 days old)

🔄 GONE REMOTES (Local tracking deleted remote)
  🔄 hotfix/quick-fix (remote deleted)
  🔄 feat/abandoned (remote deleted)

✨ ACTIVE BRANCHES (Keep)
  ✨ feat/current-work (last commit: 2 hours ago)
  ✨ feat/add-smart-branch-skill (current branch)

🔒 PROTECTED BRANCHES (Never delete)
  🔒 main
  🔒 develop
```

#### 3.2 Offer Cleanup Options

**Interactive menu:**
```
Choose cleanup action:
1. Delete all merged branches (local + remote)
2. Delete merged branches (local only)
3. Delete merged branches (remote only)
4. Delete stale branches (with confirmation)
5. Delete gone remote branches
6. Custom selection (choose specific branches)
7. Show details for a branch
8. Exit without changes
```

### Phase 4: Execute Cleanup

#### 4.1 Delete Merged Branches

**For each branch selected:**

**Local deletion:**
```bash
git branch -d <branch-name>  # Safe delete (merged check)
```

**If not merged but user insists:**
```bash
git branch -D <branch-name>  # Force delete (with warning)
```

**Remote deletion:**
```bash
git push origin --delete <branch-name>
```

**Track results:**
- ✅ Deleted successfully
- ⚠️  Not fully merged (require confirmation)
- ❌ Failed (show error)

#### 4.2 Delete Stale Branches (with confirmation)

**For each stale branch:**
1. Show details:
   - Last commit date
   - Last commit message
   - Commit count ahead of main
   - PR status

2. Ask confirmation: "Delete stale branch '<name>'? (yes/no/skip)"

3. If yes:
   ```bash
   git branch -D <branch-name>  # Force delete
   git push origin --delete <branch-name>  # If exists remotely
   ```

#### 4.3 Delete Gone Remote Branches

**These are safe to delete:**
```bash
git branch -d <branch-name>
```

**Or bulk:**
```bash
git branch -vv | grep ': gone]' | awk '{print $1}' | xargs git branch -d
```

### Phase 5: Cleanup Summary

**Show detailed report:**
```
✅ Cleanup completed successfully!

DELETED LOCAL BRANCHES (5):
  ✓ feat/add-yaml-parser
  ✓ fix/null-handling
  ✓ docs/update-readme
  ✓ hotfix/quick-fix
  ✓ feat/abandoned

DELETED REMOTE BRANCHES (3):
  ✓ origin/feat/add-yaml-parser
  ✓ origin/fix/null-handling
  ✓ origin/old-feature

KEPT BRANCHES (2):
  → feat/current-work (active)
  → experiment/new-approach (user kept)

FAILED (0):
  (none)

Repository is now clean! 🎉

Disk space saved: ~2.4 MB
Branches remaining: 4 (2 active + 2 protected)
```

### Phase 6: Optional Actions

#### 6.1 Update Main Branch

**Offer:** "Update main branch with latest changes? (yes/no)"

**If yes:**
```bash
git checkout main
git pull --ff-only origin main
```

#### 6.2 Garbage Collection

**Offer:** "Run git garbage collection to free disk space? (yes/no)"

**If yes:**
```bash
git gc --prune=now
```

**Shows space saved**

## Advanced Features

### Dry Run Mode

```bash
smart-cleanup --dry-run
```

- Shows what would be deleted
- No actual deletions
- Safe preview

### Aggressive Mode

```bash
smart-cleanup --aggressive
```

- Deletes merged branches without confirmation
- Deletes stale branches >30 days automatically
- Faster but less safe

### Age-Based Cleanup

```bash
smart-cleanup --age=60
```

- Only consider branches older than 60 days as stale
- Configurable threshold

### Protected Branch Patterns

**In config:**
```json
{
  "protectedPatterns": [
    "main",
    "master",
    "develop",
    "release/*",
    "hotfix/*"
  ]
}
```

## Rules

### Never:
- Delete protected branches (main/master/develop)
- Delete current branch
- Delete branches with unmerged commits (without confirmation)
- Delete active branches (<7 days old)
- Force operations without warning

### Always:
- Fetch and prune before analysis
- Categorize branches clearly
- Show what will be deleted
- Confirm destructive operations
- Provide detailed summary
- Track success/failures

### Prefer:
- Safe delete (`-d`) over force delete (`-D`)
- Interactive confirmation for stale branches
- Cleaning both local and remote together
- Dry run for first-time users

## Safety Features

### 1. Protected Branch Detection
- Never delete main/master/develop
- Respect config patterns
- Warn about production branches

### 2. Merge Verification
```bash
git branch --merged main
```
- Only delete truly merged branches
- Check against default branch

### 3. Confirmation Layers
- Show deletion plan
- Require user choice
- Extra confirmation for stale branches
- Warning for force deletes

### 4. Undo Information
```bash
# To restore a deleted branch
git checkout -b <branch-name> <commit-sha>
```
- Show commit SHAs in output
- Can restore if needed

## Configuration

Check for `.smartcleanup.json` (optional):
```json
{
  "autoCleanMerged": false,
  "staleThreshold": 30,
  "protectedPatterns": ["main", "master", "develop", "release/*"],
  "includeRemote": true,
  "confirmStale": true,
  "autoGc": false,
  "dryRun": false
}
```

## Integration with Other Skills

### Called from smart-merge

```bash
# After successful merge
smart-merge
✅ Merge completed!
...
Run cleanup now? (yes/no)
→ Calls smart-cleanup
```

### Periodic Maintenance

```bash
# Weekly/monthly cleanup routine
smart-cleanup --age=60
```

### Before Major Work

```bash
# Clean up before starting new project phase
smart-cleanup
smart-branch  # Start fresh
```

## Use Cases

### 1. After Merge
```
smart-merge          # Merge feature
smart-cleanup        # Clean up merged branch
```

### 2. Periodic Cleanup
```
smart-cleanup        # Monthly maintenance
→ Delete 10 merged branches
→ Review 3 stale branches
```

### 3. Repository Hygiene
```
smart-cleanup --dry-run  # Preview
smart-cleanup            # Execute
```

### 4. Onboarding Cleanup
```
# New team member cleans up old branches
smart-cleanup --age=90
```

### 5. Pre-Release Cleanup
```
# Before major release
smart-cleanup --aggressive
```

## Edge Cases

- **Branch with unpushed commits**: Warn, show commit count, require confirmation
- **Branch with open PR**: Warn, show PR link, suggest keeping
- **Current branch in list**: Skip automatically with note
- **No branches to delete**: Success message, no action needed
- **Remote already deleted**: Handle gracefully, clean up local tracking
- **Network failure**: Clean local only, warn about remote
- **Branch name conflicts**: Handle special characters, spaces
- **Detached HEAD**: Error, require checkout first
- **Protected branch in list**: Skip with warning
- **Force delete required**: Extra confirmation prompt

## Output Format

### Summary View
```
🧹 Cleanup Summary
  Merged: 5 branches (3 local, 2 remote)
  Stale: 2 branches
  Gone: 3 branches
  Protected: 2 branches
  Active: 4 branches
```

### Detailed View
```
Branch: feat/add-yaml-parser
  Status: ✅ Merged
  Age: 3 days
  Last commit: feat(parser): add YAML support
  Merged into: main
  Can delete: Yes
```

## Invocation

User can trigger with:
- "smart cleanup"
- "clean up branches"
- "delete merged branches"
- "cleanup repository"
- "remove old branches"

**With options:**
- "smart cleanup --dry-run"
- "smart cleanup --aggressive"
- "smart cleanup --age=60"
