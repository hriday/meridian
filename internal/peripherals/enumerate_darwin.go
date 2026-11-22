//go:build darwin

package peripherals

import (
	"context"
	"fmt"
	"os/exec"
)

// enumerate reads USB devices on macOS from ioreg. The device tree
// (IOUSBHostDevice) is the inventory — every attached device with its vendor,
// IDs, serial and interface classes — and the HID tree (IOHIDDevice) adds the
// precise keyboard/mouse flags the device tree can't give. Both are read-only;
// ioreg only reports state.
//
// This used to read `system_profiler SPUSBDataType`, but that command returns
// an empty device list on recent macOS (Sequoia/Tahoe) even with devices
// plugged in — which rendered as a false "0 devices". ioreg is authoritative
// and present on every macOS version.
func enumerate(ctx context.Context) ([]rawDevice, string, error) {
	out, err := exec.CommandContext(ctx, "ioreg", "-r", "-c", "IOUSBHostDevice", "-l").Output()
	if err != nil {
		return nil, "", fmt.Errorf("peripherals: couldn't read the USB device tree (ioreg): %w", err)
	}
	devs := parseIOUSBHostDevices(out)

	note := ""
	// The HID tree adds keyboard/mouse detection. If it fails, we still return
	// the inventory — just without the input-class flags — and say so, rather
	// than implying a clean bill of health.
	if ir, err := exec.CommandContext(ctx, "ioreg", "-r", "-c", "IOHIDDevice", "-l").Output(); err == nil {
		classes := parseIOHIDClasses(ir)
		for i := range devs {
			if devs[i].hasIDs {
				// Union, don't overwrite: a device already carries its interface
				// classes from the device tree (mass-storage), and this adds the
				// HID classes — the keyboard+storage flag needs both at once.
				devs[i].classes = unionClasses(devs[i].classes, classes[id32(devs[i].vendorID, devs[i].productID)])
			}
		}
	} else {
		note = "Couldn't read HID details (ioreg): this scan can't identify keyboards or other input devices, so input-related flags are missing. Treat the results as incomplete."
	}
	return devs, note, nil
}
