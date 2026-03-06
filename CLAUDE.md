# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

A portable video sorting tool for videographers. Users watch short sports videos, annotate them with subjects (player numbers), tags, and quality ratings, then the app renames files encoding metadata into the filename (e.g., `clip__S88__double_slide__great.mp4`). Single binary, cross-platform, opens in browser.

## Architecture

**Single-binary web app:** Go backend with `embed.FS` serves an HTML/JS/CSS frontend (`index.html`) via local HTTP on a random port, then auto-opens the browser. No external dependencies, no node_modules, no build tools for the frontend.

- `main.go` — HTTP server, REST API endpoints, config file parsing, session persistence
- `index.html` — Complete SPA frontend (Tailwind via CDN, vanilla JS, no framework)

**Key patterns:**
- `//go:embed index.html` embeds the frontend at compile time
- Config stored as human-readable `video-sorter-config.txt` alongside videos (section-based, one entry per line)
- Session persisted to `~/.video-sorter-session.json` (remembers last directory, file, MRU order)
- Video must be unloaded (`pause()` + remove `src` + `load()`) before rename on Windows due to file locking

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
