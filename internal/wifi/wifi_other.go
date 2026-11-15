//go:build !darwin

package wifi

import (
	"context"
	"errors"
)

// Supported reports whether Wi-Fi analysis works on this OS.
func Supported() bool { return false }

// Status is unavailable off macOS for now (Linux via iw/nl80211 is a follow-up).
func Status(ctx context.Context) (Info, error) {
	return Info{}, errors.New("Wi-Fi analysis isn't supported on this operating system yet (macOS only)")
}
