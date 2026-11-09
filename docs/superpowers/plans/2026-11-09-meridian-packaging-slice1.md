# Packaging & release (slice 1) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans (inline) to implement this plan. Steps use checkbox (`- [ ]`) syntax. This slice is CI config + docs (not unit-testable Go); the TDD "write a failing test first" rule yields to its config/generated-code exception — each task states how it is verified instead.

**Goal:** A tagged commit builds Meridian for macOS/Windows/Linux and attaches the artifacts to a GitHub Release; the repo gains a public-ready README and LICENSE — without going public yet.

**Architecture:** One additive GitHub Actions workflow (`release.yml`) with a 3-OS build matrix on native runners, each running `wails build` for its platform and packaging (DMG / `.exe` / AppImage), then a gated release job. Docs (README, LICENSE, RELEASING) are static files. Nothing under `internal/`, `app`, `cli`, or the frontend is touched.

**Tech Stack:** GitHub Actions; Wails CLI v2.10.1; Go 1.23; Node for the frontend build; `hdiutil` (macOS DMG), `appimagetool` (Linux AppImage).

**Spec:** `docs/superpowers/specs/2026-11-09-meridian-packaging-slice1-design.md`

## Global Constraints
- Go directive **1.23**; Wails CLI pinned **v2.10.1** in CI.
- Additive only: do not modify `ci.yml`, `internal/`, `app`, `cli`, or frontend logic.
- Ubuntu: `libgtk-3-dev` + `libwebkit2gtk-4.1-dev`, build with `-tags webkit2_41`.
- Pin every action to a release ≥2 weeks old.

## Review Focus
- **Non-tag trigger must not create a Release** — `workflow_dispatch` / branch push builds + uploads artifacts only; the release job is gated `if: startsWith(github.ref, 'refs/tags/')`. (Verified by the dispatch run producing artifacts and no Release.)
- **Linux webkit version** — wrong package/tag (`4.0` vs `4.1`) is the classic Wails-CI build failure; the build must use `libwebkit2gtk-4.1-dev` + `-tags webkit2_41`.
- **Artifact naming carries the version** — assets must embed the tag (e.g. `Meridian-v0.1.0-macos-universal.dmg`), not collide across OSes.
- **`contents: write` permission** — the release job needs it or the upload 403s.
- **DMG/AppImage actually contain a runnable app** — not an empty or mis-pathed bundle (spot-checked from the dispatch artifacts).

---

### Task 1: Release workflow + packaging

**Files:**
- Create: `.github/workflows/release.yml`
- Create: `build/appimage/meridian.desktop`
- Create: `build/appimage/build-appimage.sh` (bash helper the Linux leg calls)

**Interfaces:**
- Produces, per OS leg, one uploaded artifact named `Meridian-<ref>-<os>.<ext>`; a gated release job attaches all three to the tag's Release.

- [ ] **Step 1: Write `build/appimage/meridian.desktop`** — a minimal desktop entry (`Name=Meridian`, `Exec=meridian`, `Icon=meridian`, `Categories=Network;Utility;`, `Type=Application`).

- [ ] **Step 2: Write `build/appimage/build-appimage.sh`** — takes the built `build/bin/meridian` binary and `$VERSION`, assembles an `AppDir` (binary at `usr/bin/meridian`, the `.desktop` at root, `build/appicon.png` as `meridian.png`), downloads `appimagetool` (pinned release), and runs it to emit `Meridian-<version>-linux-x86_64.AppImage`. `set -euo pipefail`; echo the output path.

- [ ] **Step 3: Write `.github/workflows/release.yml`** — `on: push: tags: ['v*']` plus `workflow_dispatch` and `on: push` to the `packaging` branch (for pre-merge verification); `permissions: contents: write`. A `build` job with `strategy.matrix.include` of `{os: macos-latest, platform: darwin/universal, ext: dmg}`, `{os: windows-latest, platform: windows/amd64, ext: exe}`, `{os: ubuntu-latest, platform: linux/amd64, ext: AppImage}`. Steps: checkout; `setup-go@<pinned>` go 1.23; `setup-node@<pinned>`; `go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.1`; (ubuntu) `apt-get install -y libgtk-3-dev libwebkit2gtk-4.1-dev`; compute `VERSION` from `github.ref_name` (fallback `dev-<sha>` for dispatch); `wails build` with the leg's `-platform` (+ `-tags webkit2_41` on linux); package per OS (macOS `hdiutil create … UDZO` from `build/bin/meridian.app`; windows rename `build/bin/meridian.exe`; linux `bash build/appimage/build-appimage.sh`); `actions/upload-artifact@<pinned>` the result. A `release` job `needs: build`, `if: startsWith(github.ref, 'refs/tags/')`, downloads artifacts and runs `softprops/action-gh-release@<pinned>` with `files:` and `generate_release_notes: true`.

