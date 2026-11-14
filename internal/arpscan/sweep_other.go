//go:build !darwin && !linux && !windows

package arpscan

import (
	"context"
	"fmt"
	"net"
	"time"
)

const arpSupported = false

func sweepSubnet(ctx context.Context, iface string, srcMAC net.HardwareAddr, srcIP net.IP, ipnet *net.IPNet, window time.Duration, maxHosts int) (map[string]string, error) {
	return nil, fmt.Errorf("ARP scanning isn't supported on this operating system yet")
}
