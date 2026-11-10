//go:build unix

package wol

import "syscall"

// setBroadcast enables SO_BROADCAST on the socket so a broadcast destination is
// writable.
func setBroadcast(c syscall.RawConn) error {
	var serr error
	if err := c.Control(func(fd uintptr) {
		serr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
	}); err != nil {
		return err
	}
	return serr
}
