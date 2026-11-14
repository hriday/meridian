//go:build !darwin && !linux

package lldp

import "fmt"

const supported = false

func newCapture(iface string) (capture, error) {
	return nil, fmt.Errorf("LLDP/CDP capture isn't supported on this operating system yet")
}
