# Homebrew Cask for Meridian (macOS). The `version` and `sha256` below are
# bumped automatically by the Release workflow on each tagged release.
#
#   brew tap hriday/meridian https://github.com/hriday/meridian
#   brew install --cask meridian
cask "meridian" do
  version "1.20.0"
  sha256 "3c191a61b3ddab516dda37d77aad511d11e79aeac2b1871640ef3a326ddb1b7e"

  url "https://github.com/hriday/meridian/releases/download/v#{version}/Meridian-v#{version}-macos-universal.dmg"
  name "Meridian"
  desc "Cross-platform network engineer's toolkit"
  homepage "https://github.com/hriday/meridian"

  livecheck do
    url :url
    strategy :github_latest
  end

  depends_on :macos

  app "meridian.app"

  # The app is signed with a Developer ID and notarized by Apple, so it opens
  # normally with no first-launch workaround.

  zap trash: [
    "~/Library/Application Support/Meridian",
    "~/Library/Application Support/meridian",
    "~/Library/Caches/com.wails.meridian",
    "~/Library/HTTPStorages/com.wails.meridian",
    "~/Library/Preferences/com.wails.meridian.plist",
    "~/Library/Saved Application State/com.wails.meridian.savedState",
    "~/Library/WebKit/com.wails.meridian",
  ]
end
