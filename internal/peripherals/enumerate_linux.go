//go:build linux

package peripherals

import (
	"context"
	"fmt"
	"os"
)

const sysfsUSB = "/sys/bus/usb/devices"

// enumerate reads USB devices on Linux from sysfs. Read-only file reads; no
// device is opened or probed.
func enumerate(ctx context.Context) ([]rawDevice, string, error) {
	if _, err := os.Stat(sysfsUSB); err != nil {
		return nil, "", fmt.Errorf("peripherals: %s isn't available on this system: %w", sysfsUSB, err)
	}
	devs, err := parseSysfs(os.DirFS(sysfsUSB))
	if err != nil {
		return nil, "", fmt.Errorf("peripherals: couldn't read %s: %w", sysfsUSB, err)
	}
	return devs, "", nil
}
