# Releasing Meridian

## Cut a release
Releases are built by `.github/workflows/release.yml` from a version tag.

```bash
git tag v0.1.0
git push origin v0.1.0
```

The workflow builds on `macos-latest`, `windows-latest`, and `ubuntu-latest`,
packages each (`.dmg` / `.exe` / `.AppImage`), and attaches all three to a new
GitHub Release named for the tag (with auto-generated notes). Use semver
(`vMAJOR.MINOR.PATCH`).

## Test the pipeline without releasing
The same workflow runs on pushes to the `packaging` branch and via
**Actions → Release → Run workflow** (`workflow_dispatch`). On a non-tag run it
builds and uploads the three binaries as **workflow artifacts** and **does not
create a Release** (the release job is gated on `refs/tags/`). Download the
artifacts from the run to smoke-test a build.

## Code signing

**macOS — done (since v1.18.9).** `release.yml` imports an Apple *Developer ID
Application* cert, codesigns the `.app` with the hardened runtime, notarizes the
DMG, and staples the ticket, so the build opens with no first-launch workaround.
It runs only when the signing secrets are present (Settings → Secrets and
variables → Actions):

- `MACOS_CERTIFICATE_P12` (base64 of the `.p12`) + `MACOS_CERTIFICATE_PASSWORD`
- `MACOS_SIGN_IDENTITY` (full cert name) + `MACOS_TEAM_ID` (10-char code)
- `MACOS_NOTARY_APPLE_ID` + `MACOS_NOTARY_PASSWORD` (app-specific password)

The codesign step auto-detects the Developer ID identity from the keychain, and
notarization polls Apple in a loop that survives transient network blips. The
`packaging` branch runs the full sign+notarize pipeline **without publishing**,
for pre-tag verification.

**Windows — still unsigned.** To drop SmartScreen warnings, sign `meridian.exe`
with `signtool` using an Authenticode cert in `WINDOWS_CERT_PFX` (base64) +
`WINDOWS_CERT_PASSWORD`. These secret names are placeholders — nothing references
them yet.

## Pre-public checklist
Before flipping the repo to public:

- [ ] You've run the finished app on at least one OS and are happy with it.
- [ ] **No personal email is exposed** — check `wails.json` author details and,
      importantly, the **git commit-author emails** (`git log --format='%ae' | sort -u`),
      which become public with the repo. Scrub via history rewrite if needed.
- [ ] Confirm the **LICENSE** (currently Apache-2.0) is the one you want.
- [ ] Add real screenshots under `docs/media/` and uncomment them in `README.md`.
- [ ] Review the README's install/privacy wording.
- [ ] Decide the first public version tag (e.g. `v0.1.0`).
