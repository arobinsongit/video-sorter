---
name: smart-commit
description: Analyze staged/unstaged changes, auto-group them by concern, and auto-commit each bucket with conventional commit messages.
---

# Smart Commit Skill

**Dependencies:** Requires `git` to be installed.

An intelligent commit workflow that automatically organizes your changes into logical, well-structured commits following conventional commit standards.

## Workflow

When invoked, execute these steps:

### 1. Branch Safety Check

```bash
git branch --show-current
```

- If on `main` or `master`:
  - Ask user for feature description
  - Suggest branch name based on primary change type (e.g., `feat/add-yaml-support`)
  - Create and checkout the branch
  - Proceed with commits

### 2. Analyze Changed Files

```bash
git status --porcelain
```

Group files into logical batches by concern:

- **docs**: README, documentation files, .md files in /docs
- **automation**: .claude skills, GitHub prompts, workflow templates
- **config/chore**: CI/CD configs, tooling configs (.eslintrc, tsconfig.json), lockfiles
- **tests**: Test files (*test*, *spec*), test fixtures, test configs
- **code**: Library/implementation code (categorize as feat/fix/refactor/perf based on content)
- **misc**: Files that don't fit other categories (ask user before including)

### 3. Create Atomic Commits

For each batch, in this order:
1. **docs** - Documentation updates
2. **tests** - Test changes
3. **code** - Implementation changes
4. **automation** - Automation and workflow updates
5. **config/chore** - Configuration and tooling updates
6. **misc** - Other changes (if user approved)

For each batch:
1. Stage files explicitly: `git add <file1> <file2> ...`
2. Generate conventional commit message
3. Execute: `git commit -m "type(scope): description"`

### 4. Conventional Commit Format

Format: `type(scope): description`

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `test`: Adding or updating tests
- `refactor`: Code change that neither fixes a bug nor adds a feature
- `chore`: Changes to build process or auxiliary tools
- `ci`: Changes to CI configuration files and scripts
- `perf`: Performance improvement
- `style`: Code style changes (formatting, missing semi-colons, etc.)

**Rules:**
- Use imperative mood ("add" not "added" or "adds")
- Maximum 72 characters
- No ending period
- Scope is optional but recommended (e.g., `feat(parser): add yaml support`)
- Lowercase type and description

**Examples:**
```
feat(api): add user authentication endpoint
fix(parser): handle empty yaml documents
docs: update installation instructions
test(validator): add edge case for null values
refactor(utils): simplify file path handling
chore(deps): update dependencies
ci: add automated release workflow
```

### 5. Output Summary

After completing commits:
1. List all commits created with file counts
2. Explain batching rationale
3. Show total files committed
4. Ask: "Would you like me to push these commits?"

**Example Output:**
```
✓ Created 3 commits:

1. docs: update README with new features (2 files)
   - README.md
   - docs/guide.md

2. feat(parser): add YAML support (4 files)
   - src/parser.ts
   - src/yaml-handler.ts
   - src/types.ts
   - src/index.ts

3. test(parser): add YAML parsing tests (2 files)
   - tests/parser.test.ts
   - tests/fixtures/sample.yaml

Total: 8 files committed across 3 logical commits
Batching rationale: Separated docs, implementation, and tests for atomic review

Would you like me to push these commits?
```

## Important Rules

### Never:
- Use `git add .` or `git add -A`
- Commit to main/master without creating a feature branch
- Mix unrelated changes in one commit
- Include misc files without user approval
- Use past tense in commit messages

### Always:
- Stage files explicitly
- Group related changes together
- Use conventional commit format
- Keep commits atomic and self-contained
- Show clear output with rationale

## Edge Cases

- **No changes**: Inform user no files to commit
- **Conflicts**: Report and ask user to resolve
- **Untracked files**: Include in analysis, ask about misc files
- **Binary files**: Commit with appropriate type, mention in output
- **Large files**: Warn if >5MB, confirm before staging
- **Partial staging**: If user wants to exclude files, respect that

## Invocation

User can trigger with:
- "smart commit"
- "commit these changes"
- "organize my commits"
- "commit intelligently"
