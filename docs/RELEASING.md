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

## Adding code signing later (not done in v1)
v1 ships **unsigned**. To remove the first-launch OS warnings, add signing to
`release.yml` using repository **secrets** (Settings → Secrets and variables →
Actions):

- **macOS notarization** — import an Apple *Developer ID Application* cert from
  `MACOS_CERT_P12` (base64) + `MACOS_CERT_PASSWORD`, sign the `.app`, then
  notarize with `APPLE_ID` / `APPLE_TEAM_ID` / `APPLE_APP_PASSWORD` (e.g. via
  `xcrun notarytool`) before building the DMG.
- **Windows Authenticode** — sign `meridian.exe` with `signtool` using a cert in
  `WINDOWS_CERT_PFX` (base64) + `WINDOWS_CERT_PASSWORD`.

These secret names are placeholders — nothing references them yet.

## Pre-public checklist
Before flipping the repo to public:

- [ ] You've run the finished app on at least one OS and are happy with it.
- [ ] **`wails.json` contains `hriday@moosha.org`** — it becomes public with the
      repo. Keep, change, or remove it deliberately.
- [ ] Confirm the **LICENSE** (currently Apache-2.0) is the one you want.
- [ ] Add real screenshots under `docs/media/` and uncomment them in `README.md`.
- [ ] Review the README's install/privacy wording.
- [ ] Decide the first public version tag (e.g. `v0.1.0`).
