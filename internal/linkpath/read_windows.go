//go:build windows

package linkpath

import "os/exec"

// readLink reads an interface's speed/duplex and media type via Get-NetAdapter.
// The query is read-only. Windows adapter names differ from the Go interface
// name, so this matches on the interface's description/name best-effort.
func readLink(iface string) linkState {
	if iface == "" {
		return linkState{}
	}
	ps := `Get-NetAdapter -IncludeHidden | Where-Object { $_.Name -eq '` + psEscape(iface) +
		`' -or $_.InterfaceDescription -eq '` + psEscape(iface) +
		`' -or $_.ifAlias -eq '` + psEscape(iface) +
		`' } | Select-Object -First 1 Name,Speed,FullDuplex,PhysicalMediaType | ConvertTo-Json`
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", ps).Output()
	if err != nil {
		return linkState{}
	}
	return parseNetAdapter(out)
}

// psEscape single-quotes a value for a PowerShell literal by doubling quotes.
func psEscape(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r == '\'' {
			out = append(out, '\'')
		}
		out = append(out, r)
	}
	return string(out)
}
