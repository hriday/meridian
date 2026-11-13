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
# Bundle the command-line tool beside the GUI binary (if it was built), so the
# app's "Install command-line tool" action can copy it onto the user's PATH.
CLI_BIN="$ROOT/build/bin/meridian-cli"
[ -f "$CLI_BIN" ] && cp "$CLI_BIN" "$APPDIR/usr/bin/meridian-cli"
cp "$ICON" "$APPDIR/meridian.png"
cp "$DESKTOP" "$APPDIR/meridian.desktop"
ln -sf usr/bin/meridian "$APPDIR/AppRun"

# appimagetool (pinned release + checksum); extract-and-run avoids needing FUSE on CI.
TOOL="$ROOT/build/appimagetool.AppImage"
TOOL_SHA256="46fdd785094c7f6e545b61afcfb0f3d98d8eab243f644b4b17698c01d06083d1"
if [ ! -x "$TOOL" ]; then
  curl -fsSL -o "$TOOL" \
    "https://github.com/AppImage/appimagetool/releases/download/1.9.0/appimagetool-x86_64.AppImage"
  echo "${TOOL_SHA256}  ${TOOL}" | sha256sum -c - || {
    echo "appimagetool checksum mismatch — refusing to run" >&2; exit 1; }
  chmod +x "$TOOL"
fi

APPIMAGE_EXTRACT_AND_RUN=1 ARCH=x86_64 "$TOOL" "$APPDIR" "$OUT"
echo "$OUT"
