---
name: smart-save
description: Create WIP commits and back them up to remote for quick checkpoints during development.
---

**Dependencies:** Requires `git` to be installed.

## Workflow

Execute this workflow when you need to save progress:

### Phase 1: Pre-save Checks

#### 1.1 Check Current State

```bash
git status --porcelain
git branch --show-current
```

**Verify:**
- Not on main/master (error if so)
- Not in detached HEAD state
- Changes exist to save

**If no changes:**
- Message: "✅ No changes to save. Working directory is clean."
- Exit gracefully

**If on main/master:**
- Error: "❌ Cannot create WIP commits on protected branch. Create a feature branch first with 'smart-branch'."
- Exit

### Phase 2: Gather Context

#### 2.1 Analyze Changes

```bash
git status --short
git diff --stat
```

**Show summary:**
```
📝 Changes to save:
  Modified: 3 files
  Added: 1 file
  Deleted: 0 files

Files:
  M  src/parser.ts (+45, -12)
  M  tests/parser.test.ts (+23, -5)
  A  src/yaml-handler.ts (+89)
  M  README.md (+2, -1)
```

#### 2.2 Ask for Description

**Prompt:** "What are you saving? (optional, press Enter to skip)"

**User can provide:**
- Brief description of current work state
- What's working, what's not
- What to continue next
- Or just press Enter for auto-generated message

**If no description provided:**
- Auto-generate from file names: "wip: changes in parser, tests, yaml-handler"

### Phase 3: Create WIP Commit

#### 3.1 Stage All Changes

```bash
git add -A
```

**Note:** This is an exception to the "never use git add ." rule because:
- WIP commits are temporary/informal
- User is intentionally saving current state
- Will be cleaned up before final PR

#### 3.2 Generate Commit Message

**Format:** `wip: <description>`

**If user provided description:**
```
wip: <user description>
```

**If auto-generated:**
```
wip: changes in <file1>, <file2>, <file3>
```

**If too many files (>5):**
```
wip: changes in <N> files
```

**Add timestamp and session notes:**
```
wip: <description>

Session: <timestamp>
Files changed: <N>
<optional user notes>
```

#### 3.3 Create Commit

```bash
git commit -m "<generated message>"
```

### Phase 4: Backup to Remote

#### 4.1 Check Remote Tracking

```bash
git rev-parse --abbrev-ref --symbolic-full-name @{upstream}
```

**If tracking exists:**
- Push directly

**If no tracking:**
- Ask: "This branch isn't tracked remotely yet. Set up tracking and push? (yes/no)"
- If yes: `git push -u origin <branch>`
- If no: Skip push (local save only)

#### 4.2 Push Changes

```bash
git push
```

**If push rejected (diverged):**
- Warn: "⚠️  Remote has changes you don't have locally"
- Options:
  1. Pull and merge, then push
  2. Force push (with warning)
  3. Skip push (local only)

### Phase 5: Optional Session Notes

**Prompt:** "Add session notes? (yes/no)"

**If yes:**

Create or update `.session-notes.md`:
```markdown
# Session Notes: <branch-name>

## Last Updated: <timestamp>

## Current Status
<User's description>

## In Progress
- [ ] 

## Blockers
- 

## Next Steps
- 

---

## Session History

### <timestamp>
<description>
Files: <files>
```

**Commit session notes:**
```bash
git add .session-notes.md
git commit -m "docs: update session notes"
git push
```

### Phase 6: Summary

**Display:**
```
✅ Work saved successfully!

Branch: feat/add-yaml-parser
Commit: wip: implementing YAML parsing logic
Files saved: 4
Backup: ✅ Pushed to origin

Session notes: ✅ Updated

You can safely:
- Close your session
- Switch to another branch
- Shut down your computer

To resume:
- Continue working and commit normally
- Or run 'smart-save' again for another checkpoint
```

## Advanced Features

### Auto-Save Mode

**For frequent saves during active development:**

