package snmp

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Switch Port Mapper OIDs (BRIDGE-MIB, IF-MIB, IP-MIB).
const (
	oidFdbPort    = "1.3.6.1.2.1.17.4.3.1.2" // dot1dTpFdbPort: MAC(6) -> bridge port
	oidBasePortIf = "1.3.6.1.2.1.17.1.4.1.2" // dot1dBasePortIfIndex: bridge port -> ifIndex
	oidIfName     = "1.3.6.1.2.1.31.1.1.1.1" // ifName: ifIndex -> name (oidIfDescr is the fallback, defined in ifmib.go)
	oidArpPhys    = "1.3.6.1.2.1.4.22.1.2"   // ipNetToMediaPhysAddress: ifIndex.IP -> MAC
)

// PortEntry is one learned MAC and where it lives: the physical switch port it
// was seen on, plus its IP if the device also holds the ARP entry.
type PortEntry struct {
	MAC     string `json:"mac"`
	IP      string `json:"ip"`
	Port    string `json:"port"` // interface name (ifName / ifDescr)
	IfIndex int    `json:"ifIndex"`
}

// SwitchPorts builds the MAC-to-port table from a switch's SNMP forwarding
// database, joined to interface names and the ARP table (for IPs). Only the
// forwarding-table walk is required; the rest enrich the result.
func SwitchPorts(c Client) ([]PortEntry, error) {
	fdb, err := c.Walk(oidFdbPort)
	if err != nil {
		return nil, err
	}
	base, _ := c.Walk(oidBasePortIf)
	names, _ := c.Walk(oidIfName)
	descr, _ := c.Walk(oidIfDescr)
	arp, _ := c.Walk(oidArpPhys)

	portToIf := map[int]int{}
	for _, p := range base {
		portToIf[firstIntSuffix(p.OID, oidBasePortIf)] = asInt(p.Value)
	}
	ifName := map[int]string{}
	for _, p := range names {
		if s := asString(p.Value); s != "" {
			ifName[firstIntSuffix(p.OID, oidIfName)] = s
		}
	}
	for _, p := range descr { // fill any gaps the ifName table didn't cover
		idx := firstIntSuffix(p.OID, oidIfDescr)
		if _, ok := ifName[idx]; !ok {
			if s := asString(p.Value); s != "" {
				ifName[idx] = s
			}
		}
	}
	macToIP := map[string]string{}
	for _, p := range arp {
		if mac := macFromValue(p.Value); mac != "" {
			if ip := arpIP(p.OID, oidArpPhys); ip != "" {
				macToIP[mac] = ip
			}
		}
	}

	var out []PortEntry
	seen := map[string]bool{}
	for _, p := range fdb {
		mac := macFromFdbOID(p.OID, oidFdbPort)
		if mac == "" {
			continue
		}
		idx := portToIf[asInt(p.Value)]
		name := ifName[idx]
		if name == "" {
			name = fmt.Sprintf("if%d", idx)
		}
		if key := mac + "|" + name; seen[key] {
			continue
		} else {
			seen[key] = true
		}
		out = append(out, PortEntry{MAC: mac, IP: macToIP[mac], Port: name, IfIndex: idx})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Port != out[j].Port {
			return out[i].Port < out[j].Port
		}
		return out[i].MAC < out[j].MAC
	})
	return out, nil
}

// firstIntSuffix returns the first OID component after root as an int.
func firstIntSuffix(oid, root string) int {
	s := oidSuffix(oid, root)
	if i := strings.IndexByte(s, '.'); i >= 0 {
		s = s[:i]
	}
	n, _ := strconv.Atoi(s)
	return n
}

func oidSuffix(oid, root string) string {
	if !strings.HasPrefix(oid, root+".") {
		return ""
	}
	return oid[len(root)+1:]
}

// macFromFdbOID reads the trailing 6 decimal octets of a dot1dTpFdbPort OID as a
// colon-hex MAC.
func macFromFdbOID(oid, root string) string {
	parts := strings.Split(oidSuffix(oid, root), ".")
	if len(parts) < 6 {
		return ""
	}
	return octetsToMAC(parts[len(parts)-6:])
}

// macFromValue formats a 6-byte ipNetToMediaPhysAddress value as colon-hex.
func macFromValue(v any) string {
	var b []byte
	switch x := v.(type) {
	case string:
		b = []byte(x)
	case []byte:
		b = x
	default:
		return ""
	}
	if len(b) < 6 {
		return ""
	}
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", b[0], b[1], b[2], b[3], b[4], b[5])
}

// arpIP reads the trailing 4 octets of an ipNetToMediaPhysAddress OID as an IPv4.
func arpIP(oid, root string) string {
	parts := strings.Split(oidSuffix(oid, root), ".")
	if len(parts) < 4 {
		return ""
	}
	ip := parts[len(parts)-4:]
	for _, s := range ip {
		if n, err := strconv.Atoi(s); err != nil || n < 0 || n > 255 {
			return ""
		}
	}
	return strings.Join(ip, ".")
}

func octetsToMAC(parts []string) string {
	var b [6]byte
	for i, s := range parts {
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 || n > 255 {
			return ""
		}
		b[i] = byte(n)
	}
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", b[0], b[1], b[2], b[3], b[4], b[5])
}