- [ ] **Step 4: Verify locally** — `actionlint .github/workflows/release.yml` (or, if unavailable, a YAML parse via `python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/release.yml'))"`), and `bash -n build/appimage/build-appimage.sh`. Expected: clean.

- [ ] **Step 5: Commit**
```bash
git add .github/workflows/release.yml build/appimage/
git commit -m "ci: release workflow building macOS/Windows/Linux artifacts on tag"
```

---

### Task 2: README, LICENSE, releasing guide

**Files:**
- Create: `README.md`
- Create: `LICENSE`
- Create: `docs/RELEASING.md`
- Create: `docs/media/.gitkeep` (screenshots dropped here later)

- [ ] **Step 1: Write `LICENSE`** — MIT, `Copyright (c) 2026 Hriday Balachandran`. (Builder's call per spec; flagged in the PR as changeable while private.)

- [ ] **Step 2: Write `README.md`** — title + one-line pitch; a Features section grouped as the app is (Internet: Lookup / Traceroute (AS) / BGP & looking glass; Addressing: Subnet / IP space; Local-environment: NTP / DNS); a Screenshots section referencing `docs/media/*.png` with a note they're added per release; Install (macOS DMG + right-click-Open note; Windows `.exe` + SmartScreen note; Linux AppImage `chmod +x` + run); Build from source (`wails dev` / `wails build`, Go 1.23 + Node + Wails 2.10.1); a Privacy line (outbound queries only on user action to RDAP/RIPEstat/Team-Cymru/NTP/DNS with a generic User-Agent; IP-space stays local + encrypted); License line.

- [ ] **Step 3: Write `docs/RELEASING.md`** — how to cut a release (`git tag vX.Y.Z && git push --tags` → workflow builds + publishes); how to test without releasing (`workflow_dispatch`); where signing secrets go later (Apple Developer ID + `.p12`/password, Windows cert — named env/secret keys, marked TODO); and a **pre-public checklist** (repo visibility, `wails.json` author + git commit-author emails — scrub before flipping, LICENSE choice, screenshots added).

- [ ] **Step 4: Verify** — files render as valid Markdown (headings, fenced blocks balanced); internal links (`docs/RELEASING.md`, `docs/media/`) resolve. Expected: clean.

- [ ] **Step 5: Commit**
```bash
git add README.md LICENSE docs/RELEASING.md docs/media/.gitkeep
git commit -m "docs: public README, MIT LICENSE, releasing guide"
```

---

### Task 3: Prove the pipeline (real CI dispatch run)

**Files:** none (verification task).

- [ ] **Step 1: Push the branch** — `git push -u origin packaging` so the workflow (which triggers on push to `packaging`) runs, or trigger `workflow_dispatch` via `gh workflow run release.yml --ref packaging`.

- [ ] **Step 2: Watch the run** — `gh run watch` / `gh run view`. Expected: all three matrix legs succeed and upload their artifact; the `release` job is **skipped** (non-tag).

- [ ] **Step 3: Fix forward** — if a leg fails (most likely the Linux webkit tag or the AppImage step), read the log, correct `release.yml`/`build-appimage.sh`, commit, and re-run until green. Each fix is its own commit.

- [ ] **Step 4: Spot-check an artifact** — download the macOS DMG (or Linux AppImage) from the run and confirm it contains the app bundle/binary (`hdiutil attach` / `--appimage-extract` or `unzip -l`). Expected: a real Meridian app inside.

- [ ] **Step 5: Record** — note in the ledger the green run URL and the artifact names. No code commit unless a fix was needed.

---

## Self-review notes
- **Spec coverage:** release workflow (T1), README/LICENSE/releasing guide (T2), proven pipeline (T3). Signing is deliberately absent (non-goal, structured-for in RELEASING.md). Going public is absent (owner's manual step).
- **Review Focus:** each of the five rows is exercised by T3's real run (non-tag → no Release; webkit tag; artifact names; permissions; artifact contents spot-check).
- **Proportion:** three tasks; the workflow YAML is the substance, docs are static, T3 is verification. No logic to over-specify.
- **Additive-only:** no task edits `ci.yml`, `internal/`, `app`, `cli`, or frontend — the test gate and the shipped engine are untouched.
