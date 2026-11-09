# Meridian

An open-source desktop toolkit for getting information about networks,
addresses and routes — and recording your own IP space. Cross-platform
(Windows, Linux, macOS), built with Go and [Wails](https://wails.io).

> Status: early development. This repository is private until the core is ready.

## Building

Requires Go 1.23+, Node 18+, and the [Wails v2 CLI](https://wails.io/docs/gettingstarted/installation).

```bash
wails dev     # run in development with hot reload
wails build   # produce a binary in build/bin/
```

Engine logic lives in pure, UI-free packages under `internal/` and is exercised by:

```bash
go test ./internal/...
```

## Licence

Apache License 2.0 — see [LICENSE](LICENSE).
