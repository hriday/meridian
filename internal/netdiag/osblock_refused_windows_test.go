//go:build windows

package netdiag

import (
	"net"
	"os"
	"testing"
)

func TestIsRefused_WSAECONNREFUSED(t *testing.T) {
	if !isRefused(&net.OpError{Err: os.NewSyscallError("connect", wsaeConnRefused)}) {
		t.Error("WSAECONNREFUSED must classify as refused")
	}
}
