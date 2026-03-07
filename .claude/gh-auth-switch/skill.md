---
name: gh-auth-switch
description: Add GitHub account auto-switching for git push to a project's CLAUDE.md. Handles save/restore of the previous active account.
---

# gh-auth-switch

Adds a `## Git Push` section to the current project's CLAUDE.md with `gh auth switch` before push and restore after.

## Usage

`/gh-auth-switch <username>` — hardcoded user
`/gh-auth-switch ENV:VAR_NAME` — resolve from env var at push time
`/gh-auth-switch` — auto-detect from `gh auth status`, ask user to pick

## Steps

1. Resolve target user from args, or run `gh auth status` and ask which account
2. Read CLAUDE.md in working directory (create with standard header if missing)
3. If `## Git Push` exists, replace it; otherwise append at end
4. Insert block below (replace `<USER>` with resolved value):

```markdown
## Git Push

**GitHub push user:** `<USER>`

\```bash
RAW_USER="<USER>"
if [[ "$RAW_USER" == ENV:* ]]; then RAW_USER="${!RAW_USER#ENV:}"; fi
PREV_USER=$(gh auth status 2>&1 | grep "Active account: true" -B3 | head -1 | awk '{print $NF}')
if [ "$PREV_USER" != "$RAW_USER" ]; then gh auth switch --user "$RAW_USER"; fi
git push origin <branch>
if [ "$PREV_USER" != "$RAW_USER" ] && [ -n "$PREV_USER" ]; then gh auth switch --user "$PREV_USER" 2>/dev/null; fi
\```
```

5. Read back CLAUDE.md and confirm the section was added
