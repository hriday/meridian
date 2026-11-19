//go:build windows

package netdiag

import (
	"context"
	"os/exec"
	"strings"
)

// NOTE: environment-dependent; verified manually, not by unit tests.
//
// Baseline is the system ping.exe (ICMP, not subject to per-app firewall
// rules on TCP). Network category and Defender firewall profile state are
// best-effort corroboration appended to the detail string.
func init() {
	osBlockProvider = func(ctx context.Context, peers []string) (OSBlockState, string) {
		st, detail := osBlockCheck(ctx, peers, tcpReach, windowsBaseline)
		if c := windowsCorroboration(ctx); c != "" {
			detail += "; " + c
		}
		return st, detail
	}
}

func windowsBaseline(ctx context.Context, peer string, _ int) (bool, bool) {
	p, err := exec.LookPath("ping.exe")
	if err != nil {
		return false, false
	}
	err = exec.CommandContext(ctx, p, "-n", "1", "-w", "1000", peer).Run()
	if err == nil {
		return true, true
	}
	if _, isExit := err.(*exec.ExitError); isExit {
		return false, true
	}
	return false, false
}

func windowsCorroboration(ctx context.Context) string {
	ps, err := exec.LookPath("powershell.exe")
	if err != nil {
		return ""
	}
	script := "$c=(Get-NetConnectionProfile|Select -First 1).NetworkCategory;" +
		"$f=(Get-NetFirewallProfile|%{$_.Name+'='+$_.Enabled}) -join ',';" +
		"\"network=$c firewall[$f]\""
	out, err := exec.CommandContext(ctx, ps, "-NoProfile", "-Command", script).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
