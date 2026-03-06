---
name: smart-status
description: Show comprehensive repository overview with branch status, PRs, and quick navigation actions.
---

# Smart Status Skill

An intelligent workflow hub that provides a comprehensive overview of your repository state and enables quick navigation between branches. Perfect for starting your day or checking current status.

**Dependencies:** Requires `git`. Optional `gh` CLI (GitHub CLI) for viewing open pull requests.
- Install: https://cli.github.com/
- Authenticate: `gh auth login`
- Note: Skill works without `gh` but won't display PR information

## Workflow

Execute this workflow to check state and take quick actions:

### Phase 1: Current State Overview

#### 1.1 Branch Information

```bash
git branch --show-current
git status --porcelain
```

**Display:**
```
📍 CURRENT STATUS

Branch: feat/add-yaml-parser
Status: 3 files modified, 1 untracked
Tracking: origin/feat/add-yaml-parser (up to date)
Last commit: 2 hours ago
```

#### 1.2 Working Directory Status

**If clean:**
```
✅ Working directory clean
```

**If dirty:**
```
⚠️  Uncommitted changes:
  M  src/parser.ts
  M  tests/parser.test.ts
  ?? temp/notes.md

Suggestion: Run 'smart-save' to checkpoint or 'smart-commit' to organize
```

#### 1.3 Sync Status

```bash
git fetch origin
git rev-list --left-right --count @{upstream}...HEAD
```

**Show:**
- ↑ X commits ahead of remote
- ↓ Y commits behind remote
- ✅ Up to date
- ⚠️ Diverged (both ahead and behind)

### Phase 2: Repository Overview

#### 2.1 List All Branches

```bash
git branch -vv --sort=-committerdate
```

**Categorize and display:**

```
🌿 YOUR BRANCHES

✨ ACTIVE (Recent activity):
  → feat/add-yaml-parser (current)
      Last: 2 hours ago - "wip: implementing parser"
      Status: ↑2 ahead of origin
      
  → feat/add-smart-branch-skill
      Last: 1 day ago - "feat(skills): add smart-branch"
      Status: ✅ up to date
      
  → fix/memory-leak
      Last: 3 days ago - "fix: resolve memory leak in parser"
      Status: ↓1 behind origin

💤 STALE (No activity in 30+ days):
  → experiment/new-approach
      Last: 45 days ago - "wip: trying new approach"
      Status: No remote tracking

🔒 PROTECTED:
  → main
      Last: 1 day ago - "Merge initial-setup into main"
      Status: ✅ up to date
```

#### 2.2 Open Pull Requests

```bash
gh pr list
```

**Display:**
```
📋 OPEN PULL REQUESTS (2)

  #123 feat: add YAML parsing support
       Branch: feat/add-yaml-parser
       Status: ✅ All checks passed
       Reviews: 1 approved
       
  #120 fix: memory leak in parser  
       Branch: fix/memory-leak
       Status: ⏳ CI running
       Reviews: 0
```

#### 2.3 Recent Activity

```bash
git log --oneline -5 --all --decorate
```

**Show last 5 commits across all branches:**
```
📜 RECENT ACTIVITY

  aaec520 (feat/add-yaml-parser) wip: implementing parser
  35015d2 (feat/add-smart-branch-skill) feat(skills): add smart-branch
  1acc7f8 (main) Merge initial-setup into main
  7de07ef feat(skills): add smart-commit, smart-pull-request, smart-merge
  43fb9fc docs: add Claude skills documentation
```

### Phase 3: Quick Actions Menu

**Present interactive menu:**

```
⚡ QUICK ACTIONS

Branch Navigation:
  1. Switch to existing branch
  2. Create new branch (smart-branch)
  3. Continue on current branch

Work Management:
  4. Save progress (smart-save)
  5. Commit changes (smart-commit)
  6. Create PR (smart-pull-request)
  7. Merge branch (smart-merge)

Maintenance:
  8. Clean up branches (smart-cleanup)
  9. View detailed git status
  10. Refresh status

  0. Exit

Choose action (0-10):
```

### Phase 4: Branch Switching

**If user selects "1. Switch to existing branch":**

#### 4.1 Check Working Directory

**If uncommitted changes:**
```
⚠️  You have uncommitted changes.

Options:
  1. Save changes first (smart-save)
  2. Commit changes first (smart-commit)
  3. Stash changes and switch
  4. Discard changes and switch (dangerous!)
  5. Cancel

Choose option (1-5):
```

#### 4.2 Show Branch List

**Interactive branch selection:**
```
🌿 SELECT BRANCH TO SWITCH TO

ACTIVE BRANCHES:
  1. feat/add-smart-branch-skill (1 day ago)
  2. fix/memory-leak (3 days ago)
  3. main (1 day ago)

STALE BRANCHES:
  4. experiment/new-approach (45 days ago)

Enter number or branch name (or 'cancel'):
```

#### 4.3 Execute Switch

```bash
# If changes handled
git checkout <selected-branch>

# Update from remote if needed
git pull --ff-only
```

**After switch:**
```
✅ Switched to branch: feat/add-smart-branch-skill

Branch info:
  Last commit: 1 day ago
  Status: ✅ Up to date with origin
  Files: Clean working directory

Recent commits on this branch:
  35015d2 feat(skills): add smart-branch
  
Continue working or run 'smart-status' again for options.
```

#### 4.4 Smart Branch Loading

**If branch has session notes (.session-notes.md):**
```
📝 SESSION NOTES FOUND

Last session: 2024-01-19 15:30

Status: Implementing smart-branch skill, 80% complete

In Progress:
  - [ ] Add branch validation
  - [x] Create branch name generator
  - [x] Add tracking setup

Next Steps:
  - Complete validation logic
  - Test with edge cases
  - Update documentation

Load full notes? (yes/no)
```

