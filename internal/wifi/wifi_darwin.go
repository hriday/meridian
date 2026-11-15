//go:build darwin

package wifi

import (
	"context"
	"fmt"
	"os/exec"
)

// Supported reports whether Wi-Fi analysis works on this OS.
func Supported() bool { return true }

// Status reads the Wi-Fi environment from `system_profiler SPAirPortDataType`.
// No cgo, no privileges. SSIDs are redacted by macOS without Location Services;
// the RF metrics are unaffected (Info.Note flags this).
func Status(ctx context.Context) (Info, error) {
	out, err := exec.CommandContext(ctx, "system_profiler", "SPAirPortDataType").Output()
	if err != nil {
		return Info{}, fmt.Errorf("couldn't read Wi-Fi info (is Wi-Fi on?): %w", err)
	}
	info := parseAirport(string(out))
	if info.Interface == "" {
		return info, fmt.Errorf("no Wi-Fi interface found")
	}
	return info, nil
}
