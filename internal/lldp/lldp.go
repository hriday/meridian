// Package lldp listens for LLDP and CDP advertisements on the local link and
// reports the neighbor they describe — typically "which switch and port am I
// plugged into". Switches announce LLDP every ~30s and CDP every ~60s, so a
// capture has to listen for a while.
//
// Frame parsing is pure and fully unit-tested. The raw L2 capture it feeds on
// (BPF on macOS, AF_PACKET on Linux) needs elevated privileges and lives in the
// capture_*.go files.
package lldp

import (
	"encoding/binary"
	"net"
	"strings"

	"github.com/hriday/meridian/internal/l2"
)

// Neighbor is one discovered device.
type Neighbor struct {
	Protocol     string   `json:"protocol"` // "LLDP" or "CDP"
	SystemName   string   `json:"systemName"`
	ChassisID    string   `json:"chassisID,omitempty"`
	PortID       string   `json:"portID"`
	PortDesc     string   `json:"portDesc,omitempty"`
	SystemDesc   string   `json:"systemDesc,omitempty"`
	Platform     string   `json:"platform,omitempty"` // CDP
	MgmtIP       string   `json:"mgmtIP,omitempty"`
	VLAN         int      `json:"vlan,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	SourceMAC    string   `json:"sourceMAC"`
}

// Key identifies a neighbor for de-duplication across repeated frames.
func (n Neighbor) Key() string {
	return n.Protocol + "|" + n.ChassisID + "|" + n.SystemName + "|" + n.PortID
}

// Classify inspects one Ethernet frame and, if it carries LLDP or CDP, returns
// the parsed neighbor.
func Classify(frame []byte) (Neighbor, bool) {
	if len(frame) < 14 {
		return Neighbor{}, false
	}
	srcMAC := net.HardwareAddr(frame[6:12]).String()
	etherType := binary.BigEndian.Uint16(frame[12:14])
	payload := frame[14:]

	switch {
	case etherType == 0x88cc: // LLDP
		n, ok := parseLLDP(payload)
		if ok {
			n.SourceMAC = srcMAC
		}
		return n, ok
	case etherType <= 1500: // 802.3 length => LLC/SNAP; check for CDP
		if len(payload) >= 8 &&
			payload[0] == 0xaa && payload[1] == 0xaa && payload[2] == 0x03 &&
			payload[3] == 0x00 && payload[4] == 0x00 && payload[5] == 0x0c &&
			binary.BigEndian.Uint16(payload[6:8]) == 0x2000 { // CDP protocol id
			n, ok := parseCDP(payload[8:])
			if ok {
				n.SourceMAC = srcMAC
			}
			return n, ok
		}
	}
	return Neighbor{}, false
}

// parseLLDP walks LLDP TLVs (7-bit type, 9-bit length).
func parseLLDP(p []byte) (Neighbor, bool) {
	n := Neighbor{Protocol: "LLDP"}
	for i := 0; i+2 <= len(p); {
		h := binary.BigEndian.Uint16(p[i:])
		typ := h >> 9
		ln := int(h & 0x1ff)
		i += 2
		if typ == 0 { // End of LLDPDU
			break
		}
		if i+ln > len(p) {
			break
		}
		v := p[i : i+ln]
		i += ln
		switch typ {
		case 1: // Chassis ID: subtype(1) + id
			if len(v) >= 2 {
				n.ChassisID = idString(v[0], v[1:])
			}
		case 2: // Port ID
			if len(v) >= 2 {
				n.PortID = idString(v[0], v[1:])
			}
		case 4:
			n.PortDesc = clean(v)
		case 5:
			n.SystemName = clean(v)
		case 6:
			n.SystemDesc = clean(v)
		case 7: // Capabilities: supported(2) enabled(2)
			if len(v) >= 4 {
				n.Capabilities = decodeCaps(binary.BigEndian.Uint16(v[2:4]))
			}
		case 8: // Management Address: addrlen(1) subtype(1) addr(addrlen-1) ...
			if len(v) >= 2 {
				al := int(v[0])
				if al >= 2 && 1+al <= len(v) {
					if ip := addrToIP(v[1], v[2:1+al]); ip != "" {
						n.MgmtIP = ip
					}
				}
			}
		case 127: // Org-specific: OUI(3) subtype(1) value
			if len(v) >= 6 && v[0] == 0x00 && v[1] == 0x80 && v[2] == 0xc2 && v[3] == 1 {
				n.VLAN = int(binary.BigEndian.Uint16(v[4:6])) // IEEE 802.1 port VLAN ID
			}
		}
	}
	if n.ChassisID == "" && n.SystemName == "" && n.PortID == "" {
		return n, false
	}
	return n, true
}

// parseCDP walks CDP TLVs after the version/ttl/checksum header.
func parseCDP(p []byte) (Neighbor, bool) {
	if len(p) < 4 {
		return Neighbor{}, false
	}
	n := Neighbor{Protocol: "CDP"}
	for i := 4; i+4 <= len(p); {
		typ := binary.BigEndian.Uint16(p[i:])
		ln := int(binary.BigEndian.Uint16(p[i+2:]))
		if ln < 4 || i+ln > len(p) {
			break
		}
		v := p[i+4 : i+ln]
		i += ln
		switch typ {
		case 0x0001:
			n.SystemName = clean(v)
		case 0x0002:
			n.MgmtIP = firstCDPAddr(v)
		case 0x0003:
			n.PortID = clean(v)
		case 0x0004:
			n.Capabilities = decodeCDPCaps(v)
		case 0x0005:
			n.SystemDesc = firstLine(clean(v))
		case 0x0006:
			n.Platform = clean(v)
		case 0x000a:
			if len(v) >= 2 {
				n.VLAN = int(binary.BigEndian.Uint16(v))
			}
		}
	}
	if n.SystemName == "" && n.PortID == "" {
		return n, false
	}
	return n, true
}

// idString renders an LLDP chassis/port ID per its subtype (4 & 3 are MACs).
func idString(subtype byte, v []byte) string {
	switch subtype {
	case 4: // MAC address
		if len(v) == 6 {
			return net.HardwareAddr(v).String()
		}
	case 5: // network address
		if ip := addrToIP(firstByte(v), rest(v)); ip != "" {
			return ip
		}
	}
	return clean(v)
}

// addrToIP maps an LLDP management-address family + bytes to an IP string.
func addrToIP(family byte, addr []byte) string {
	switch family {
	case 1: // IPv4
		if len(addr) == 4 {
			return net.IP(addr).String()
		}
	case 2: // IPv6
		if len(addr) == 16 {
			return net.IP(addr).String()
		}
	}
	return ""
}

// firstCDPAddr returns the first address in a CDP Addresses TLV. Layout:
// count(4), then per address: proto-type(1) proto-len(1) proto(n) addr-len(2) addr(m).
func firstCDPAddr(v []byte) string {
	if len(v) < 4 {
		return ""
	}
	i := 4
	if i+2 > len(v) {
		return ""
	}
	protoLen := int(v[i+1])
	i += 2 + protoLen
	if i+2 > len(v) {
		return ""
	}
	addrLen := int(binary.BigEndian.Uint16(v[i:]))
	i += 2
	if i+addrLen > len(v) {
		return ""
	}
	if addrLen == 4 || addrLen == 16 {
		return net.IP(v[i : i+addrLen]).String()
	}
	return ""
}

func decodeCaps(bits uint16) []string {
	names := []struct {
		bit  uint16
		name string
	}{
		{1 << 0, "Other"}, {1 << 1, "Repeater"}, {1 << 2, "Bridge"},
		{1 << 3, "WLAN-AP"}, {1 << 4, "Router"}, {1 << 5, "Phone"},
		{1 << 6, "DOCSIS"}, {1 << 7, "Station"},
	}
	var out []string
	for _, c := range names {
		if bits&c.bit != 0 {
			out = append(out, c.name)
		}
	}
	return out
}

func decodeCDPCaps(v []byte) []string {
	if len(v) < 4 {
		return nil
	}
	bits := binary.BigEndian.Uint32(v)
	names := []struct {
		bit  uint32
		name string
	}{
		{0x01, "Router"}, {0x02, "Transparent-Bridge"}, {0x04, "Source-Route-Bridge"},
		{0x08, "Switch"}, {0x10, "Host"}, {0x20, "IGMP"}, {0x40, "Repeater"},
	}
	var out []string
	for _, c := range names {
		if bits&c.bit != 0 {
			out = append(out, c.name)
		}
	}
	return out
}

func clean(v []byte) string { return strings.TrimRight(string(v), "\x00") }

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimRight(s[:i], "\r")
	}
	return s
}

func firstByte(v []byte) byte {
	if len(v) == 0 {
		return 0
	}
	return v[0]
}

func rest(v []byte) []byte {
	if len(v) <= 1 {
		return nil
	}
	return v[1:]
}

// ErrNeedsPrivilege is returned when the raw-capture device can't be opened
// without elevated privileges. It aliases the shared l2 sentinel so callers can
// match on either.
var ErrNeedsPrivilege = l2.ErrNeedsPrivilege
