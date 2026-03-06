---
name: smart-pull-request
description: Run quality gates, auto-fix issues, and create a GitHub PR with generated description.
---

# Smart Pull Request Skill

An intelligent pull request workflow with comprehensive quality gates, auto-fixing, and CI monitoring. Ensures only high-quality PRs are created by enforcing checks and automatically fixing common issues.

**Dependencies:** Requires `git` and `gh` CLI (GitHub CLI) to be installed and authenticated.
- Install: https://cli.github.com/
- Authenticate: `gh auth login`

## Workflow

Execute this 5-phase workflow:

### Phase 1: Pre-flight Checks

Run comprehensive checks before attempting to push:

```bash
git status --porcelain
git branch --show-current
git log origin/$(git branch --show-current)..HEAD --oneline
```

**Verify:**
- Currently on a feature branch (not main/master)
- Branch has commits to push
- Working directory is clean (no uncommitted changes)

**If issues found:**
- Uncommitted changes: Suggest using smart-commit first
- On main/master: Error and stop
- No commits to push: Inform user nothing to do

### Phase 2: Quality Gates & Auto-Fixing

Run quality checks in this order, auto-fixing when possible:

#### 2.1 Linting & Formatting (Auto-fixable)

**For Python projects:**
```bash
ruff check --fix .
ruff format .
```

**For TypeScript/JavaScript:**
```bash
npm run lint -- --fix
npm run format
```

**If fixes applied:**
- Stage fixed files: `git add <fixed-files>`
- Create commit: `git commit -m "style: auto-fix linting and formatting errors"`
- Show summary of what was fixed

#### 2.2 Type Checking

**For Python:**
```bash
mypy . || pyright .
```

**For TypeScript:**
```bash
npm run type-check || tsc --noEmit
```

**If type errors found:**
- Display errors clearly
- Offer to attempt common fixes:
  - Add missing type hints
  - Fix obvious type mismatches
  - Add `# type: ignore` comments with justification (requires approval)
- If auto-fixed, commit with: `fix(types): resolve type checking errors`
- If not fixable, require user to fix before proceeding

#### 2.3 Tests

```bash
# Python
pytest --cov --cov-report=term-missing

# JavaScript/TypeScript
npm test -- --coverage
```

**Requirements:**
- All tests must pass (fail fast if any fail)
- Track coverage percentage
- Warn if coverage < 80%
- Fail if coverage < 70% (require override)

**If tests fail:**
- Show failure details
- Stop PR creation
- Suggest: "Fix tests before creating PR"

#### 2.4 Build Check

```bash
# Python
python -m build

# Node.js
npm run build
```

**If build fails:**
- Show build errors
- Stop PR creation
- No auto-fix available

### Phase 3: Push Changes

```bash
git push -u origin <branch-name>
```

**Handle push scenarios:**
- First push: Set upstream and push
- Subsequent push: Regular push
- Push rejected: Offer to pull with rebase or force push (with warning)
- Remote conflicts: Require user intervention

### Phase 4: Generate PR Description

#### 4.1 Read PR Template

Check for `.github/pull_request_template.md` and read it:

```bash
if (Test-Path .github/pull_request_template.md) {
    Get-Content .github/pull_request_template.md
}
```

#### 4.2 Extract Information

**From commits:**
```bash
git log origin/main..HEAD --pretty=format:"%s%n%b"
```

**Parse for:**
- Feature descriptions (from feat: commits)
- Bug fixes (from fix: commits)
- Breaking changes (from commit bodies with BREAKING CHANGE:)
- Performance improvements (from perf: commits)
- Refactoring notes (from refactor: commits)

#### 4.3 Auto-fill Template

Generate PR description following template structure:

**Title:** Use first commit message or summarize if multiple features

**Description sections:**
- **What**: Summarize changes from commit messages
- **Why**: Extract from commit bodies and BREAKING CHANGE notes
- **How**: List key implementation details
- **Testing**: Auto-check based on test results
  - ✅ Unit tests passing (X tests)
  - ✅ Coverage at Y%
  - List new test files added
- **Screenshots/Videos**: Leave empty with note if needed
- **Breaking Changes**: Auto-detect from commits with BREAKING CHANGE:
- **Performance Impact**: Extract from perf: commits
- **Auto-fixes Applied**: List fixes from Phase 2

**Checklist auto-completion:**
- ✅ Tests pass
- ✅ Coverage maintained/improved
- ✅ Types check
- ✅ Linting clean
- ✅ Build succeeds
- ⚠️ Breaking changes (if detected)

### Phase 5: Create PR

```bash
gh pr create --title "<title>" --body "<generated-description>" --web
```

