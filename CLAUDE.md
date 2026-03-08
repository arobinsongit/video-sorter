# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

A portable media sorting tool. Users browse videos and photos, annotate them with configurable metadata groups (subjects, tags, quality, or any custom group), then the app renames/moves/copies files encoding metadata into the filename (e.g., `clip__S88__double_slide__great.mp4`). Single binary, cross-platform, opens in browser. Supports video (.mp4, .mov, .avi, .mkv, .webm) and photo (.jpg, .jpeg, .png, .gif, .webp, .bmp, .tiff, .heic, .heif) files.

## Architecture

**Single-binary web app:** Go backend with `embed.FS` serves a bundled frontend via local HTTP on a random port, then auto-opens the browser. Frontend is built with esbuild (minified + bundled), embedded at compile time.

### Backend (`cmd/` + `internal/`)
- `cmd/media-sorter/main.go` — Entry point, server startup, browser launch, `//go:embed` for static assets
- `internal/server/` — HTTP server, API handlers, config management:
  - `server.go` — Server struct, route registration, provider routing, JSON helpers
  - `api.go` — Core API handlers (list, media, config, rename, open-folder)
  - `api_cloud.go` — Cloud API handlers (providers, credentials, connect, disconnect, browse, OAuth callback)
  - `api_session.go` — Session and user settings handlers
  - `config.go` — ProjectConfig types, load/save/migrate, legacy format parsing
- `internal/storage/` — Storage abstraction layer:
  - `storage.go` — Provider interface, FileInfo, MediaExts, LocalStorage, cloud path helpers (CloudJoin/CloudDir/CloudBase)
  - `gdrive/` — Google Drive provider (OAuth2, Drive API v3, embedded credentials via ldflags)
  - `s3/` — Amazon S3 provider (AWS SDK v2, access key credentials)
  - `dropbox/` — Dropbox provider (OAuth2, Dropbox HTTP API v2)
  - `onedrive/` — OneDrive provider (OAuth2, Microsoft Graph API v1.0)

### Frontend (source in `frontend/`)
- `frontend/index.html` — HTML shell with Tailwind CSS via CDN
- `frontend/src/` — ES modules bundled by esbuild:
  - `main.js` — Entry point, initialization, media player, keyboard shortcuts
  - `state.js` — Shared state object
  - `api.js` — All fetch wrappers for backend endpoints
  - `groups.js` — Dynamic metadata group rendering (multi-select, single-select, slider)
  - `preview.js` — Template-based filename preview + annotation parsing
  - `fileList.js` — File list with sorting and filtering
  - `configEditor.js` — Settings editor modal (output format, mode, groups)
  - `theme.js` — Dark/light theme toggle + button class helpers
  - `modal.js` — Reusable modal dialog
  - `utils.js` — MRU helpers, formatSize, clearChildren
  - `cloud.js` — Cloud storage connection UI (Google Drive, S3, Dropbox, OneDrive)

### Build output
- `cmd/media-sorter/static/` — Built frontend files (index.html, app.min.js, favicon.svg) — committed to git so `go build` works without npm
- `dist/` — Release binaries (gitignored)

**Key patterns:**
- Config stored as `media-sorter-config.json` per media folder
- Session persisted to `~/.media-sorter-session.json` (remembers last directory, file, MRU order per group)
- User settings at `~/.media-sorter-settings.json` (theme, keybindings)
- Media player must be unloaded (`pause()` + remove `src` + `load()`) before rename on Windows due to file locking
- MRU (Most Recently Used) sorting for all group options
- Dynamic group rendering from config — no hardcoded metadata types
- Template-based output format with `{token}` placeholders
- Cloud storage via StorageProvider interface — `gdrive://`, `s3://`, `dropbox://`, `onedrive://` path prefixes
- Cloud credentials stored in `~/.media-sorter/` (OAuth tokens, client credentials, S3 access keys)

## Config Format

Stored as `media-sorter-config.json` in the media folder:
```json
{
  "version": 1,
  "outputFormat": "{basename}__{S}__{tags}__{quality}.{ext}",
  "outputFolder": "",
  "outputMode": "rename",
  "groups": [
    {
      "name": "Subject", "key": "S", "type": "multi-select",
      "inputType": "number", "options": ["1","2",...],
      "allowCustom": true, "separator": "__", "prefix": "S"
    },
    {
      "name": "Tags", "key": "tags", "type": "multi-select",
      "inputType": "text", "options": ["slide","catch",...],
      "allowCustom": true, "separator": "_", "prefix": ""
    },
    {
      "name": "Quality", "key": "quality", "type": "single-select",
      "inputType": "slider", "options": ["bad","ok","good","great"],
      "allowCustom": false, "separator": "", "prefix": ""
    }
  ]
}
```

Group types: `multi-select` (toggle buttons, Set), `single-select` (radio buttons or slider).
Input types: `number`, `text`, `slider`.
Output modes: `rename` (in place), `move` (to outputFolder), `copy` (to outputFolder).

## API Endpoints

