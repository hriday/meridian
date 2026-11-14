//go:build linux

package mtu

import "golang.org/x/sys/unix"

// setDontFragment turns on kernel Path-MTU discovery, which sets the
// Don't-Fragment bit on every outgoing packet. On Linux this is
// IP_MTU_DISCOVER = IP_PMTUDISC_DO, and it works unprivileged on the ping
// (datagram ICMP) socket.
func setDontFragment(fd int) error {
	return unix.SetsockoptInt(fd, unix.IPPROTO_IP, unix.IP_MTU_DISCOVER, unix.IP_PMTUDISC_DO)
}
