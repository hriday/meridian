//go:build !darwin && !linux && !windows

package wifi

import (
	"context"
	"errors"
)

// Supported reports whether Wi-Fi analysis works on this OS.
func Supported() bool { return false }

// Status is unavailable on this OS (macOS, Linux and Windows are supported).
func Status(ctx context.Context) (Info, error) {
	return Info{}, errors.New("Wi-Fi analysis isn't supported on this operating system")
}
