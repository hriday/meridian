# Homebrew Cask for Meridian (macOS). The `version` and `sha256` below are
# bumped automatically by the Release workflow on each tagged release.
#
#   brew tap hriday/meridian https://github.com/hriday/meridian
#   brew install --cask meridian
cask "meridian" do
  version "1.5.0"
  sha256 "76ef92d97fa98325df576cbb48aafaa8de21dcdbf7395db08dd31122112e8737"

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
