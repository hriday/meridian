# Go-public prep Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans (inline). Steps use checkbox (`- [ ]`). This slice is docs + screenshots + a Homebrew cask + a runbook (no app code); the TDD config/docs exception applies — each task states how it's verified.
>
> **PRECONDITION:** PRs #10/#11/#12 are merged into master and this branch is cut from that updated master, so screenshots and the clean history include every tool.

**Goal:** Make the repo public-ready — README finalized, repo-hygiene files, screenshots, a Homebrew cask, and a precise go-public runbook — ship unsigned, no certs.

**Spec:** `docs/superpowers/specs/2026-11-09-meridian-gopublic-design.md`

## Global Constraints
- No app code / no new deps; additive docs + a cask + images.
- Nothing in the tree contains the personal email or any secret — verified (Task 5).
- Unsigned; build-from-source is the foregrounded clean path.

## Review Focus
- The README's install claims match reality (Homebrew line resolves once released; open steps correct for current macOS; build-from-source actually works per the prerequisites).
- The cask's URL/filename matches the release workflow's artifact name exactly.
- No personal email / secret anywhere (whole tree + rendered screenshots).

---

### Task 1: Repo-hygiene files
**Files:** Create `SECURITY.md`, `CONTRIBUTING.md`; verify `LICENSE`/add `NOTICE` if needed.
- [ ] **Step 1:** `SECURITY.md` — how to report a vulnerability (a contact channel — a GitHub security advisory / issue; **no email address**), supported-versions note, and the scope line: Meridian runs locally and hosts nothing for others; the AD audit/netcat/scanners are defensive/self-use tools.
- [ ] **Step 2:** `CONTRIBUTING.md` — prerequisites, `go test ./...` + `wails build`, PR conventions, a one-paragraph description of the slice workflow, and the "scan/audit only what you're authorized to" note.
- [ ] **Step 3:** confirm `LICENSE` is Apache-2.0 and add a short `NOTICE` if Apache convention wants one (attribution for bundled fonts — Geist OFL — and any vendored notices).
- [ ] **Step 4: verify** — files render; links resolve; `grep -ri "moosha\|gmail" .` (excluding `.git`) returns nothing.
- [ ] **Step 5: commit** `docs: SECURITY, CONTRIBUTING, NOTICE for public release`.

### Task 2: Screenshots
**Files:** Create `docs/media/*.png` (several); modify `README.md` (uncomment/point at them).
- [ ] **Step 1:** run `wails dev` (background); confirm it serves `http://localhost:34115`.
- [ ] **Step 2:** drive a browser to that URL and capture the key tools at a consistent window size, dark theme: Lookup entity page (`1.1.1.1` → pivot), BGP & looking glass (`1.1.1.1`), Ping sweep, Port scanner, NTP, AD password audit, Netcat. Save to `docs/media/` with descriptive names. No identifying data in the frames.
- [ ] **Step 3:** reference 3–5 of the best in the README's Screenshots section (remove the placeholder comments).
- [ ] **Step 4: verify** — images exist, are committed, render in the README; stop `wails dev`.
- [ ] **Step 5: commit** `docs: add screenshots`.

### Task 3: README finalization
**Files:** Modify `README.md`.
- [ ] **Step 1:** restructure so **Install** leads with (a) **Homebrew** (`brew install --cask hriday/tap/meridian`), (b) per-OS binaries with the open steps, (c) **Build from source** foregrounded as the warning-free path (full prereqs already present). Tighten the feature overview; keep the privacy section; link SECURITY/CONTRIBUTING.
- [ ] **Step 2: verify** — Markdown renders; all internal links resolve; no personal email.
- [ ] **Step 3: commit** `docs: finalize README for public release`.

### Task 4: Homebrew cask + tap docs
**Files:** Create `packaging/homebrew/meridian.rb`; `docs/RELEASING.md` (expanded, see Task 5).
- [ ] **Step 1:** write `meridian.rb` — `version`, `sha256 :no_check` placeholder (to be pinned at release), `url` built from the release DMG name `Meridian-v#{version}-macos-universal.dmg`, `name`/`desc`/`homepage`, `app "meridian.app"`. A comment on how it's published to the tap.
- [ ] **Step 2: verify** — `ruby -c packaging/homebrew/meridian.rb` parses (if ruby present), else structural review; the URL matches `release.yml`'s artifact name.
- [ ] **Step 3: commit** `packaging: homebrew cask`.

### Task 5: Go-public runbook + final sweep
**Files:** Modify `docs/RELEASING.md`.
- [ ] **Step 1:** expand with the ordered go-public runbook: (1) verify clean history (`git log --format='%ae' | sort -u` → only noreply); (2) create a fresh GitHub repo (or delete+recreate `hriday/meridian`) and `git push` the clean master + the `v0.1.0` tag; (3) flip the repo **Public**; (4) the `v*` tag triggers the release build → binaries published; (5) pin the cask's `version`+`sha256` from the macOS DMG and push the `hriday/homebrew-tap` repo; (6) post-public checklist (screenshots present, no email, LICENSE, Scoop/winget later, signing later).
- [ ] **Step 2: final sweep** — `grep -rI "moosha\|gmail\|@users.noreply" .` (excluding `.git`) and `git log --all --format='%ae%n%ce' | sort -u` to confirm the only emails are noreply; record the result.
- [ ] **Step 3: commit** `docs: go-public runbook`.

## Self-review notes
- **Spec coverage:** hygiene (T1), screenshots (T2), README (T3), cask (T4), runbook+sweep (T5). Signing and actually-going-public are deliberate non-goals (runbook hands them to the owner).
- **Review Focus:** install-claims accuracy (T3), cask-URL match (T4), no-email/secret sweep (T1/T5).
- **Proportion:** all docs/assets; no code, no tests beyond rendering/parse/grep verification.
