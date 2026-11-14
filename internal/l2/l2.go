// Package l2 is a thin raw link-layer socket: open a capture bound to an
// interface, read Ethernet frames, and write them. It wraps BPF on macOS and
// AF_PACKET on Linux so the LLDP/CDP and ARP tools share one home for the
// platform-specific syscall work. It needs elevated privileges (or BPF access
// via ChmodBPF on macOS).
package l2

import "errors"

// ErrNeedsPrivilege is returned by Open when the raw device can't be opened for
// lack of privileges.
var ErrNeedsPrivilege = errors.New("raw packet access needs elevated privileges (or ChmodBPF on macOS) — run with sudo, or launch Meridian elevated")

// ErrReadTimeout is returned by Read when its poll interval elapsed with no
// frames. It is not a real error — the caller loops.
var ErrReadTimeout = errors.New("read timeout")

// Supported reports whether raw L2 access is implemented on this OS.
func Supported() bool { return supported }
