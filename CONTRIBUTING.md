# Contributing to Meridian

Thanks for your interest. Meridian is a cross-platform network toolkit — a Go
engine with a Wails (Svelte) UI and a CLI.

## Prerequisites

- **Go 1.23+**, **Node 20+**, and the **Wails v2.10.1** CLI:
  `go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.1`
- Per-OS toolchain deps are listed in the README's **Build from source** section
  (GTK/WebKit on Linux, Xcode CLT on macOS, WebView2 on Windows). Run
  `wails doctor` to check your environment.

## Developing

```bash
wails dev          # hot-reload; also serves the UI at http://localhost:34115
go test ./...      # engine + CLI tests
gofmt -l .         # formatting check
wails build        # release build → build/bin/
```

**Architecture:** engine logic lives in pure, UI-free packages under
`internal/` and is exercised by `go test ./...`. The Wails controller
(`internal/app`) and the CLI (`internal/cli`) are thin adapters over those
packages — keep business logic out of them. External effects (crackers, ICMP,
HTTP) sit behind small interfaces so the engine is tested offline.

## Pull requests

- Keep changes focused and add tests for engine/CLI logic (TDD preferred —
  write the failing test first).
- Run `go test ./...` and `gofmt -w` before pushing; `wails build` should stay
  green.
- Design/plan docs for each slice live under `docs/superpowers/`.

## Scope and conduct

Meridian is a **defensive/diagnostic** toolkit. Please don't add stealth,
evasion, or attack features, or anything that scans, cracks, or targets systems
you aren't authorized to test. Be respectful in issues and reviews.
