//go:build windows

package netdiag

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
)

// NOTE: environment-dependent; verified manually, not by unit tests.
//
// Baseline is the PowerShell TcpClient (same TCP port as own, non-Meridian process). Network category and Defender firewall profile state are
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

func windowsBaseline(ctx context.Context, peer string, port int) (bool, bool) {
	ps, err := exec.LookPath("powershell.exe")
	if err != nil {
		return false, false
	}
	// Same-protocol, same-port TCP probe from a non-Meridian process. Refused
	// (WSAECONNREFUSED 10061) counts as reached, matching tcpReach.
	script := "$c=New-Object Net.Sockets.TcpClient;" +
		"try{$a=$c.BeginConnect('" + peer + "'," + strconv.Itoa(port) + ",$null,$null);" +
		"if($a.AsyncWaitHandle.WaitOne(1500)){try{$c.EndConnect($a);'reached'}catch{if($_.Exception.InnerException.ErrorCode -eq 10061){'reached'}else{'unreached'}}}else{'unreached'}}catch{'unreached'}"
	out, err := exec.CommandContext(ctx, ps, "-NoProfile", "-Command", script).Output()
	if ctx.Err() != nil {
		return false, false
	}
	if err != nil {
		return false, false
	}
	return strings.TrimSpace(string(out)) == "reached", true
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
