#!/usr/bin/env bash
# Package the built Linux GUI (and the CLI) into a .deb whose Depends let apt
# pull the GTK3 + WebKitGTK runtime automatically — so `apt install ./*.deb`
# just works on Ubuntu, instead of the bare AppImage's missing-library errors.
#
# Usage: build-deb.sh <version> [arch]   (writes dist/meridian_<version>_<arch>.deb)
set -euo pipefail

VERSION="${1:?usage: build-deb.sh <version> [arch]}"
ARCH="${2:-amd64}"
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

# Debian versions can't start with "v"; dev builds get a sortable placeholder.
DEBVER="${VERSION#v}"
case "$DEBVER" in dev-* | "") DEBVER="0.0.0" ;; esac

BIN="$ROOT/build/bin/meridian"
CLI="$ROOT/build/bin/meridian-cli"
ICON="$ROOT/build/appicon.png"
DESKTOP="$ROOT/build/appimage/meridian.desktop"
DIST="$ROOT/dist"
PKG="$ROOT/build/meridian-deb"

[ -f "$BIN" ] || {
  echo "no binary at $BIN — run 'wails build' first" >&2
  exit 1
}

rm -rf "$PKG"
mkdir -p "$PKG/DEBIAN" "$PKG/usr/bin" \
  "$PKG/usr/share/applications" \
  "$PKG/usr/share/icons/hicolor/512x512/apps" "$DIST"

install -m 0755 "$BIN" "$PKG/usr/bin/meridian"
[ -f "$CLI" ] && install -m 0755 "$CLI" "$PKG/usr/bin/meridian-cli"
install -m 0644 "$DESKTOP" "$PKG/usr/share/applications/meridian.desktop"
install -m 0644 "$ICON" "$PKG/usr/share/icons/hicolor/512x512/apps/meridian.png"

# The alternatives cover the Ubuntu 24.04 "t64" library renames as well as the
# 22.04 names, so one .deb installs on both.
cat >"$PKG/DEBIAN/control" <<EOF
Package: meridian
Version: ${DEBVER}
Architecture: ${ARCH}
Maintainer: Hriday Balachandran <hriday@users.noreply.github.com>
Section: net
Priority: optional
Homepage: https://github.com/hriday/meridian
Depends: libc6, libgtk-3-0 | libgtk-3-0t64, libwebkit2gtk-4.1-0 | libwebkit2gtk-4.1-0t64
Description: Cross-platform network engineer's toolkit
 Meridian is a cross-platform network engineer's toolkit — lookups, routing,
 BGP, TLS/DNS, SNMP, Fibre Channel, printers and more — as a desktop app and a
 static CLI. This package installs the desktop GUI and the meridian-cli command.
EOF

OUT="$DIST/meridian_${DEBVER}_${ARCH}.deb"
dpkg-deb --root-owner-group --build "$PKG" "$OUT"
echo "$OUT"