### Phase 5: Suggested Next Action

**Based on current state, suggest action:**

**Scenario 1: Clean directory, current branch**
```
💡 SUGGESTED NEXT ACTION
Your working directory is clean. You can:
  → Continue working on feat/add-yaml-parser
  → Switch to another branch
  → Start new work with 'smart-branch'
```

**Scenario 2: Uncommitted changes**
```
💡 SUGGESTED NEXT ACTION
You have uncommitted changes. You should:
  → 'smart-save' to checkpoint your work
  → 'smart-commit' to organize into proper commits
```

**Scenario 3: Ready for PR**
```
💡 SUGGESTED NEXT ACTION
Your work looks ready! Consider:
  → 'smart-pull-request' to create a pull request
  → Push remaining commits: git push
```

**Scenario 4: Merged PR**
```
💡 SUGGESTED NEXT ACTION
PR #123 was merged! Next steps:
  → 'smart-merge' to merge locally
  → 'smart-cleanup' to clean up branches
  → 'smart-branch' to start new work
```

**Scenario 5: Behind remote**
```
💡 SUGGESTED NEXT ACTION
Your branch is behind remote. You should:
  → Pull latest changes: git pull
  → Or if you prefer: git pull --rebase
```

## Display Modes

### Compact Mode (Default)
```bash
smart-status
```
- Current branch and status
- Branch list (active only)
- Quick actions menu

### Full Mode
```bash
smart-status --full
```
- Everything in compact
- Stale branches
- Recent activity
- Detailed sync status
- All PRs

### Quick Mode
```bash
smart-status --quick
```
- Current branch only
- Sync status
- Uncommitted changes
- Next action suggestion
- No menu

## Smart Features

### 1. Context Awareness

**Start of Day:**
- Detect if it's a new day since last commit
- Show: "Welcome back! Here's what's in progress..."
- Suggest resuming yesterday's work

**After Merge:**
- Detect recently merged PRs
- Suggest cleanup
- Suggest starting new work

**Stale Work:**
- Detect branches inactive >7 days
- Suggest reviewing or cleaning up

### 2. Branch Health

**Indicators:**
- 🟢 Up to date, clean
- 🟡 Needs attention (behind, uncommitted)
- 🔴 Issues (conflicts, failed CI)
- 💤 Stale (old)

### 3. Quick Branch Info

```bash
smart-status <branch-name>
```

Shows detailed info about specific branch:
- Last commits
- Sync status
- PR status
- Age
- Can switch to it

## Configuration

Check for `.smartstatus.json` (optional):
```json
{
  "defaultMode": "compact",
  "showStale": true,
  "staleThreshold": 30,
  "maxBranchesDisplay": 10,
  "showPRs": true,
  "showSuggestions": true,
  "colorOutput": true,
  "quickActions": true
}
```

## Integration with Workflow

### Start of Day Routine

```bash
smart-status
→ Shows overview
→ Select branch to work on
→ Load session notes
→ Continue working
```

### Context Switching

```bash
# While working on feature A
smart-save  # Save current work

smart-status
→ Switch to branch B

# Work on B...

smart-status  
→ Switch back to A
→ Resume where left off
```

### End of Day

```bash
smart-save  # Save progress
smart-status --quick  # Quick check
→ See everything is backed up
→ Safe to close
```

## Rules

### Never:
- Switch branches with uncommitted changes (without handling them)
- Show confusing or overwhelming information
- Lose context when switching
- Force actions without confirmation

### Always:
- Show current state clearly
- Provide actionable suggestions
- Handle uncommitted changes safely
- Show sync status with remote
- Enable quick navigation

### Prefer:
- Compact view by default
- Recent/active branches first
- Clear categorization
- Actionable suggestions
- Interactive menus

## Edge Cases

- **Detached HEAD**: Show warning, offer to create branch
- **No branches**: Just main, suggest starting new work
- **No remote**: Skip remote sync info
- **Many branches**: Paginate, show most recent first
- **Merge in progress**: Show merge status, suggest resolution
- **Rebase in progress**: Show rebase status, offer continue/abort
- **No git repo**: Error, suggest git init
- **Diverged branch**: Show both ahead/behind, suggest rebase
- **Uncommitted changes on switch**: Handle with options

## Output Examples

### Clean State
```
📍 CURRENT STATUS
Branch: main
Status: ✅ Clean, up to date

💡 Ready to start new work!
Run 'smart-branch' to create a feature branch.
```

### Work in Progress
```
📍 CURRENT STATUS  
Branch: feat/add-yaml-parser
Status: ⚠️  3 files modified
Tracking: ↑2 ahead of origin

🌿 OTHER BRANCHES (2)
  feat/add-smart-branch-skill (1 day ago)
  main (1 day ago)

💡 Save your progress with 'smart-save'
```

### Multiple Active Branches
```
📍 CURRENT STATUS
Branch: main (protected)
Status: ✅ Clean, up to date

🌿 YOUR ACTIVE BRANCHES (3)
  1. feat/add-yaml-parser (↑2, 3 files modified)
  2. feat/add-smart-branch-skill (✅ up to date)
  3. fix/memory-leak (↓1 behind)

📋 OPEN PRS (2)
  #123 feat: add YAML support (✅ approved)
  #120 fix: memory leak (⏳ CI running)

⚡ Switch to branch? (enter number or 'no')
```

## Invocation

User can trigger with:
- "smart status"
- "status"
- "what's going on"
- "show branches"
- "switch branch"
- "where am I"

**With options:**
- "smart status --full"
- "smart status --quick"
- "smart status feat/my-branch"
