package netdiag

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"syscall"
	"time"
)

// reachFn reports whether peer:port is reachable. ran=false means the probe
// could not be executed at all (missing binary, unsupported OS), in which case
// ok carries no information.
type reachFn func(ctx context.Context, peer string, port int) (ok bool, ran bool)

// osBlockPort is the TCP port used for the plain-connect probe. A refused
// connection still counts as reached (see tcpReach).
const osBlockPort = 443

// osBlockProvider is the per-OS seam, set in each osblock_<os>.go init().
// It is nil on unsupported platforms. Tests replace it.
var osBlockProvider func(ctx context.Context, peers []string) (OSBlockState, string)

// OSBlockSupported reports whether this platform has an OS-block check.
func OSBlockSupported() bool { return osBlockProvider != nil }

// OSBlockCheck asks whether the OS is blocking Meridian from the LAN peers.
// It never claims OSBlockBlocked unless an OS-exempt baseline reached a peer
// that Meridian's own plain TCP connect could not.
func OSBlockCheck(ctx context.Context, peers []string) (OSBlockState, string) {
	if osBlockProvider == nil {
		return OSBlockCouldNotDetermine, "OS-block check not supported on this platform"
	}
	return osBlockProvider(ctx, peers)
}

// osBlockCheck is the pure comparison. For each peer: baseline reaches but
// own does not => Blocked. If no peer could be baselined => CouldNotDetermine.
// Otherwise NotBlocked (own reaches, or both fail so the OS is not the cause).
func osBlockCheck(ctx context.Context, peers []string, ownReach, baselineReach reachFn) (OSBlockState, string) {
	if len(peers) == 0 {
		return OSBlockCouldNotDetermine, "no peers to probe"
	}
	baselined := 0
	for _, p := range peers {
		bOK, bRan := baselineReach(ctx, p, osBlockPort)
		if !bRan {
			continue
		}
		oOK, oRan := ownReach(ctx, p, osBlockPort)
		if !oRan {
			continue
		}
		baselined++
		if bOK && !oOK {
			return OSBlockBlocked, fmt.Sprintf("OS-exempt probe reached %s but Meridian's own connection could not", p)
		}
	}
	if baselined == 0 {
		return OSBlockCouldNotDetermine, "no OS-exempt baseline probe could be run"
	}
	return OSBlockNotBlocked, "no peer was reachable by the OS baseline but not by Meridian"
}

// tcpReach is Meridian's "own" reachability: a plain unprivileged TCP connect,
// what the real LAN tools do. A refused connection (RST) means the host
// answered, so it counts as reached; timeouts and no-route do not.
func tcpReach(ctx context.Context, peer string, port int) (bool, bool) {
	cctx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()
	var d net.Dialer
	c, err := d.DialContext(cctx, "tcp", net.JoinHostPort(peer, strconv.Itoa(port)))
	if err == nil {
		c.Close()
		return true, true
	}
	if errors.Is(err, syscall.ECONNREFUSED) {
		return true, true
	}
	return false, true
}
