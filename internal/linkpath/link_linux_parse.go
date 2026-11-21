package linkpath

import (
	"io/fs"
	"strconv"
	"strings"
)

// parseLinuxLink reads an interface's speed/duplex from a /sys/class/net view
// and detects whether it's wireless. sysfs exposes <iface>/speed (Mb/s, or -1
// when unknown/down), <iface>/duplex ("full"/"half"/"unknown"), and a
// <iface>/wireless directory (or phy80211 link) on Wi-Fi interfaces. Taking an
// fs.FS keeps it pure and unit-testable. Build-tag-free.
func parseLinuxLink(fsys fs.FS, iface string) linkState {
	var ls linkState

	if dx := strings.TrimSpace(readFile(fsys, iface+"/duplex")); dx == "full" || dx == "half" {
		ls.duplex = dx
	}
	if s := strings.TrimSpace(readFile(fsys, iface+"/speed")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			ls.speedMbps = n
		}
	}
	ls.wireless = isDir(fsys, iface+"/wireless") || isDir(fsys, iface+"/phy80211")
	ls.known = ls.duplex != "" || ls.speedMbps > 0
	return ls
}

func readFile(fsys fs.FS, name string) string {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return ""
	}
	return string(b)
}

func isDir(fsys fs.FS, name string) bool {
	info, err := fs.Stat(fsys, name)
	return err == nil && info.IsDir()
}
