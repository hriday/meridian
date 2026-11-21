//go:build !darwin && !linux && !windows

package peripherals

import "context"

// enumerate is the fallback for platforms Meridian doesn't enumerate USB on. It
// returns no devices and an honest note rather than an empty "nothing attached".
func enumerate(_ context.Context) ([]rawDevice, string, error) {
	return nil, "Peripheral enumeration isn't supported on this platform.", nil
}