```bash
# Every 15 minutes (requires configuration)
smart-save --auto --interval=15m --silent
```

**Features:**
- Silent operation (no prompts)
- Auto-generated messages with timestamps
- Only saves if changes exist
- Always pushes for backup

### Session Recovery

**If returning to work after smart-save:**

```bash
smart-save --resume
```

**Shows:**
- Last session notes
- What was in progress
- Uncommitted changes (if any)
- Suggests next steps

### Clean Up WIP Commits

**Before creating PR:**

The WIP commits should be cleaned up using `smart-commit` which will:
1. Squash all WIP commits into logical commits
2. Create proper conventional commits
3. Remove WIP history

**Or use interactive rebase:**
```bash
git rebase -i origin/main
```

## Rules

### Never:
- Save WIP commits to main/master
- Force push without warning
- Lose user's work
- Create WIP commits in final PR (remind to clean up)

### Always:
- Use "wip:" prefix
- Show what's being saved
- Confirm push succeeded
- Provide clear summary
- Make it safe to interrupt

### Prefer:
- Pushing for remote backup
- Descriptive WIP messages
- Session notes for complex work
- Timestamped commits
- Clear state summaries

## Configuration

Check for `.smartsave.json` (optional):
```json
{
  "autoPush": true,
  "sessionNotes": false,
  "autoSave": {
    "enabled": false,
    "interval": "15m"
  },
  "messageFormat": "wip: {description}",
  "includeTimestamp": true
}
```

## Use Cases

### 1. End of Day
```
User: "smart-save - end of day, YAML parser 80% complete"
→ Commits and pushes
→ Safe to shut down
```

### 2. Switching Context
```
User: "smart-save - pausing to work on hotfix"
→ Saves current work
→ Can switch branches safely
→ Can return later with 'git checkout feat/...'
```

### 3. Quick Backup
```
User: "smart-save"
→ Quick commit + push
→ Continue working
→ No interruption
```

### 4. Experimental Work
```
User: "smart-save - trying new approach, may revert"
→ Creates checkpoint
→ Can experiment freely
→ Can revert if needed
```

### 5. Before Risky Operation
```
User: "smart-save - before rebasing"
→ Safety checkpoint
→ Can recover if rebase goes wrong
```

## Integration with Other Skills

### With smart-branch
```bash
smart-branch     # Start work
# ... code ...
smart-save       # Quick save
# ... more code ...
smart-save       # Another save
# ... done coding ...
smart-commit     # Clean up WIP commits
smart-pull-request  # Create PR
```

### With smart-commit
```bash
# During development
smart-save       # Multiple WIP commits
smart-save
smart-save

# When ready for PR
smart-commit     # Reorganizes into logical commits
                 # Removes WIP history
```

### WIP Commit Cleanup

**When running smart-commit, it should:**
1. Detect WIP commits
2. Offer to squash them
3. Create proper conventional commits
4. Remove WIP markers

**Update smart-commit to handle this**

## Edge Cases

- **No remote configured**: Offer to add remote or save locally only
- **Large files**: Warn if adding >5MB files, suggest .gitignore
- **Binary files**: Show list, confirm before adding
- **Merge in progress**: Error, cannot save during merge
- **Rebase in progress**: Error, cannot save during rebase
- **Stash exists**: Warn about existing stash
- **Untracked files**: Include in WIP commit (all state saved)
- **Push failure**: Offer retry, force push, or local-only

## Difference from Stash

**git stash:**
- Temporary, unnamed storage
- Not backed up remotely
- Easy to forget about
- No history/context

**smart-save:**
- Proper commit in git history
- Backed up to remote (GitHub)
- Visible in git log
- Can include notes and context
- Part of normal workflow
- Can be cleaned up later

## Invocation

User can trigger with:
- "smart save"
- "save my work"
- "wip commit"
- "backup my changes"
- "pause work"
- "end of day save"

**With description:**
- "smart save - YAML parser mostly working"
- "save my work - about to try risky refactor"
