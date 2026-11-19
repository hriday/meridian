# Homebrew Cask for Meridian (macOS). The `version` and `sha256` below are
# bumped automatically by the Release workflow on each tagged release.
#
#   brew tap hriday/meridian https://github.com/hriday/meridian
#   brew install --cask meridian
cask "meridian" do
  version "1.18.9"
  sha256 "63958f6e7e56f820f2721bd146c9a751a390dd78fb7a7ce224ea286c13b3df35"

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
