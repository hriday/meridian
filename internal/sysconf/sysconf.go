// Package sysconf best-effort-detects what the host is configured to use for
// time (NTP) and name resolution (DNS). Detection shells out to small OS tools
// or reads well-known files; the parsers here are pure and tested, while the
// exec is thin and selected by runtime.GOOS. Every entry point returns a Found
// — never a panic — with Err set and Values empty when nothing is detectable.
package sysconf

import (
	"context"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Found is a detection result: the values, the source they came from, and a
// reason string when detection failed or found nothing.
type Found struct {
	Values []string `json:"values"`
	Source string   `json:"source"`
	Err    string   `json:"err"`
}

// Resolvers detects the host's configured DNS resolver addresses.
func Resolvers() Found {
	switch runtime.GOOS {
	case "darwin":
		if out, err := run("scutil", "--dns"); err == nil {
			if v := parseScutilDNS(out); len(v) > 0 {
				return Found{Values: v, Source: "scutil --dns"}
			}
		}
		return fromResolvConf()
	case "windows":
		if out, err := run("ipconfig", "/all"); err == nil {
			if v := parseIpconfigDNS(out); len(v) > 0 {
				return Found{Values: v, Source: "ipconfig /all"}
			}
		}
		return Found{Source: "ipconfig /all", Err: "couldn't detect DNS servers"}
	default:
		return fromResolvConf()
	}
}

func fromResolvConf() Found {
	b, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		return Found{Source: "/etc/resolv.conf", Err: err.Error()}
	}
	v := parseResolvConf(string(b))
	if len(v) == 0 {
		return Found{Source: "/etc/resolv.conf", Err: "no nameservers found"}
	}
	return Found{Values: v, Source: "/etc/resolv.conf"}
}

// TimeServers detects the host's configured NTP time server(s).
func TimeServers() Found {
	switch runtime.GOOS {
	case "darwin":
		if out, err := run("systemsetup", "-getnetworktimeserver"); err == nil {
			if s := parseSystemsetupNTP(out); s != "" {
				return Found{Values: []string{s}, Source: "systemsetup"}
			}
		}
		if b, err := os.ReadFile("/etc/ntp.conf"); err == nil {
			if s := parseNtpConfServer(string(b)); s != "" {
				return Found{Values: []string{s}, Source: "/etc/ntp.conf"}
			}
		}
		return Found{Source: "systemsetup", Err: "couldn't detect a time server"}
	case "linux":
		if out, err := run("timedatectl", "show", "-p", "ServerName", "--value"); err == nil {
			if s := parseTimedatectl(out); s != "" {
				return Found{Values: []string{s}, Source: "timedatectl"}
			}
		}
		if b, err := os.ReadFile("/etc/systemd/timesyncd.conf"); err == nil {
			if s := parseNtpConfServer(string(b)); s != "" {
				return Found{Values: []string{s}, Source: "/etc/systemd/timesyncd.conf"}
			}
		}
		return Found{Source: "timedatectl", Err: "couldn't detect a time server"}
	case "windows":
		if out, err := run("w32tm", "/query", "/source"); err == nil {
			if s := parseW32tmSource(out); s != "" {
				return Found{Values: []string{s}, Source: "w32tm"}
			}
		}
		return Found{Source: "w32tm", Err: "couldn't detect a time server"}
	default:
		return Found{Err: "detection not supported on this OS"}
	}
}

func run(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).Output()
	return string(out), err
}

// --- pure parsers ---

func parseResolvConf(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if f := strings.Fields(line); len(f) >= 2 && f[0] == "nameserver" {
			out = append(out, f[1])
		}
	}
	return out
}

func parseScutilDNS(text string) []string {
	var out []string
	seen := map[string]bool{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "nameserver[") {
			continue
		}
		if i := strings.Index(line, ":"); i >= 0 {
			ip := strings.TrimSpace(line[i+1:])
			if ip != "" && !seen[ip] {
				seen[ip] = true
				out = append(out, ip)
			}
		}
	}
	return out
}

func parseSystemsetupNTP(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Network Time Server:") {
			v := strings.TrimSpace(strings.TrimPrefix(line, "Network Time Server:"))
			if v == "(null)" {
				return ""
			}
			return v
		}
	}
	return ""
}

func parseTimedatectl(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return ""
}

func parseW32tmSource(text string) string {
	return strings.TrimSpace(text)
}

func parseIpconfigDNS(text string) []string {
	var out []string
	inDNS := false
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(line, "DNS Servers") {
			inDNS = true
			if i := strings.Index(line, ":"); i >= 0 {
				if ip := strings.TrimSpace(line[i+1:]); net.ParseIP(ip) != nil {
					out = append(out, ip)
				}
			}
			continue
		}
		// A continuation line is just an IP (possibly IPv6, which contains
		// colons) — detect it by parsing, not by colon-presence. Anything
		// else (a new labelled field, a blank line) ends the DNS block.
		if inDNS {
			if net.ParseIP(trimmed) != nil {
				out = append(out, trimmed)
				continue
			}
			inDNS = false
		}
	}
	return out
}

func parseNtpConfServer(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "NTP=") {
			if v := strings.TrimSpace(strings.TrimPrefix(line, "NTP=")); v != "" {
				return strings.Fields(v)[0]
			}
		}
		if f := strings.Fields(line); len(f) >= 2 && (f[0] == "server" || f[0] == "pool") {
			return f[1]
		}
	}
	return ""
}
