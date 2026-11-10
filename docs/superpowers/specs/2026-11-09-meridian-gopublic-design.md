# Meridian — Go-public prep (design)

**Status:** approved by owner (2026-11-09) with the decisions below.
**Sequencing:** run AFTER PRs #10 (AD audit), #11 (netcat), #12 (localnet) are
merged, so screenshots and the clean-history push include the complete toolkit.

## Intent and why
Get Meridian ready to be a public open-source project and cut its first release
— without paying for code-signing certificates, and without leaking the owner's
email through git history.

## Decisions (confirmed)
1. **Email history:** **delete + recreate the GitHub repo** from the clean local
   history (the one with all commits authored as `…@users.noreply.github.com`).
   Rewriting master alone doesn't purge the old-email commits GitHub keeps in
   merged-PR refs; a fresh repo guarantees none remain. The solo repo has no
   external contributors, so losing the PR/issue threads is acceptable.
2. **Signing:** **ship unsigned for v1.** No Apple/Windows certs. Two warning-free
   paths cover most users: **building from source** (locally-built apps aren't
   quarantined → no Gatekeeper/SmartScreen prompt) and a **Homebrew cask**
   (strips the quarantine flag on install). The raw `.dmg`/`.exe` downloads keep
   the documented one-time "open anyway" step. Certs can come later if traction
   warrants (Apple Developer ID; Azure Trusted Signing for Windows).
3. **Screenshots:** captured from the browser (`wails dev` serves the full UI at
   `http://localhost:34115`, so the owner's native-window issue on macOS 26 beta
   doesn't block this).

## Goals
- **Screenshots:** capture the key tools (Lookup/entity page, BGP & looking
  glass, Ping sweep, Port scanner, NTP, AD audit) into `docs/media/` and
  reference them in the README. Dark theme, representative data.
- **README finalization:** foreground **Build from source** as the clean,
  warning-free path; keep per-OS binary install with the open steps; add a
  Homebrew install line; a feature overview with screenshots; badges optional.
- **Repo hygiene:** `SECURITY.md` (how to report a vuln — a contact + "no hosting
  of others' data" scope note), `CONTRIBUTING.md` (build/test/PR conventions, the
  slice workflow in brief), a license/NOTICE check, and a final sweep confirming
  no personal email or secrets anywhere in the tree.
- **Homebrew cask:** a `meridian.rb` cask (points at the GitHub-release DMG by
  version + SHA256, installs `meridian.app`) plus setup docs for a
  `hriday/homebrew-tap` repo, so `brew install --cask hriday/tap/meridian` works
  once v0.1.0 is released. The release artifact name already matches
  (`Meridian-v<ver>-macos-universal.dmg`).
- **Go-public runbook** (`docs/RELEASING.md` expanded): the exact ordered steps —
  verify clean history, delete + recreate the repo and push, flip public, push
  the `v0.1.0` tag (CI builds + publishes all three binaries), fill the cask's
  version/sha256 from the release and push the tap.

## Non-goals / deferred
- Buying/standing up real code signing or notarization.
- Scoop/winget/Chocolatey (Windows package managers) and Flathub — Homebrew is
  the one for v1; the others are a later follow-up.
- Auto-updating, a project website, CI to auto-bump the cask on release
  (manual for v1).
- Making the repo public or pushing the tag — the owner's manual steps, from the
  runbook.

## Constraints
- No new app code or dependencies; this slice is docs + screenshots + a cask +
  the runbook. Additive only.
- Nothing in the tree may contain the personal email or any secret — verified.
- Screenshots are captured locally; no identifying data in them.

## What the builder decides
- Screenshot selection/layout and `docs/media/` filenames; README structure
  within the goals; SECURITY/CONTRIBUTING wording; the cask/tap file layout
  (suggest `packaging/homebrew/meridian.rb` in-repo as the source of truth, and
  document creating the separate tap repo from it).
