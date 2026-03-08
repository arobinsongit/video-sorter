# Media Sorter

A portable media sorting tool for videos and photos. Browse your files, annotate them with configurable metadata (subjects, tags, quality, or any custom group), and the app renames/moves/copies files encoding that metadata into the filename.

```
clip.mp4  →  clip__S12__double_slide__great.mp4
```

Single binary, cross-platform, opens in your browser. No install, no account, no cloud required.

**Supported formats:** `.mp4` `.mov` `.avi` `.mkv` `.webm` `.jpg` `.jpeg` `.png` `.gif` `.webp` `.bmp` `.tiff` `.heic` `.heif`

---

## Quick Start (Download Only)

If you just want to run Media Sorter without building from source:

1. Go to the [Releases](https://github.com/arobinsongit/media-sorter/releases) page
2. Download the binary for your platform:
   - **Windows:** `media-sorter-windows.exe`
   - **Mac (Apple Silicon):** `media-sorter-mac-arm`
   - **Mac (Intel):** `media-sorter-mac-intel`
   - **Linux:** `media-sorter-linux`
3. Run it — a browser tab opens automatically

**Mac/Linux users:** You may need to make it executable first:
```bash
chmod +x media-sorter-mac-arm
./media-sorter-mac-arm
```

**Mac users:** If macOS blocks the app ("unidentified developer"), right-click → Open, or run:
```bash
xattr -d com.apple.quarantine media-sorter-mac-arm
```

---

## Building from Source

### Prerequisites

You need **VS Code** (code editor), **Node.js** (for the frontend build), **Go** (for the backend/binary), **Git**, the **GitHub CLI** (for PRs, issues, and releases), **Claude Code** (AI development assistant), **RTK** (token optimizer for Claude Code), and **Claude Code Monitor** (usage tracking).

#### GitHub Account

If you don't already have a GitHub account, create one first — you'll need it to clone the repo and contribute:

1. Go to https://github.com/signup
2. Follow the prompts to create your free account (email, password, username)
3. Verify your email address when GitHub sends you a confirmation link
4. **Recommended:** Enable two-factor authentication (2FA) under **Settings → Password and authentication**. GitHub requires 2FA for all contributors and will eventually require it for all accounts. Set it up now to avoid being locked out later.

#### Windows

1. **Install Node.js** (includes npm):
   - Download the LTS installer from https://nodejs.org/
   - Run the installer, accept all defaults
   - Restart your terminal after install

2. **Install Go:**
   - Download the installer from https://go.dev/dl/
   - Run the `.msi` installer, accept all defaults
   - Restart your terminal after install

3. **Install Git** (if you don't have it):
   - Download from https://git-scm.com/download/win
   - Run the installer — the defaults are fine, but when asked about the default editor, pick whatever you're comfortable with
   - **Recommended:** Choose "Git from the command line and also from 3rd-party software" when prompted

4. **Install GitHub CLI** (for creating PRs, issues, and releases from the terminal):
   - Download the latest `.msi` installer from https://cli.github.com/
   - Run the installer, accept defaults
   - After install, authenticate:
     ```
     gh auth login
     ```
     Follow the prompts — choose **GitHub.com**, **HTTPS**, and **Login with a web browser**.

5. **Install VS Code** (code editor):
   - Download from https://code.visualstudio.com/
   - Run the installer, accept defaults
   - **Recommended:** Check "Add to PATH" during install so you can open files from the terminal with `code filename`

6. **Install Claude Code** (AI development assistant):
   - Open a terminal and run:
     ```
     npm install -g @anthropic-ai/claude-code
     ```
   - Sign up for a Claude plan at https://claude.ai/ — the $20/month Pro plan is a good starting point
   - Run `claude` from the project directory to start an interactive session

7. **Install RTK** (token optimizer — reduces Claude Code's token usage by up to 89%):
   - Open Git Bash and run:
     ```
     curl -fsSL https://raw.githubusercontent.com/rtk-ai/rtk/refs/heads/master/install.sh | sh
     rtk init --global
     ```
   - Learn more at https://www.rtk-ai.app/

8. **Install Claude Code Monitor** (tracks token usage, costs, and predictions):
   - Open a terminal and run:
     ```
     pip install claude-monitor
     ```
   - Run `ccm` in a separate terminal while using Claude Code to monitor usage
   - Learn more at https://github.com/Maciek-roboblog/Claude-Code-Usage-Monitor

9. **Verify everything is installed** — open a new terminal (Command Prompt or PowerShell) and run:
   ```
   node --version
   go version
   git --version
   gh --version
   code --version
   claude --version
   rtk --version
   ccm --version
   ```
   You should see version numbers for all of these. If any command isn't found, restart your terminal.

#### macOS

1. **Install Homebrew** (if you don't have it):
   ```bash
   /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
   ```
   Follow any instructions it prints about adding Homebrew to your PATH.

2. **Install Node.js, Go, Git, and GitHub CLI:**
   ```bash
   brew install node go git gh
   ```

3. **Authenticate GitHub CLI:**
   ```bash
   gh auth login
   ```
   Follow the prompts — choose **GitHub.com**, **HTTPS**, and **Login with a web browser**.

4. **Install VS Code:**
   ```bash
   brew install --cask visual-studio-code
   ```

5. **Install Claude Code** (AI development assistant):
   ```bash
   npm install -g @anthropic-ai/claude-code
   ```
   Sign up for a Claude plan at https://claude.ai/ — the $20/month Pro plan is a good starting point.

6. **Install RTK** (token optimizer — reduces Claude Code's token usage by up to 89%):
   ```bash
   curl -fsSL https://raw.githubusercontent.com/rtk-ai/rtk/refs/heads/master/install.sh | sh
   rtk init --global
   ```
   Learn more at https://www.rtk-ai.app/

7. **Install Claude Code Monitor** (tracks token usage, costs, and predictions):
   ```bash
   pip install claude-monitor
   ```
   Run `ccm` in a separate terminal while using Claude Code. Learn more at https://github.com/Maciek-roboblog/Claude-Code-Usage-Monitor

8. **Verify:**
   ```bash
   node --version
   go version
   git --version
   gh --version
   code --version
   claude --version
   rtk --version
   ccm --version
   ```

#### Linux (Ubuntu/Debian)

```bash
# Node.js (via NodeSource for a current version)
curl -fsSL https://deb.nodesource.com/setup_22.x | sudo -E bash -
sudo apt-get install -y nodejs

# Go
sudo snap install go --classic
# Or download manually from https://go.dev/dl/

# Git
sudo apt-get install -y git

# GitHub CLI
# See https://github.com/cli/cli/blob/trunk/docs/install_linux.md for other distros
(type -p wget >/dev/null || (sudo apt update && sudo apt-get install wget -y)) \
  && sudo mkdir -p -m 755 /etc/apt/keyrings \
  && out=$(mktemp) && wget -nv -O$out https://cli.github.com/packages/githubcli-archive-keyring.gpg \
  && cat $out | sudo tee /etc/apt/keyrings/githubcli-archive-keyring.gpg > /dev/null \
  && sudo chmod go+r /etc/apt/keyrings/githubcli-archive-keyring.gpg \
  && echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | sudo tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
  && sudo apt update \
  && sudo apt install gh -y

# Authenticate
gh auth login

# VS Code
sudo snap install code --classic
# Or download from https://code.visualstudio.com/

# Claude Code (AI development assistant)
npm install -g @anthropic-ai/claude-code
# Sign up at https://claude.ai/ ($20/month Pro plan is a good starting point)

# RTK (token optimizer for Claude Code — up to 89% savings)
curl -fsSL https://raw.githubusercontent.com/rtk-ai/rtk/refs/heads/master/install.sh | sh
rtk init --global
# Learn more at https://www.rtk-ai.app/

# Claude Code Monitor (tracks token usage, costs, and predictions)
pip install claude-monitor
# Run `ccm` in a separate terminal while using Claude Code
# Learn more at https://github.com/Maciek-roboblog/Claude-Code-Usage-Monitor

# Verify
node --version
go version
git --version
gh --version
code --version
claude --version
rtk --version
ccm --version
```

### Clone and Build

```bash
git clone https://github.com/arobinsongit/media-sorter.git
cd media-sorter
```

#### First-time setup (install frontend dependencies):

```bash
npm install
```

#### Build and run:

**macOS / Linux:**
```bash
npm run build && go build -o media-sorter ./cmd/media-sorter && ./media-sorter
```

**Windows (Git Bash or similar):**
```bash
npm run build && go build -o media-sorter.exe ./cmd/media-sorter && ./media-sorter.exe
```

**Windows (Command Prompt):**
```
npm run build && go build -o media-sorter.exe .\cmd\media-sorter && media-sorter.exe
```

The app will print a URL and open your browser automatically.

### Rebuild after making changes

If you already have a running instance, kill it first then rebuild:

**macOS / Linux:**
```bash
pkill -f media-sorter; npm run build && go build -o media-sorter ./cmd/media-sorter && ./media-sorter
```

**Windows (Git Bash):**
```bash
taskkill //F //IM media-sorter.exe 2>/dev/null; npm run build && go build -o media-sorter.exe ./cmd/media-sorter && ./media-sorter.exe
```

> **Tip:** If you only changed Go code (not frontend), you can skip `npm run build` and just run the `go build` + run step.

### Release builds (all platforms)

To build optimized binaries for all four platforms at once:

**macOS / Linux:**
```bash
bash build.sh
```

**Windows:**
```
build.bat
```

Output goes to the `dist/` folder.

---

## How It Works

1. **Launch** the app — it starts a local web server and opens your browser
2. **Browse** to a folder containing your media files
3. **Select** a file to view/play it
4. **Annotate** using the metadata groups (click buttons or use keyboard shortcuts)
5. **Apply** — the file is renamed with your annotations encoded in the filename

### Filename Format

The output filename is controlled by a template. The default is:

```
{basename}__{S}__{tags}__{quality}.{ext}
```

Which produces filenames like:

```
clip__S5__diving_catch__great.mp4
photo__S12_S7__celebration__good.jpg
```

- `__` (double underscore) separates sections
- Each group's values are joined by its configured separator
- You can customize the template and all groups in Settings

### Configuration

Each media folder gets its own `media-sorter-config.json` file with:

- **Output format** — the filename template
- **Output mode** — `rename` (in place), `move` (to a folder), or `copy` (to a folder)
- **Metadata groups** — fully customizable: name, type, options, separators, prefixes

A file is considered "tagged" if its name contains `__`.

### Data Storage

| What | Where |
|------|-------|
| Project config | `media-sorter-config.json` in each media folder |
| Session state | `~/.media-sorter-session.json` (last folder, file, MRU order) |
| User settings | `~/.media-sorter-settings.json` (theme, keybindings) |
| Cloud credentials | `~/.media-sorter/` (OAuth tokens, API keys) |

Nothing is sent to the internet. Everything stays on your machine.

---

## Cloud Storage (Optional)

Media Sorter can browse and sort files stored in cloud services:

- **Google Drive**
- **Amazon S3**
- **Dropbox**
- **OneDrive**

Connect via the cloud icon in the UI. Each provider needs its own API credentials — see the in-app setup flow for details.

---

## Project Structure

```
media-sorter/
├── cmd/media-sorter/         # Go entry point + embedded static files
│   ├── main.go
│   └── static/               # Built frontend (committed to git)
├── internal/
│   ├── server/               # HTTP handlers, config, cloud providers
│   └── storage/              # Storage abstraction (local + cloud)
│       ├── gdrive/
│       ├── s3/
│       ├── dropbox/
│       └── onedrive/
├── frontend/
│   ├── index.html            # HTML shell
│   └── src/                  # ES module sources (bundled by esbuild)
├── build.sh                  # Cross-platform release build (Mac/Linux)
├── build.bat                 # Cross-platform release build (Windows)
├── build.mjs                 # Frontend build script (esbuild)
└── package.json
```

**Architecture:** Go backend with `embed.FS` serves a bundled frontend. The frontend is built with esbuild (vanilla JS, Tailwind CSS via CDN) and embedded into the Go binary at compile time. The result is a single executable with zero runtime dependencies.

---

## Running Tests

```bash
go test ./...
```

With verbose output:

```bash
go test -v ./...
```

With static analysis:

```bash
go vet ./...
```

---

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `←` Left Arrow | Previous file |
| `→` Right Arrow | Next file |
| `Enter` | Apply annotation (rename/move/copy) |
| `Escape` | Clear current selections |

Keyboard shortcuts are disabled when typing in the folder path input or editing settings.

---

## Troubleshooting

**"Port already in use" or app won't start:**
A previous instance may still be running. Kill it first:
- Windows: `taskkill /F /IM media-sorter.exe`
- Mac/Linux: `pkill -f media-sorter`

**Browser doesn't open automatically:**
The terminal shows the URL (e.g., `http://127.0.0.1:54321`). Copy and paste it into your browser manually.

**Files won't rename on Windows:**
If a video is playing in Media Sorter, Windows may lock the file. The app handles this by unloading the media player before renaming, but if it still fails, try pausing the video first.

**`go build` fails with "cannot find module":**
Make sure you ran `npm install` at least once (it creates `node_modules/`). Also verify you're in the project root directory where `go.mod` lives.

**`npm run build` fails:**
Run `npm install` first. If it still fails, delete `node_modules/` and try again:
```bash
rm -rf node_modules
npm install
npm run build
```

**Changes to frontend code don't show up:**
You need to run `npm run build` after editing any file in `frontend/`. The Go binary serves the built files from `cmd/media-sorter/static/`, not the source files directly.

---

## Contributing

### Git Workflow for Beginners

If you're new to Git, here's the basic workflow:

```bash
# 1. Make sure you're on main and up to date
git checkout main
git pull

# 2. Create a branch for your work
git checkout -b feat/my-feature

# 3. Make your changes, then check what changed
git status
git diff

# 4. Stage and commit
git add file1.go file2.js
git commit -m "feat: add my new feature"

# 5. Push your branch
git push -u origin feat/my-feature

# 6. Create a pull request
gh pr create --title "Add my new feature" --body "Description of what this does"

# 7. After PR is reviewed and merged, clean up
git checkout main
git pull
git branch -d feat/my-feature
```

**Commit message format:** Start with a type — `feat:` for new features, `fix:` for bug fixes, `docs:` for documentation, `test:` for tests, `refactor:` for code cleanup.

### Development Workflow

1. Fork and clone the repo
2. Create a feature branch: `git checkout -b feat/my-feature`
3. Make your changes
4. Run tests: `go test ./...`
5. Build and test manually: `npm run build && go build -o media-sorter ./cmd/media-sorter`
6. Push and open a PR against `main`

### Best Practices

- **Run `go vet ./...` before committing.** It catches common mistakes that compile fine but are almost certainly bugs.
- **Run tests before pushing.** CI runs `go test -race ./...` and will catch what you miss.
- **Frontend changes require `npm run build`** before the Go binary will pick them up. The built files in `cmd/media-sorter/static/` are committed to git so that `go build` works without Node.js installed.
- **One concern per commit.** Keep bug fixes, features, and refactors in separate commits.
- **The `main` branch is the release branch.** Every push to `main` triggers CI that builds all platforms and creates a GitHub release.

### Google Drive Credentials (Optional)

For Google Drive support in local builds, create a `.env.local` file in the project root:

```bash
GDRIVE_CLIENT_ID=your-client-id-here
GDRIVE_CLIENT_SECRET=your-client-secret-here
```

Then source it before building:

```bash
source .env.local
go build -ldflags "-X media-sorter/internal/storage/gdrive.embeddedClientID=$GDRIVE_CLIENT_ID -X media-sorter/internal/storage/gdrive.embeddedClientSecret=$GDRIVE_CLIENT_SECRET" -o media-sorter ./cmd/media-sorter
```

This is only needed if you want to test Google Drive integration. The app works fine without it.

---

## Using Claude Code for Development

This project uses [Claude Code](https://docs.anthropic.com/en/docs/claude-code) as an AI-powered development assistant. Claude Code can write code, run tests, create commits, manage issues and PRs, and more — all from the command line.

### Install Claude Code

Claude Code requires **Node.js 18+** (which you already have from the build prerequisites).

```bash
npm install -g @anthropic-ai/claude-code
```

### Sign Up for Claude

1. Go to https://claude.ai/ and create an account
2. Subscribe to the **Pro plan** ($20/month) — this is a good starting point to see how it works for you

### Run Claude Code

From the project root:

```bash
claude
```

This opens an interactive session where you can ask Claude to:
- Explain how any part of the codebase works
- Write new features or fix bugs
- Run tests and fix failures
- Create branches, commits, and PRs
- Create and manage GitHub issues

**Example prompts:**
```
> explain how the config system works
> add a new metadata group type called "rating" with 1-5 stars
> run the tests and fix any failures
> create a branch, commit my changes, and open a PR
```

### Tips

- **Start from the project root.** Claude Code reads `CLAUDE.md` for project context automatically.
- **Be specific.** "Fix the bug where rename fails on Windows" works better than "fix bugs".
- **Let it run tests.** Claude Code can run `go test ./...` and iterate on failures.
- **Review before committing.** Claude will ask for confirmation before git operations. Read the diffs.

### Smart Skills (Slash Commands)

This project includes a set of "smart skills" — slash commands you can type inside Claude Code to automate common Git workflows. They handle branching, committing, merging, PRs, and more so you don't have to remember all the Git commands.

**How to use:** Type the command inside your Claude Code session. You can add a description or options after it.

#### `/smart-branch` — Start new work

Creates a properly named feature branch from an up-to-date main.

```
> /smart-branch adding cloud storage support
```

Claude will detect the type (`feat`), suggest a branch name like `feat/add-cloud-storage`, confirm with you, create the branch, and optionally push it.

#### `/smart-commit` — Organize and commit changes

Analyzes all your changed files, groups them by concern (docs, tests, code, config), and creates separate conventional commits for each group.

```
> /smart-commit
```

Claude will:
1. Check which branch you're on (creates one if you're on main)
2. Group changed files by type (docs, tests, implementation, config)
3. Stage each group and commit with a proper message like `feat(parser): add YAML support`
4. Show a summary and offer to push

You never need to manually `git add` or write commit messages.

#### `/smart-merge` — Merge branches safely

Merges your feature branch into main (or another target) with safety checks, conflict handling, and optional test runs.

```
> /smart-merge
> /smart-merge strategy=squash
> /smart-merge target=develop run_tests=true
```

Claude will:
1. Detect source and target branches automatically
2. Check for uncommitted changes and warn you
3. Sync the target branch with remote
4. Merge using your chosen strategy (merge, rebase, or squash)
5. Help resolve conflicts if any arise
6. Show a summary with next steps

**Strategies:**
- `merge` (default) — preserves full history with a merge commit
- `rebase` — replays your commits for linear history
- `squash` — combines all commits into one clean commit

#### `/smart-save` — Quick WIP checkpoint

Saves all current work as a WIP commit and pushes to remote. Great for end-of-day saves, switching context, or backing up before risky operations.

```
> /smart-save
> /smart-save implementing YAML parser, 80% done
```

WIP commits can be cleaned up later with `/smart-commit` before creating a PR.

#### `/smart-pull-request` — Create a PR with quality gates

Runs quality checks (tests, linting), auto-fixes issues, then creates a GitHub pull request with an auto-generated description.

```
> /smart-pull-request
```

#### `/smart-pr-review` — Review a PR locally

Checks out a PR locally, runs tests, and helps you leave feedback or approve/request changes.

```
> /smart-pr-review 42
```

#### `/smart-cleanup` — Remove old branches

Finds and deletes merged, stale, and orphaned branches (local and remote) with safety checks.

```
> /smart-cleanup
> /smart-cleanup --dry-run
```

#### `/smart-status` — Repository overview

Shows a dashboard of your repo: current branch, open PRs, recent commits, and quick navigation actions.

```
> /smart-status
```

#### `/test` — Run tests

Runs the Go test suite and fixes any failures.

```
> /test
```

#### Typical workflow

Here's how these skills fit together for a typical feature:

```
> /smart-branch adding video thumbnails     # create branch
  ... write code ...
> /smart-save halfway done with thumbnails  # checkpoint
  ... write more code ...
> /smart-commit                              # organize commits
> /test                                      # verify tests pass
> /smart-pull-request                        # create PR
  ... PR gets reviewed and approved ...
> /smart-merge                               # merge to main
> /smart-cleanup                             # delete old branch
```

### RTK (Token Optimizer)

[RTK](https://www.rtk-ai.app/) sits between Claude Code and your terminal, compressing command output before it reaches the AI context window. This reduces token usage by up to 89%, letting you do more in each session.

Once installed, just prefix your commands with `rtk`:
```bash
rtk git status      # compact git output
rtk go test ./...   # test failures only, noise removed
rtk git diff        # compressed diffs
```

RTK is already configured for this project — all commands in `CLAUDE.md` use it.

### Claude Code Monitor (CCM)

[Claude Code Monitor](https://github.com/Maciek-roboblog/Claude-Code-Usage-Monitor) tracks your Claude Code token usage in real time with cost analysis and predictions. Run it in a separate terminal while you work:

```bash
ccm
```

It shows you how many tokens you've used, estimated costs, and alerts when you're approaching limits.

### Learn More

- Claude Code: https://docs.anthropic.com/en/docs/claude-code
- RTK: https://www.rtk-ai.app/
- Claude Code Monitor: https://github.com/Maciek-roboblog/Claude-Code-Usage-Monitor

---

## Architecture Overview

For a deeper look at the codebase — file locations, API endpoints, config format, and build details — see [CLAUDE.md](CLAUDE.md). That file is primarily for AI-assisted development but serves as thorough internal documentation.
