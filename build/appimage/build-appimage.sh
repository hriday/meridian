#!/usr/bin/env bash
# Wrap the built Linux binary (build/bin/meridian) into a portable AppImage.
# Usage: build-appimage.sh <version>   (writes dist/Meridian-<version>-linux-x86_64.AppImage)
set -euo pipefail

VERSION="${1:?usage: build-appimage.sh <version>}"
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

BIN="$ROOT/build/bin/meridian"
ICON="$ROOT/build/appicon.png"
DESKTOP="$ROOT/build/appimage/meridian.desktop"
APPDIR="$ROOT/build/Meridian.AppDir"
DIST="$ROOT/dist"
OUT="$DIST/Meridian-${VERSION}-linux-x86_64.AppImage"

[ -f "$BIN" ] || { echo "no binary at $BIN — run 'wails build' first" >&2; exit 1; }

rm -rf "$APPDIR"
mkdir -p "$APPDIR/usr/bin" "$DIST"
cp "$BIN" "$APPDIR/usr/bin/meridian"
cp "$ICON" "$APPDIR/meridian.png"
cp "$DESKTOP" "$APPDIR/meridian.desktop"
ln -sf usr/bin/meridian "$APPDIR/AppRun"

# appimagetool (pinned release); extract-and-run avoids needing FUSE on CI.
TOOL="$ROOT/build/appimagetool.AppImage"
if [ ! -x "$TOOL" ]; then
  curl -fsSL -o "$TOOL" \
    "https://github.com/AppImage/appimagetool/releases/download/1.9.0/appimagetool-x86_64.AppImage"
  chmod +x "$TOOL"
fi

APPIMAGE_EXTRACT_AND_RUN=1 ARCH=x86_64 "$TOOL" "$APPDIR" "$OUT"
echo "$OUT"
