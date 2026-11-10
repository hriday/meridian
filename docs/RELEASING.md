# Releasing Meridian

## Going public (first time)

The repo was developed privately; its merged PRs retain commits authored under
old personal emails, which a history rewrite of `master` alone does **not**
purge. So go public by pushing the clean local history to a **fresh** repo.

1. **Verify the local history is clean**
   ```bash
   git log --all --format='%ae%n%ce' | sort -u      # should be only the noreply address
   grep -rIn "<your-old-email-domain>" . --exclude-dir=.git   # should print nothing
   ```
2. **Create the public repo from clean history.** Either delete and recreate
   `hriday/meridian` on GitHub, or make a new empty repo, then from a clean
   checkout:
   ```bash
   git remote set-url origin git@github.com:hriday/meridian.git   # the fresh repo
   git push -u origin master
   ```
   (Optionally decide whether to keep `docs/superpowers/` — the slice-by-slice
   dev specs/plans — in the public repo, or remove it as internal scaffolding.)
3. **Flip the repo to Public** (Settings → General → Danger Zone).
4. **Cut v0.1.0** — `git tag v0.1.0 && git push origin v0.1.0`. The release
   workflow builds and publishes the macOS/Windows/Linux binaries.
5. **Publish the Homebrew cask** — from `packaging/homebrew/meridian.rb`, set
   `sha256` to the released DMG's checksum
   (`shasum -a 256 Meridian-v0.1.0-macos-universal.dmg`), and push it to a
   `hriday/homebrew-tap` repo at `Casks/meridian.rb`. Users then run
   `brew install --cask hriday/tap/meridian`.
6. **Later (optional):** Scoop/winget manifests, Flatpak, code signing.

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

- [x] Repo hygiene: `SECURITY.md`, `CONTRIBUTING.md`, `NOTICE`, Apache-2.0 LICENSE.
- [x] Screenshots under `docs/media/`, embedded in the README.
- [x] README foregrounds Homebrew + build-from-source; privacy section accurate.
- [x] Homebrew cask prepared (`packaging/homebrew/meridian.rb`).
- [ ] **No personal email anywhere** — run the two checks in *Going public* step 1.
- [ ] You've run the finished app on at least one OS and are happy with it.
- [ ] Decide whether `docs/superpowers/` ships publicly.
- [ ] Follow *Going public* to create the fresh repo, flip public, and tag `v0.1.0`.
