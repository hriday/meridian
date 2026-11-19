//go:build darwin

package netdiag

import (
	"context"
	"os/exec"
)

// NOTE: these real providers are environment-dependent (Local Network Privacy,
// installed binaries) and are verified manually, not by unit tests.
//
// Baseline is Apple's /sbin/ping, which is exempt from Local Network Privacy.
func init() {
	osBlockProvider = func(ctx context.Context, peers []string) (OSBlockState, string) {
		return osBlockCheck(ctx, peers, tcpReach, darwinBaseline)
	}
}

func darwinBaseline(ctx context.Context, peer string, _ int) (bool, bool) {
	if _, err := exec.LookPath("/sbin/ping"); err != nil {
		return false, false
	}
	err := exec.CommandContext(ctx, "/sbin/ping", "-c1", "-t1", peer).Run()
	if err == nil {
		return true, true
	}
	if _, isExit := err.(*exec.ExitError); isExit {
		return false, true // ran, no reply
	}
	return false, false
}
