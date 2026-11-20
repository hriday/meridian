package netdiag

import (
	"context"
	"errors"
	"net"
	"strings"
)

// Neighbor is one usable entry from the OS's existing IPv4 neighbor (ARP)
// table. State is normalized to "reachable" or "stale"; incomplete/failed
// entries are never returned by Neighbors.
type Neighbor struct {
	IP    string
	MAC   string
	State string
}

// neighborsRaw returns the raw text of the OS neighbor table. Each per-OS file
// sets it in init(); tests replace it. It is passive: it only reads what the
// kernel already knows and never sends packets.
var neighborsRaw func(ctx context.Context) (string, error)

// NeighborsSupported reports whether this platform can read a neighbor table.
func NeighborsSupported() bool { return neighborsRaw != nil }

// Neighbors returns the usable (reachable/stale) IPv4 neighbors.
func Neighbors(ctx context.Context) ([]Neighbor, error) {
	if neighborsRaw == nil {
		return nil, errors.New("neighbor table not supported on this platform")
	}
	out, err := neighborsRaw(ctx)
	if err != nil {
		return nil, err
	}
	return parseNeighbors(out), nil
}

// parseNeighbors understands three line shapes, detected per line:
//
//	darwin:  "? (192.168.1.1) at a4:2b:b0:1:2:3 on en0 ifscope [ethernet]"
//	linux:   "192.168.1.1  0x1  0x2  a4:2b:b0:01:02:03  *  eth0" (/proc/net/arp)
//	windows: "  192.168.1.1   a4-2b-b0-01-02-03   dynamic"
//
// Incomplete/failed rows, broadcast and multicast MACs are dropped.
func parseNeighbors(out string) []Neighbor {
	var res []Neighbor
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		var ip, mac string
		state := "reachable"
		switch {
		case (len(f) >= 4 && f[1] == "at" && strings.HasPrefix(f[0], "(")) || (len(f) >= 4 && f[2] == "at" && strings.HasPrefix(f[1], "(")):
			i := 0
			if f[1] != "at" {
				i = 1
			}
			ip = strings.Trim(f[i], "()")
			mac = f[i+2]
		case len(f) >= 4 && net.ParseIP(f[0]) != nil && strings.HasPrefix(f[2], "0x"):
			ip, mac = f[0], f[3]
			switch f[2] {
			case "0x0": // ATF_COM unset: incomplete/failed
				continue
			case "0x2": // ATF_COM only: resolved
			default: // complete but not plain 0x2 (e.g. permanent/published)
				state = "stale"
			}
		case len(f) >= 3 && net.ParseIP(f[0]) != nil:
			ip, mac = f[0], f[1]
		default:
			continue
		}
		if net.ParseIP(ip) == nil || net.ParseIP(ip).To4() == nil {
			continue
		}
		m, ok := normMAC(mac)
		if !ok {
			continue // "(incomplete)", garbage
		}
		if m[0]&1 == 1 { // multicast or broadcast
			continue
		}
		res = append(res, Neighbor{IP: ip, MAC: m.String(), State: state})
	}
	return res
}

// normMAC parses colon- or dash-separated MACs, tolerating the unpadded octets
// macOS prints (e.g. "a4:2b:b0:1:2:3"), and rejects all-zero.
func normMAC(s string) (net.HardwareAddr, bool) {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == ':' || r == '-' })
	if len(parts) != 6 {
		return nil, false
	}
	for i, p := range parts {
		if len(p) == 1 {
			parts[i] = "0" + p
		}
	}
	m, err := net.ParseMAC(strings.Join(parts, ":"))
	if err != nil {
		return nil, false
	}
	var zero bool = true
	for _, b := range m {
		if b != 0 {
			zero = false
		}
	}
	return m, !zero
}
