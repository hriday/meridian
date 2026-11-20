//go:build linux

package netdiag

import (
	"context"
	"os/exec"
	"strings"
)

// NOTE: environment-dependent; verified manually, not by unit tests.
//
// Linux has no per-app LAN gating, so there is no OS-exempt baseline: the
// baseline reports ran=false and the check yields CouldNotDetermine. A
// best-effort ufw/firewalld status query only enriches the detail string.
func init() {
	osBlockProvider = func(ctx context.Context, peers []string) (OSBlockState, string) {
		st, detail := osBlockCheck(ctx, peers, tcpReach, func(context.Context, string, int) (bool, bool) { return false, false })
		if fw := linuxFirewallStatus(ctx); fw != "" {
			detail += "; " + fw
		}
		return st, detail
	}
}

func linuxFirewallStatus(ctx context.Context) string {
	if p, err := exec.LookPath("ufw"); err == nil {
		if out, err := exec.CommandContext(ctx, p, "status").Output(); err == nil {
			return "ufw: " + strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
		}
	}
	if p, err := exec.LookPath("firewall-cmd"); err == nil {
		if out, err := exec.CommandContext(ctx, p, "--state").Output(); err == nil {
			return "firewalld: " + strings.TrimSpace(string(out))
		}
	}
	return ""
}