| Endpoint | Method | Purpose |
|---|---|---|
| `/` | GET | Serve embedded frontend |
| `/api/list?dir=` | GET | List media files with metadata |
| `/api/media?dir=&file=` | GET | Serve media file for playback |
| `/api/config?dir=` | GET | Read JSON config (auto-migrates from legacy formats) |
| `/api/config/save` | POST | Write JSON config `{dir, config}` |
| `/api/rename` | POST | Rename/move/copy file `{dir, oldName, newName, outputMode, outputFolder}` |
| `/api/session` | GET | Load session state |
| `/api/session/save` | POST | Save session state |
| `/api/user-settings` | GET/POST | Read/write user settings |
| `/api/open-folder?dir=` | GET | Open directory in OS file explorer |
| `/api/cloud/providers` | GET | List cloud providers and connection status |
| `/api/cloud/credentials` | POST | Save provider credentials from UI `{provider, credentials}` |
| `/api/cloud/connect` | POST | Initiate OAuth flow or direct connect `{provider}` |
| `/api/cloud/disconnect` | POST | Remove stored credentials `{provider}` |
| `/api/cloud/browse?provider=&path=` | GET | Browse cloud folder structure |
| `/api/cloud/callback` | GET | OAuth callback handler |

## Build

```bash
# Prerequisites: Node.js 18+, Go 1.21+

# Dev: build frontend + Go binary (Windows)
npm install          # first time only
npm run build        # bundles JS, copies HTML/favicon to cmd/media-sorter/static/
source .env.local    # loads GDRIVE_CLIENT_ID and GDRIVE_CLIENT_SECRET
go build -ldflags "-X media-sorter/internal/storage/gdrive.embeddedClientID=$GDRIVE_CLIENT_ID -X media-sorter/internal/storage/gdrive.embeddedClientSecret=$GDRIVE_CLIENT_SECRET" -o media-sorter.exe ./cmd/media-sorter && ./media-sorter.exe

# Dev: build frontend + Go binary (Mac/Linux)
npm install && npm run build
source .env.local
go build -ldflags "-X media-sorter/internal/storage/gdrive.embeddedClientID=$GDRIVE_CLIENT_ID -X media-sorter/internal/storage/gdrive.embeddedClientSecret=$GDRIVE_CLIENT_SECRET" -o media-sorter ./cmd/media-sorter && ./media-sorter

# Cross-platform release builds
bash build.sh        # from Mac/Linux
build.bat            # from Windows
```

**Quick rebuild & run (Windows):** Kill existing process, rebuild, and start:
```bash
taskkill //F //IM media-sorter.exe 2>/dev/null; source .env.local && npm run build && go build -ldflags "-X media-sorter/internal/storage/gdrive.embeddedClientID=$GDRIVE_CLIENT_ID -X media-sorter/internal/storage/gdrive.embeddedClientSecret=$GDRIVE_CLIENT_SECRET" -o media-sorter.exe ./cmd/media-sorter && ./media-sorter.exe
```

**Quick rebuild & run (Mac/Linux):** Kill existing process, rebuild, and start:
```bash
pkill -f media-sorter 2>/dev/null; source .env.local && npm run build && go build -ldflags "-X media-sorter/internal/storage/gdrive.embeddedClientID=$GDRIVE_CLIENT_ID -X media-sorter/internal/storage/gdrive.embeddedClientSecret=$GDRIVE_CLIENT_SECRET" -o media-sorter ./cmd/media-sorter && ./media-sorter
```

**Google Drive credentials:** `.env.local` (gitignored) holds `GDRIVE_CLIENT_ID` and `GDRIVE_CLIENT_SECRET` for local builds. CI uses GitHub repository secrets.

## Filename Encoding Format

Template-based: `{basename}__{S}__{tags}__{quality}.{ext}` (configurable)

Default encoding: `basename__S{subject1}__S{subject2}__tag1_tag2__quality.ext`
- `__` (double underscore) separates sections
- `S` prefix for subjects (configurable per group)
- Quality values: `great`, `good`, `ok`, `bad`
- A file is considered "tagged" if its name contains `__`

## CI/CD

GitHub Actions workflow (`.github/workflows/build.yml`) runs on every push to main:
1. Installs Node.js 22 + Go
2. Builds frontend (`npm install && npm run build`)
3. Builds all 4 platform binaries
4. Uploads as build artifacts
5. Creates a GitHub Release tagged `build-<short-sha>`

## Git Push

Before pushing, switch to the correct GitHub account. Set `GH_PUSH_USER` below (or reference an env var with `ENV:VAR_NAME`).

**GitHub push user:** `arobinsongit`

```bash
# Read push user from CLAUDE.md instruction above (or resolve ENV: reference)
RAW_USER="arobinsongit"  # ← keep in sync with "GitHub push user" above
if [[ "$RAW_USER" == ENV:* ]]; then RAW_USER="${!RAW_USER#ENV:}"; fi
PREV_USER=$(gh auth status 2>&1 | grep "Active account: true" -B3 | head -1 | awk '{print $NF}')
if [ "$PREV_USER" != "$RAW_USER" ]; then gh auth switch --user "$RAW_USER"; fi
git push origin <branch>
if [ "$PREV_USER" != "$RAW_USER" ] && [ -n "$PREV_USER" ]; then gh auth switch --user "$PREV_USER" 2>/dev/null || true; fi
```

To use an env var instead, change the push user line to e.g. `ENV:GH_REPO_USER`.

## Frontend Conventions

- ES modules bundled by esbuild into single minified `app.min.js`
- Tailwind CSS via CDN — no build step for CSS
- All state centralized in `state.js` — plain object with files, config, selections, MRU
- Dynamic group rendering — `groups.js` generates UI from config definition
- No framework dependencies — vanilla JS with DOM APIs
