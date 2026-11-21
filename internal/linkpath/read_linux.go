//go:build linux

package linkpath

import "os"

// readLink reads an interface's speed/duplex and wireless flag from sysfs.
// Read-only file reads, no privilege needed.
func readLink(iface string) linkState {
	if iface == "" {
		return linkState{}
	}
	return parseLinuxLink(os.DirFS("/sys/class/net"), iface)
}
