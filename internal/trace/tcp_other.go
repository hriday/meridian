//go:build !darwin && !linux

package trace

import "fmt"

// NewTCPProber is unavailable on this OS. A TCP traceroute must craft TTL-scoped
// SYNs and read ICMP errors from a raw socket; modern Windows blocks raw TCP
// sends, so the feature is macOS/Linux only for now.
func NewTCPProber(network string, port int) (Prober, error) {
	return nil, fmt.Errorf("TCP traceroute isn't supported on this operating system yet (macOS/Linux only)")
}
