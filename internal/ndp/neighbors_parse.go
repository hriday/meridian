package ndp

import "strings"

// parseDarwinNeighbors parses `ndp -an` output. Columns:
//
//	Neighbor  Linklayer Address  Netif  Expire  St  Flgs  Prbs
func parseDarwinNeighbors(out string) []Neighbor {
	var ns []Neighbor
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) < 3 || strings.HasPrefix(line, "Neighbor") {
			continue
		}
		ip, iface := splitZone(f[0])
		n := Neighbor{IP: ip, Interface: f[2]}
		if iface != "" {
			n.Interface = iface
		}
		if strings.Contains(f[1], ":") { // a MAC, not "(incomplete)"
			n.MAC = f[1]
		}
		if len(f) >= 5 {
			n.State = darwinState(f[4])
		}
		if len(f) >= 6 && strings.Contains(f[5], "R") { // Flgs column: R = router
			n.Router = true
		}
		ns = append(ns, n)
	}
	return ns
}

func darwinState(s string) string {
	switch s {
	case "R":
		return "reachable"
	case "S":
		return "stale"
	case "D":
		return "delay"
	case "P":
		return "probe"
	case "I":
		return "incomplete"
	case "N":
		return "nostate"
	default:
		return strings.ToLower(s)
	}
}

// parseLinuxNeighbors parses `ip -6 neighbor show` lines:
//
//	fe80::1 dev eth0 lladdr aa:bb:cc:dd:ee:ff router REACHABLE
func parseLinuxNeighbors(out string) []Neighbor {
	states := map[string]bool{
		"REACHABLE": true, "STALE": true, "DELAY": true, "PROBE": true,
		"FAILED": true, "INCOMPLETE": true, "NOARP": true, "PERMANENT": true, "NONE": true,
	}
	var ns []Neighbor
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		n := Neighbor{IP: f[0]}
		for i := 1; i < len(f); i++ {
			switch {
			case f[i] == "dev" && i+1 < len(f):
				n.Interface = f[i+1]
				i++
			case f[i] == "lladdr" && i+1 < len(f):
				n.MAC = f[i+1]
				i++
			case f[i] == "router":
				n.Router = true
			case states[f[i]]:
				n.State = strings.ToLower(f[i])
			}
		}
		ns = append(ns, n)
	}
	return ns
}

// parseWindowsNeighbors parses `netsh interface ipv6 show neighbors`, tracking
// the current interface from the "Interface N: Name" headers.
func parseWindowsNeighbors(out string) []Neighbor {
	var ns []Neighbor
	iface := ""
	for _, line := range strings.Split(out, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "Interface ") && strings.Contains(t, ":") {
			iface = strings.TrimSpace(t[strings.Index(t, ":")+1:])
			continue
		}
		f := strings.Fields(t)
		if len(f) < 2 || strings.HasPrefix(t, "Internet Address") || strings.HasPrefix(t, "---") {
			continue
		}
		if !strings.Contains(f[0], ":") { // not an IPv6 address line
			continue
		}
		n := Neighbor{IP: f[0], Interface: iface}
		if strings.Contains(f[1], "-") { // aa-bb-cc-dd-ee-ff
			n.MAC = strings.ReplaceAll(f[1], "-", ":")
		}
		rest := strings.ToLower(strings.Join(f[2:], " "))
		n.Router = strings.Contains(rest, "(router)")
		n.State = strings.TrimSpace(strings.ReplaceAll(rest, "(router)", ""))
		ns = append(ns, n)
	}
	return ns
}

// splitZone splits "addr%zone" into the address and its zone (interface).
func splitZone(addr string) (ip, zone string) {
	if i := strings.Index(addr, "%"); i >= 0 {
		return addr[:i], addr[i+1:]
	}
	return addr, ""
}