**Options:**
- `--draft`: Create as draft PR
- `--assignee @me`: Auto-assign to self
- `--label`: Auto-add labels based on commit types
  - feat: → `enhancement`
  - fix: → `bug`
  - docs: → `documentation`
  - perf: → `performance`
  - Detected breaking change → `breaking-change`

**After creation:**
- Display PR URL
- Show PR number
- Offer CI monitoring: "Would you like me to monitor the CI checks?"

## Phase 6: CI Monitoring (Optional)

If user agrees to monitor:

```bash
gh pr checks <pr-number> --watch
```

**Monitor CI results:**
- Wait for checks to complete
- Report status in real-time
- If failures detected:
  - Fetch failure logs: `gh run view <run-id> --log-failed`
  - Analyze failure type
  - Attempt auto-fix if possible:
    - Linting failures: Re-run Phase 2.1
    - Test failures: Report, cannot auto-fix
    - Build failures: Report, cannot auto-fix
  - If auto-fixed:
    - Push fix
    - Continue monitoring
    - Update PR with note about fix

## Smart Auto-Fixing

### Auto-fixable Issues

1. **Linting/Formatting**
   - Unused imports
   - Line length
   - Indentation
   - Trailing whitespace
   - Quote style
   - Import ordering

2. **Common Type Hints**
   - Missing return types for simple functions
   - Obvious parameter types
   - Basic generic types

3. **Documentation**
   - Missing docstrings (add template)
   - Docstring format issues

### Non-auto-fixable Issues

Require user intervention:
- Logic errors
- Test failures
- Complex type issues
- Breaking API changes
- Security vulnerabilities

## Rules

### Never:
- Create PR with failing tests
- Push to main/master directly
- Auto-fix without showing what changed
- Force push without warning
- Add `# type: ignore` without user approval
- Proceed with uncommitted changes

### Always:
- Run quality gates in order
- Show summary of auto-fixes
- Fail fast on critical errors
- Track coverage and warn on drops
- Generate meaningful PR descriptions
- Extract information from commits
- Auto-check template items when possible

## Configuration

Check for project-specific config:

**.smartpullrequest.json** (optional):
```json
{
  "minCoverage": 80,
  "strictCoverage": 70,
  "autoAssign": true,
  "draft": false,
  "labels": {
    "feat": ["enhancement"],
    "fix": ["bug"],
    "docs": ["documentation"]
  },
  "skipChecks": [],
  "customChecks": []
}
```

## Output Format

### Quality Gates Summary
```
🔍 Running Pre-flight Checks...
✅ On feature branch: feat/add-yaml-support
✅ 3 commits ready to push
✅ Working directory clean

🔧 Running Quality Gates...

Linting & Formatting:
  ✅ Fixed 12 linting issues
  ✅ Formatted 5 files
  📝 Auto-committed: style: auto-fix linting and formatting errors

Type Checking:
  ✅ No type errors found

Tests:
  ✅ 45 tests passed
  ✅ Coverage: 87% (target: 80%)
  ✅ All tests passing

Build:
  ✅ Build successful

🚀 Pushing to origin/feat/add-yaml-support...
✅ Pushed successfully
```

### PR Creation Summary
```
📝 Generating PR Description...
  ✅ Read PR template
  ✅ Extracted info from 3 commits
  ✅ Auto-checked 5 checklist items
  ✅ Added auto-fix summary

🎯 Creating Pull Request...
  Title: feat: add YAML parsing support
  Labels: enhancement, documentation
  Assignee: @me

✅ PR created successfully!
  URL: https://github.com/user/repo/pull/123
  Number: #123

Auto-fixes applied:
  • Fixed 12 linting issues (ruff)
  • Formatted 5 files
  • Added 2 missing type hints

Would you like me to monitor the CI checks?
```

### CI Monitoring Output
```
👀 Monitoring CI checks for PR #123...

✅ Lint (2m 15s)
✅ Type Check (1m 45s)
⏳ Tests (running...)
⏳ Build (queued)

---

✅ Tests (3m 42s) - 45 passed
✅ Build (2m 10s)

🎉 All checks passed!
```

## Edge Cases

- **No PR template**: Generate basic description from commits
- **Multiple base branches**: Ask user which branch to target
- **Draft PR**: Add `--draft` flag if requested
- **Existing PR**: Update existing PR instead of creating new
- **Coverage drop**: Warn and require confirmation to proceed
- **Breaking changes**: Add warning label and highlight in description
- **Large PR**: Warn if >500 lines changed, suggest splitting
- **Merge conflicts**: Report and require user to resolve
- **CI not configured**: Skip monitoring phase
- **Private repo without gh auth**: Provide manual PR creation link

## Invocation

User can trigger with:
- "smart pull request"
- "smart pr"
- "create pr"
- "make a pull request"
- "pr this"
- "submit for review"
