// Package arpscan sweeps the local subnet with ARP requests and lists every
// host that answers — its IP, MAC, and (best-effort) vendor. ARP is link-local,
// so it finds devices that ignore ping and reports their real hardware address.
//
// Building and parsing the ARP frames is pure and unit-tested; the raw send/
// receive uses the shared l2 package (needs elevated privileges / BPF access).
package arpscan

import (
	"encoding/binary"
	"net"
)

const etherTypeARP = 0x0806

var broadcast = net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

// Host is one device that answered.
type Host struct {
	IP     string `json:"ip"`
	MAC    string `json:"mac"`
	Vendor string `json:"vendor,omitempty"`
}

// Result is the whole sweep.
type Result struct {
	Interface string `json:"interface"`
	Subnet    string `json:"subnet"`
	Hosts     []Host `json:"hosts"`
}

// BuildRequest assembles a complete Ethernet+ARP "who has targetIP" request
// frame broadcast from srcMAC/srcIP.
func BuildRequest(srcMAC net.HardwareAddr, srcIP, targetIP net.IP) []byte {
	f := make([]byte, 42)
	// Ethernet header
	copy(f[0:6], broadcast)
	copy(f[6:12], srcMAC)
	binary.BigEndian.PutUint16(f[12:14], etherTypeARP)
	// ARP
	binary.BigEndian.PutUint16(f[14:16], 1)      // hardware type: Ethernet
	binary.BigEndian.PutUint16(f[16:18], 0x0800) // protocol type: IPv4
	f[18] = 6                                     // hardware size
	f[19] = 4                                     // protocol size
	binary.BigEndian.PutUint16(f[20:22], 1)      // opcode: request
	copy(f[22:28], srcMAC)                        // sender hardware addr
	copy(f[28:32], srcIP.To4())                   // sender protocol addr
	// target hardware addr (32:38) stays zero
	copy(f[38:42], targetIP.To4()) // target protocol addr
	return f
}

// ParseReply returns the sender IP and MAC of an ARP reply frame.
func ParseReply(f []byte) (ip, mac string, ok bool) {
	if len(f) < 42 {
		return "", "", false
	}
	if binary.BigEndian.Uint16(f[12:14]) != etherTypeARP {
		return "", "", false
	}
	if binary.BigEndian.Uint16(f[20:22]) != 2 { // opcode: reply
		return "", "", false
	}
	sha := net.HardwareAddr(append([]byte(nil), f[22:28]...))
	spa := net.IP(append([]byte(nil), f[28:32]...))
	return spa.String(), sha.String(), true
}

// hostsIn returns the usable host IPs in ipnet (excluding network and broadcast
// for IPv4 prefixes shorter than /31), capped so a wide mask can't explode.
func hostsIn(ipnet *net.IPNet, cap int) []net.IP {
	ip4 := ipnet.IP.To4()
	if ip4 == nil {
		return nil
	}
	mask := binary.BigEndian.Uint32(ipnet.Mask)
	base := binary.BigEndian.Uint32(ip4) & mask
	size := ^mask // host-part size - 1 for a normal prefix

	var first, last uint32
	if size <= 1 { // /31 or /32: every address is usable
		first, last = base, base+size
	} else {
		first, last = base+1, base+size-1 // skip network and broadcast
	}

	var out []net.IP
	for h := first; h <= last && h >= first; h++ {
		var b [4]byte
		binary.BigEndian.PutUint32(b[:], h)
		out = append(out, net.IP(b[:]).To4())
		if len(out) >= cap {
			break
		}
	}
	return out
}
