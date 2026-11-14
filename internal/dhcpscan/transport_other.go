//go:build !darwin && !linux && !windows

package dhcpscan

import (
	"fmt"
	"time"
)

const supported = false

type udpTransport struct{}

func newTransport(window time.Duration) (*udpTransport, error) {
	return nil, fmt.Errorf("DHCP discovery isn't supported on this operating system yet")
}

func (u *udpTransport) Discover(disco []byte) ([][]byte, error) {
	return nil, fmt.Errorf("unsupported")
}

func (u *udpTransport) Close() error { return nil }
