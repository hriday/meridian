# Meridian — packaging & release (slice 1) design

**Status:** approved by owner (2026-11-09) with three decisions below.

## Intent and why
Meridian is "core first, then open up." The core is complete (lookups with
IP/ASN/prefix/domain pages, AS-aware traceroute, subnet calc, encrypted IP
space, NTP/DNS, BGP & looking glass). This slice builds the **release
machinery** so a tagged commit produces downloadable apps for all three OSes,
and writes the public-facing **README** — without yet flipping the repo public.

## Decisions (confirmed)
1. **Unsigned for v1.** No Apple notarization / Windows Authenticode yet. The
   CI is structured so signing can be dropped in later via GitHub secrets. Docs
   tell users how to open an unsigned build (macOS right-click → Open; Windows
   SmartScreen → More info → Run anyway).
2. **Build the machinery, stay private.** Add CI + README now; test a release on
   the private repo. The owner flips the repo public themselves once they've run
   the finished app. ("Release when it's all ready.")
3. **Linux = AppImage only.** macOS = `.app` + DMG; Windows = `.exe` (portable).
   No `.deb`/Flatpak this slice.

## Goals
- A **release workflow** (`.github/workflows/release.yml`) that, on a `vX.Y.Z`
  tag, builds Meridian natively on each OS runner and attaches artifacts to a
  GitHub Release:
  - **macOS** (`macos-latest`): `wails build -platform darwin/universal` → `.app`
    → DMG (via `hdiutil`, no extra deps) → `Meridian-<ver>-macos-universal.dmg`.
  - **Windows** (`windows-latest`): `wails build -platform windows/amd64` →
    `Meridian-<ver>-windows-amd64.exe`.
  - **Linux** (`ubuntu-latest`): install `libgtk-3-dev` + `libwebkit2gtk-4.1-dev`,
    `wails build -platform linux/amd64 -tags webkit2_41` → wrap the binary into
    `Meridian-<ver>-linux-x86_64.AppImage`.
- The same workflow runs on **`workflow_dispatch`** (and non-tag pushes) to
  **build and upload artifacts for testing without creating a Release** — the
  Release step is gated on `startsWith(github.ref, 'refs/tags/')`. This is how
  we verify the pipeline on the private repo before any public tag.
- A **README.md** with: what Meridian is, a feature overview, per-OS install
  (including the unsigned-open steps), build-from-source, and a Screenshots
  section (scaffolded; images dropped in once the app is run).
- A **LICENSE** so the repo is ready to be public.
- A short **go-public checklist** (`docs/RELEASING.md`) covering: tag → release
  flow, where to add signing secrets later, and pre-public items.

## Non-goals / deferred ("soon")
- Code signing / notarization (structured for, not done).
- `.deb`, `.rpm`, Flatpak; Windows NSIS/MSI installer; Linux arm64; macOS
  separate arm64/amd64 (universal covers both).
- In-app version display (the release/tag carries the version this slice).
- Auto-update, Homebrew tap, winget, Flathub submission.
- Actually making the repo public (owner's manual step).

## Constraints (global)
- Go directive stays **1.23**; pin the Wails CLI to **v2.10.1** in CI (matches
  local). No new app dependencies — this slice is CI config + docs only; it
  must not touch `internal/`, `app`, `cli`, or the frontend logic.
- `ci.yml` (vet + test on `internal/`) stays as the test gate; `release.yml` is
  additive and independent.
- Ubuntu runner uses `libwebkit2gtk-4.1` with the `webkit2_41` build tag (the
  4.0 package is gone on ubuntu-24.04); this is the known Wails-on-CI footgun to
  get right.

## What the builder decides (marked as the builder's calls)
- **LICENSE = MIT** unless the owner prefers otherwise (permissive, standard for
  a tool like this; reversible while the repo is private — flagged in the PR).
- Exact runner image tags, action versions (pin to a recent release ≥2 weeks
  old), and the AppImage helper (`appimagetool`) invocation details.
- README wording and structure within the sections above.
- Note for the owner (not changed by the builder): `wails.json` author details
  and the git commit-author emails become public when the repo does — listed in
  the go-public checklist so the owner can scrub them before flipping.

## Verification
The pipeline is proven by a real **`workflow_dispatch` run on the private repo**:
all three matrix legs build and upload their artifact, with no Release created.
A later real `vX.Y.Z` tag (owner's call) produces the first Release.
