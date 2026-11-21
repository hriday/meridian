//go:build darwin

package peripherals

import (
	"context"
	"fmt"
	"os/exec"
)

// enumerate reads USB devices on macOS from system_profiler (the device tree)
// and ioreg (the HID classes), then merges the two: system_profiler gives every
// attached device with its vendor and IDs, ioreg says which of them present as a
// keyboard or mouse. Read-only — both commands only report state.
func enumerate(ctx context.Context) ([]rawDevice, string, error) {
	out, err := exec.CommandContext(ctx, "system_profiler", "SPUSBDataType", "-json").Output()
	if err != nil {
		return nil, "", fmt.Errorf("peripherals: couldn't read the USB device list (system_profiler): %w", err)
	}
	devs, err := parseSystemProfiler(out)
	if err != nil {
		return nil, "", fmt.Errorf("peripherals: couldn't parse the USB device list: %w", err)
	}

	note := ""
	// ioreg adds keyboard/mouse detection. If it fails, we still return the
	// inventory — just without the input-class flags — and say so, rather than
	// implying a clean bill of health.
	if ir, err := exec.CommandContext(ctx, "ioreg", "-r", "-c", "IOHIDDevice", "-l").Output(); err == nil {
		classes := parseIOHIDClasses(ir)
		for i := range devs {
			if devs[i].hasIDs {
				if cs := classes[id32(devs[i].vendorID, devs[i].productID)]; len(cs) > 0 {
					devs[i].classes = cs
				}
			}
		}
	} else {
		note = "Couldn't read HID details (ioreg) — keyboard/mouse flags may be incomplete."
	}
	return devs, note, nil
}
