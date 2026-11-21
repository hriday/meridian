//go:build windows

package peripherals

import (
	"context"
	"fmt"
	"os/exec"
)

// enumerate reads USB devices on Windows via PowerShell's CIM provider. The
// query is read-only. ConvertTo-Json emits a bare object (not an array) when a
// single device matches; parsePnP handles both.
func enumerate(ctx context.Context) ([]rawDevice, string, error) {
	const ps = `Get-CimInstance Win32_PnPEntity | Where-Object { $_.PNPDeviceID -like 'USB*' } | Select-Object Name,Manufacturer,PNPClass,PNPDeviceID | ConvertTo-Json -Depth 3`
	out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", ps).Output()
	if err != nil {
		return nil, "", fmt.Errorf("peripherals: couldn't read the USB device list (Get-CimInstance): %w", err)
	}
	devs, err := parsePnP(out)
	if err != nil {
		return nil, "", fmt.Errorf("peripherals: couldn't parse the USB device list: %w", err)
	}
	return devs, "", nil
}
