# Homebrew Cask for Meridian (macOS). The `version` and `sha256` below are
# bumped automatically by the Release workflow on each tagged release.
#
#   brew tap hriday/meridian https://github.com/hriday/meridian
#   brew install --cask meridian
cask "meridian" do
  version "1.1.0"
  sha256 "ca1c9abfbae98d2ca5f5274986aaa0f4a08591ab856f4d1da95a672a1b39372b"

  url "https://github.com/hriday/meridian/releases/download/v#{version}/Meridian-v#{version}-macos-universal.dmg",
      verified: "github.com/hriday/meridian/"
  name "Meridian"
  desc "Cross-platform network engineer's toolkit"
  homepage "https://github.com/hriday/meridian"

  app "meridian.app"

  caveats <<~EOS
    Meridian is not yet code-signed, so macOS blocks it on first launch:
      * macOS 14 and earlier: right-click the app -> Open -> Open.
      * macOS 15 (Sequoia) and later: System Settings -> Privacy & Security ->
        scroll to the "meridian was blocked" notice -> Open Anyway.
  EOS

  zap trash: [
    "~/Library/Application Support/meridian",
    "~/Library/Preferences/com.meridian.app.plist",
  ]
end
