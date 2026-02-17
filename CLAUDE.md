# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**crev** is an interactive TUI code review tool written in Go. It displays git diffs in a two-pane interface (file tree sidebar + diff view) where users can add inline comments with severity levels, then submit structured JSON reviews. It integrates with Claude Code as a skill (`/review`) and command (`/crev`).

## Build Commands

- `make build` — compile the `crev` binary
- `make run` — build and run locally
- `make install` — build, codesign (macOS), and install to `~/.claude/skills/review/` with symlinks in `~/.local/bin/`
- `make test` — run all tests (`go test -v ./...`)
- `make fmt` — format code (`go fmt ./...`)
- `make lint` — lint with `golangci-lint run`
- `make build-all` — cross-compile for darwin/linux (amd64 + arm64)

Run a single test: `go test -v -run TestName ./internal/diff/`

## Architecture

```
cmd/crev/main.go      — Entry point, CLI flags, git operations (status + diff)
internal/
  diff/
    types.go           — Data structures: DiffResult, File, Hunk, Line
    parser.go          — Unified diff parser using regex
  review/
    types.go           — Review, Comment, Severity types
    output.go          — JSON serialization of review output
  tui/
    app.go             — Main bubbletea Model: state, Update(), View()
    diff_view.go       — Diff pane rendering (line numbers, +/- prefixes)
    sidebar.go         — File tree sidebar rendering
    tree.go            — Tree data structure & flat-list navigation
    keys.go            — Keybinding definitions
    styles.go          — lipgloss terminal styling/colors
```

### Key Patterns

- **Bubbletea (Elm architecture)**: Model holds all state, `Update()` handles messages, `View()` renders. View functions are pure and never modify state.
- **Mode-based input handling**: The TUI has modes (Normal, Commenting, Help, Summary, QuitConfirm) that determine which keys are active in `Update()`.
- **Focus system**: Input routing switches between Sidebar and Main area focus.
- **Tree → flat list**: The file tree is converted to a flat list for keyboard navigation with `j/k`.
- **Git integration**: Uses `os/exec` to shell out to `git status` and `git diff`. Handles staged, unstaged, and untracked files.

### CLI Flags

`-o` output file, `-d` directory, `-staged` staged only, `-json` raw JSON output, `-version`
