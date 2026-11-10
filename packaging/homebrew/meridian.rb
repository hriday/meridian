# Homebrew cask for Meridian (macOS).
#
# This is the source of truth; copy it to a tap repo (e.g. hriday/homebrew-tap,
# at Casks/meridian.rb) so users can:
#
#   brew install --cask hriday/tap/meridian
#
# At each release, bump `version` and set `sha256` to the macOS universal DMG's
# checksum (`shasum -a 256 Meridian-v<ver>-macos-universal.dmg`). The cask
# staging strips the quarantine flag, so the unsigned app opens without a
# Gatekeeper prompt.
cask "meridian" do
  version "0.1.0"
  sha256 "0000000000000000000000000000000000000000000000000000000000000000" # set at release

  url "https://github.com/hriday/meridian/releases/download/v#{version}/Meridian-v#{version}-macos-universal.dmg"
  name "Meridian"
  desc "Cross-platform network info toolkit — lookups, BGP, NTP/DNS, scanning, more"
  homepage "https://github.com/hriday/meridian"

  app "meridian.app"

  zap trash: [
    "~/Library/Application Support/meridian",
  ]
end
