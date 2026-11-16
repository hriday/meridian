# Homebrew Cask for Meridian (macOS). The `version` and `sha256` below are
# bumped automatically by the Release workflow on each tagged release.
#
#   brew tap hriday/meridian https://github.com/hriday/meridian
#   brew install --cask meridian
cask "meridian" do
  version "1.17.0"
  sha256 "4a3aaa84a00efc2efb430bc40dbf66f8099e243ce7de683d092b247c9cfe39d5"

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

  zap trash: [
    "~/Library/Application Support/Meridian",
    "~/Library/Application Support/meridian",
    "~/Library/Caches/com.wails.meridian",
    "~/Library/HTTPStorages/com.wails.meridian",
    "~/Library/Preferences/com.wails.meridian.plist",
    "~/Library/Saved Application State/com.wails.meridian.savedState",
    "~/Library/WebKit/com.wails.meridian",
  ]

  caveats <<~EOS
    Meridian is not yet code-signed, so macOS blocks it on first launch:
      * macOS 14 and earlier: right-click the app -> Open -> Open.
      * macOS 15 (Sequoia) and later: System Settings -> Privacy & Security ->
        scroll to the "meridian was blocked" notice -> Open Anyway.
  EOS
end
