package stun

import (
	"encoding/binary"
	"net"
	"testing"
)

func TestBuildRequest(t *testing.T) {
	var tx [12]byte
	tx[0] = 0xAB
	req := buildRequest(tx)
	if len(req) != 20 {
		t.Fatalf("len = %d, want 20", len(req))
	}
	if binary.BigEndian.Uint16(req[0:]) != 0x0001 {
		t.Error("not a binding request")
	}
	if binary.BigEndian.Uint32(req[4:]) != magicCookie {
		t.Error("magic cookie missing")
	}
	if req[8] != 0xAB {
		t.Error("transaction id not copied")
	}
}

// buildXorResponse crafts a Binding Success with an XOR-MAPPED-ADDRESS for ip:port.
func buildXorResponse(txID [12]byte, ip net.IP, port int) []byte {
	ip = ip.To4()
	val := make([]byte, 8)
	val[0] = 0
	val[1] = 0x01 // IPv4
	binary.BigEndian.PutUint16(val[2:], uint16(port)^uint16(magicCookie>>16))
	for i := 0; i < 4; i++ {
		val[4+i] = ip[i] ^ magicBytes[i]
	}
	attr := make([]byte, 4+len(val))
	binary.BigEndian.PutUint16(attr[0:], 0x0020)
	binary.BigEndian.PutUint16(attr[2:], uint16(len(val)))
	copy(attr[4:], val)

	msg := make([]byte, 20)
	binary.BigEndian.PutUint16(msg[0:], 0x0101) // Binding Success
	binary.BigEndian.PutUint16(msg[2:], uint16(len(attr)))
	binary.BigEndian.PutUint32(msg[4:], magicCookie)
	copy(msg[8:20], txID[:])
	return append(msg, attr...)
}

func TestParseXorMappedAddress(t *testing.T) {
	var tx [12]byte
	resp := buildXorResponse(tx, net.ParseIP("203.0.113.55"), 54321)
	ip, port, err := parseResponse(tx, resp)
	if err != nil {
		t.Fatalf("parseResponse: %v", err)
	}
	if ip.String() != "203.0.113.55" {
		t.Errorf("ip = %s, want 203.0.113.55", ip)
	}
	if port != 54321 {
		t.Errorf("port = %d, want 54321", port)
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	if _, _, err := parseResponse([12]byte{}, []byte{1, 2, 3}); err == nil {
		t.Error("short input should error")
	}
	// A valid header with no address attribute.
	msg := make([]byte, 20)
	binary.BigEndian.PutUint16(msg[0:], 0x0101)
	binary.BigEndian.PutUint32(msg[4:], magicCookie)
	if _, _, err := parseResponse([12]byte{}, msg); err == nil {
		t.Error("response without a mapped address should error")
	}
}
