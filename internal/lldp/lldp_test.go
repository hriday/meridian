package lldp

import (
	"encoding/binary"
	"testing"
)

// lldpTLV encodes one LLDP TLV (7-bit type, 9-bit length).
func lldpTLV(typ int, v []byte) []byte {
	h := uint16(typ)<<9 | uint16(len(v))
	out := make([]byte, 2+len(v))
	binary.BigEndian.PutUint16(out, h)
	copy(out[2:], v)
	return out
}

func ethFrame(dst, src [6]byte, etherType uint16, payload []byte) []byte {
	f := make([]byte, 14+len(payload))
	copy(f[0:6], dst[:])
	copy(f[6:12], src[:])
	binary.BigEndian.PutUint16(f[12:14], etherType)
	copy(f[14:], payload)
	return f
}

func TestClassifyLLDP(t *testing.T) {
	switchMAC := [6]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	var lldpu []byte
	lldpu = append(lldpu, lldpTLV(1, append([]byte{4}, switchMAC[:]...))...)   // chassis ID subtype 4 = MAC
	lldpu = append(lldpu, lldpTLV(2, append([]byte{5}, []byte("Gi0/1")...))...) // port ID subtype 5 = ifname
	lldpu = append(lldpu, lldpTLV(3, []byte{0, 120})...)                        // TTL
	lldpu = append(lldpu, lldpTLV(4, []byte("GigabitEthernet0/1"))...)          // port desc
	lldpu = append(lldpu, lldpTLV(5, []byte("core-sw-1"))...)                   // system name
	lldpu = append(lldpu, lldpTLV(7, []byte{0x00, 0x1c, 0x00, 0x04})...)        // caps: Bridge enabled
	lldpu = append(lldpu, lldpTLV(8, []byte{5, 1, 192, 168, 1, 2})...)          // mgmt addr: len5 subtype1(IPv4)
	lldpu = append(lldpu, lldpTLV(127, []byte{0x00, 0x80, 0xc2, 0x01, 0x00, 0x64})...) // 802.1 VLAN 100
	lldpu = append(lldpu, lldpTLV(0, nil)...)                                   // end

	frame := ethFrame([6]byte{0x01, 0x80, 0xc2, 0, 0, 0x0e}, switchMAC, 0x88cc, lldpu)
	n, ok := Classify(frame)
	if !ok {
		t.Fatal("LLDP frame should classify")
	}
	if n.Protocol != "LLDP" {
		t.Errorf("protocol = %q", n.Protocol)
	}
	if n.SystemName != "core-sw-1" {
		t.Errorf("systemName = %q, want core-sw-1", n.SystemName)
	}
	if n.ChassisID != "00:11:22:33:44:55" {
		t.Errorf("chassisID = %q, want MAC", n.ChassisID)
	}
	if n.PortID != "Gi0/1" {
		t.Errorf("portID = %q, want Gi0/1", n.PortID)
	}
	if n.MgmtIP != "192.168.1.2" {
		t.Errorf("mgmtIP = %q, want 192.168.1.2", n.MgmtIP)
	}
	if n.VLAN != 100 {
		t.Errorf("vlan = %d, want 100", n.VLAN)
	}
	if n.SourceMAC != "00:11:22:33:44:55" {
		t.Errorf("sourceMAC = %q", n.SourceMAC)
	}
	hasBridge := false
	for _, c := range n.Capabilities {
		if c == "Bridge" {
			hasBridge = true
		}
	}
	if !hasBridge {
		t.Errorf("capabilities should include Bridge: %v", n.Capabilities)
	}
}

// cdpTLV encodes one CDP TLV (type(2) length(2) value), length covers the header.
func cdpTLV(typ uint16, v []byte) []byte {
	out := make([]byte, 4+len(v))
	binary.BigEndian.PutUint16(out, typ)
	binary.BigEndian.PutUint16(out[2:], uint16(4+len(v)))
	copy(out[4:], v)
	return out
}

func TestClassifyCDP(t *testing.T) {
	switchMAC := [6]byte{0x00, 0xaa, 0xbb, 0xcc, 0xdd, 0xee}
	cdp := []byte{2, 180, 0, 0} // version 2, ttl 180, checksum 0
	cdp = append(cdp, cdpTLV(0x0001, []byte("dist-sw-2.lab"))...)                   // device id
	cdp = append(cdp, cdpTLV(0x0003, []byte("FastEthernet0/24"))...)               // port id
	cdp = append(cdp, cdpTLV(0x0006, []byte("cisco WS-C2960"))...)                 // platform
	cdp = append(cdp, cdpTLV(0x0004, []byte{0, 0, 0, 0x08})...)                    // caps: Switch
	cdp = append(cdp, cdpTLV(0x000a, []byte{0, 50})...)                            // native VLAN 50

	snap := []byte{0xaa, 0xaa, 0x03, 0x00, 0x00, 0x0c, 0x20, 0x00}
	payload := append(snap, cdp...)
	// CDP uses an 802.3 length field, not an ethertype.
	frame := ethFrame([6]byte{0x01, 0x00, 0x0c, 0xcc, 0xcc, 0xcc}, switchMAC, uint16(len(payload)), payload)

	n, ok := Classify(frame)
	if !ok {
		t.Fatal("CDP frame should classify")
	}
	if n.Protocol != "CDP" {
		t.Errorf("protocol = %q", n.Protocol)
	}
	if n.SystemName != "dist-sw-2.lab" {
		t.Errorf("systemName = %q", n.SystemName)
	}
	if n.PortID != "FastEthernet0/24" {
		t.Errorf("portID = %q", n.PortID)
	}
	if n.Platform != "cisco WS-C2960" {
		t.Errorf("platform = %q", n.Platform)
	}
	if n.VLAN != 50 {
		t.Errorf("vlan = %d, want 50", n.VLAN)
	}
}

func TestClassifyIgnoresOtherFrames(t *testing.T) {
	// A plain IPv4 frame must not classify.
	frame := ethFrame([6]byte{1, 2, 3, 4, 5, 6}, [6]byte{6, 5, 4, 3, 2, 1}, 0x0800, []byte{0x45, 0, 0, 20})
	if _, ok := Classify(frame); ok {
		t.Error("an IPv4 frame should not be LLDP/CDP")
	}
	if _, ok := Classify([]byte{1, 2, 3}); ok {
		t.Error("a runt frame should not classify")
	}
}

func TestNeighborKeyDedups(t *testing.T) {
	a := Neighbor{Protocol: "LLDP", ChassisID: "x", SystemName: "sw", PortID: "p"}
	b := Neighbor{Protocol: "LLDP", ChassisID: "x", SystemName: "sw", PortID: "p", SystemDesc: "differs"}
	if a.Key() != b.Key() {
		t.Error("neighbors differing only in description should share a key")
	}
}
