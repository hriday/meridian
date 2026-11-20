//go:build windows

package netdiag

import (
	"errors"
	"syscall"
)

// wsaeConnRefused is Winsock's WSAECONNREFUSED, which errors.Is does not map
// to syscall.ECONNREFUSED.
const wsaeConnRefused = syscall.Errno(10061)

func isRefusedPlatform(err error) bool { return errors.Is(err, wsaeConnRefused) }
