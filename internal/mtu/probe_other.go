//go:build !darwin && !linux && !windows

package mtu

import (
	"fmt"
	"net/netip"
	"time"
)

const supported = false

// newProber is unavailable on this OS: setting the Don't-Fragment bit on an
// unprivileged ICMP socket needs platform-specific code that only macOS and
// Linux have here.
func newProber(ip netip.Addr, timeout time.Duration) (Prober, error) {
	return nil, fmt.Errorf("path MTU discovery isn't supported on this operating system yet")
}
