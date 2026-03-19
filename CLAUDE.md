# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

Grove Street is a Go CLI that plays GTA San Andreas (CJ) voice lines as audio notifications for AI coding agents. It hooks into Claude Code's event system (Stop, Notification, SubagentStop, PreCompact) and plays a random sound from the matching category.

## Build & Test

```sh
go build ./...                          # build all packages
go build -o grove-street ./cmd/grove-street  # build the binary
go test ./...                           # run all tests
go vet ./...                            # lint
```

The `cmd/grove-street` directory is the main entrypoint (referenced in release workflow but not yet created). Version is injected via `-ldflags "-X main.version=..."` at build time. The `VERSION` file tracks the current version (0.1.0).

## Architecture

The project is a standard Go CLI with four internal packages:

- **`internal/hooks`** — Classifies Claude Code hook events (JSON payloads from stdin) into sound categories: `session_start`, `task_complete`, `task_error`, `input_required`, `resource_limit`, `user_spam`. The classification logic is in `Classify()` which maps hook types + stop reasons/messages to categories.
- **`internal/player`** — Picks a random audio file from `~/.grove-street/sounds/<category>/` and plays it using platform-specific commands (`afplay` on macOS, PipeWire/PulseAudio/FFmpeg/mpv/ALSA on Linux, PowerShell MediaPlayer on Windows). Audio playback is non-blocking (`cmd.Start()`).
- **`internal/config`** — Reads/writes `~/.grove-street/config.json` (enabled, volume, auto_update). Falls back to defaults if missing.
- **`internal/updater`** — Self-update mechanism via GitHub Releases API. Downloads new binary, does atomic swap (rename old, rename new, delete old).

## Key Design Decisions

- Sound files (.wav/.mp3/.ogg) are stored at runtime in `~/.grove-street/sounds/<category>/`, not in the repo (gitignored). The repo `sounds/manifest.json` describes the voice lines metadata.
- Audio playback is fire-and-forget (`cmd.Start()` without `Wait()`).
- Cross-platform: macOS, Linux, Windows — each with its own audio player detection chain.
- Hooks are registered in `~/.claude/settings.json` — the `grove-street hook` command receives JSON on stdin from Claude Code.

## Testing Requirements

- **Always run `go test ./...` before committing** to ensure no regressions.
- **Update or add tests** when modifying or adding functionality. Every package should have a `*_test.go` file with table-driven tests covering its exported and key unexported functions.
- Tests must not depend on real filesystem state (e.g., `~/.grove-street/config.json`). Use `t.TempDir()` or mock data.

## Release Process

Releases are done via the `workflow_dispatch` trigger on `.github/workflows/release.yml`. The workflow handles everything automatically — version bump, tagging, cross-compilation, GitHub Release creation, and Homebrew formula updates.

### How to release

1. **Trigger the workflow** from the GitHub Actions UI or CLI:
   ```sh
   gh workflow run release.yml -f version=X.Y.Z
   ```
2. **That's it.** The workflow does the rest:
   - Bumps `VERSION` file and commits to `main`
   - Creates (or replaces) the `vX.Y.Z` git tag
   - Cross-compiles for darwin/linux/windows (amd64+arm64)
   - Packages the binary + `icon.png` + notification scripts into tarballs
   - Packages `sounds/` directory into `sounds.tar.gz` and `sounds.zip`
   - Creates a GitHub Release with auto-generated release notes
   - Computes sha256 checksums from the release assets
   - Updates `Formula/grove-street.rb` with the new version and checksums, commits to `main`
   - Pushes the updated formula to the `notuselessdev/homebrew-tap` repo (requires `HOMEBREW_TAP_TOKEN` secret)

### Secrets required

- `GITHUB_TOKEN` — automatic, used for releases and formula commits
- `HOMEBREW_TAP_TOKEN` — PAT with write access to `notuselessdev/homebrew-tap`

### Notes

- The workflow is idempotent: re-running with the same version deletes the old tag and release first.
- Version format is `X.Y.Z` (no `v` prefix) — the workflow adds the `v` prefix for tags.
- Do NOT manually edit `VERSION` or tag — let the workflow handle it.
