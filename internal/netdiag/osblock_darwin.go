//go:build darwin

package netdiag

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
)

// NOTE: these real providers are environment-dependent (Local Network Privacy,
// installed binaries) and are verified manually, not by unit tests.
//
// Baseline is Apple's /usr/bin/nc, which is exempt from Local Network Privacy,
// probing the SAME TCP port as Meridian's own connect with the same semantics
// (refused = host answered = reached; timeout/no-route = not reached).
func init() {
	osBlockProvider = func(ctx context.Context, peers []string) (OSBlockState, string) {
		return osBlockCheck(ctx, peers, tcpReach, darwinBaseline)
	}
}

func darwinBaseline(ctx context.Context, peer string, port int) (bool, bool) {
	const nc = "/usr/bin/nc"
	if _, err := exec.LookPath(nc); err != nil {
		return false, false
	}
	out, err := exec.CommandContext(ctx, nc, "-vz", "-G1", peer, strconv.Itoa(port)).CombinedOutput()
	if ctx.Err() != nil {
		return false, false // killed probe did not really run
	}
	if err == nil {
		return true, true
	}
	if _, isExit := err.(*exec.ExitError); isExit {
		return strings.Contains(strings.ToLower(string(out)), "refused"), true
	}
	return false, false
}
