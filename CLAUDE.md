# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

A portable video sorting tool for videographers. Users watch short sports videos, annotate them with subjects (player numbers), tags, and quality ratings, then the app renames files encoding metadata into the filename (e.g., `clip__S88__double_slide__great.mp4`). Single binary, cross-platform, opens in browser.

## Architecture

**Single-binary web app:** Go backend with `embed.FS` serves an HTML/JS/CSS frontend (`index.html`) via local HTTP on a random port, then auto-opens the browser. No external dependencies, no node_modules, no build tools for the frontend.

- `main.go` — HTTP server, REST API endpoints, config file parsing, session persistence
- `index.html` — Complete SPA frontend (Tailwind via CDN, vanilla JS, no framework)
- `favicon.svg` — App icon (purple gradient play button with film strip and orange tag)
- `.github/workflows/build.yml` — CI: builds all platforms on push to main, creates GitHub Release with binaries

**Key patterns:**
- `//go:embed index.html` and `//go:embed favicon.svg` embed assets at compile time
- Config stored as human-readable `video-sorter-config.txt` alongside videos (section-based: `# Subjects`, `# Tags`, one entry per line)
- Session persisted to `~/.video-sorter-session.json` (remembers last directory, file, MRU order)
- Video must be unloaded (`pause()` + remove `src` + `load()`) before rename on Windows due to file locking
- MRU (Most Recently Used) sorting for subjects and tags — last-clicked items appear first
- Modal callback pattern: save callback ref before `hideModal()` nullifies it

## API Endpoints

| Endpoint | Method | Purpose |
|---|---|---|
| `/` | GET | Serve embedded HTML |
| `/api/list?dir=` | GET | List video files with metadata |
| `/api/video?dir=&file=` | GET | Serve video file for playback |
| `/api/config?dir=` | GET | Read config text file as JSON |
| `/api/config/save` | POST | Write config from JSON to text file |
| `/api/rename` | POST | Rename video file (metadata encoding) |
| `/api/session` | GET | Load session state |
| `/api/session/save` | POST | Save session state |
| `/api/open-folder?dir=` | GET | Open directory in OS file explorer |
| `/favicon.svg` | GET | Serve embedded favicon |

## Build

```bash
# Dev: build and run (Windows)
go build -o video-sorter.exe . && ./video-sorter.exe

# Dev: build and run (Mac/Linux)
go build -o video-sorter . && ./video-sorter

# Cross-platform release builds (outputs to dist/)
bash build.sh        # from Mac/Linux
build.bat            # from Windows
```

Builds produce 4 binaries: Windows amd64, Mac Intel, Mac ARM, Linux amd64. Uses `-ldflags="-s -w"` for size optimization.

**Important on Windows:** Kill existing processes before rebuilding — the browser holds a connection that keeps the exe locked:
```bash
taskkill //F //IM video-sorter.exe 2>/dev/null; go build -o video-sorter.exe .
```

## Filename Encoding Format

`basename__S{subject1}_S{subject2}__tag1_tag2__quality.ext`

- `__` (double underscore) separates sections
- `S` prefix for subjects (not `P` — may not be players)
- Quality values: `great`, `good`, `ok`, `bad`
- A file is considered "tagged" if its name contains `__`
- Special suffix `__review` marks a file as flagged for review

## CI/CD

GitHub Actions workflow (`.github/workflows/build.yml`) runs on every push to main:
1. Builds all 4 platform binaries
2. Uploads as build artifacts
3. Creates a GitHub Release tagged `build-<short-sha>` with all binaries attached

Releases: https://github.com/arobinsongit/video-sorter/releases

## Config File Format

Stored as `video-sorter-config.txt` in the video folder:
```
# Subjects
# One per line
21
8
93

# Tags
# One per line (use dashes instead of spaces, e.g. home-run)
slide
home-run
warmup
```

Parsed by `parseConfigText()` / written by `buildConfigText()` in `main.go`.

## Frontend Conventions

- Vanilla JS with Tailwind CSS via CDN — no build step, no framework
- All state is in plain variables/Sets at the top of the `<script>` block
- `table-layout: fixed` with explicit pixel widths for file list columns
- Light/dark mode toggle stored in `localStorage('theme')`
- `selectedSubjects` is a Set (multi-select), `selectedTags` is a Set (toggle on/off)
- `mruBump(arr, val)` / `mruSort(items, mru)` handle MRU ordering

## Issue Backlog

Feature issues are tracked on GitHub Issues (#1-#24). Key themes:
- Speed: keyboard shortcuts (#1), auto-play (#2), smart suggestions (#16), voice commands (#22)
- Visibility: untagged filter (#3), progress bar (#9), color-coded rows (#14), duration column (#13)
- Batch ops: batch tagging (#7), apply to all visible (#18), presets (#8)
- Infrastructure: audit log (#21), folder history (#11), cloud storage (#24), iOS app (#23)
