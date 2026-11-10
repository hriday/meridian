// Package wol builds and sends Wake-on-LAN "magic packets". MagicPacket is
// pure: it validates a MAC and returns the 102-byte payload (6 bytes of 0xFF
// then the MAC sixteen times). Send broadcasts that payload over UDP, setting
// SO_BROADCAST so a broadcast address is writable. The CLI and Wails controller
// are thin adapters over Send.
package wol

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"syscall"
)

// MagicPacket validates mac (EUI-48, in colon/hyphen/dot or bare-hex form) and
// returns the 102-byte Wake-on-LAN payload.
func MagicPacket(mac string) ([]byte, error) {
	hw, err := parseMAC(mac)
	if err != nil {
		return nil, err
	}
	pkt := make([]byte, 0, 102)
	for i := 0; i < 6; i++ {
		pkt = append(pkt, 0xFF)
	}
	for i := 0; i < 16; i++ {
		pkt = append(pkt, hw...)
	}
	return pkt, nil
}

// parseMAC accepts the common EUI-48 spellings, including bare 12-hex-digit
// form, and requires a 6-byte address.
func parseMAC(mac string) (net.HardwareAddr, error) {
	s := strings.TrimSpace(mac)
	if len(s) == 12 && isHex(s) {
		// Bare hex (aabbccddeeff): insert colons so net.ParseMAC accepts it.
		var b strings.Builder
		for i := 0; i < len(s); i += 2 {
			if i > 0 {
				b.WriteByte(':')
			}
			b.WriteString(s[i : i+2])
		}
		s = b.String()
	}
	hw, err := net.ParseMAC(s)
	if err != nil {
		return nil, err
	}
	if len(hw) != 6 {
		return nil, fmt.Errorf("need a 6-byte (EUI-48) MAC, got %d bytes", len(hw))
	}
	return hw, nil
}

// Send broadcasts a magic packet for mac to bcast:port (default
// 255.255.255.255:9) over UDP. A bad MAC returns an error before any socket is
// opened. SO_BROADCAST is set so a broadcast destination is writable.
func Send(ctx context.Context, mac, bcast string, port int) error {
	pkt, err := MagicPacket(mac)
	if err != nil {
		return err
	}
	d := net.Dialer{
		Control: func(_, _ string, c syscall.RawConn) error { return setBroadcast(c) },
	}
	conn, err := d.DialContext(ctx, "udp", target(bcast, port))
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = conn.Write(pkt)
	return err
}

// target resolves the broadcast address and port, applying the defaults
// (255.255.255.255 and 9).
func target(bcast string, port int) string {
	if strings.TrimSpace(bcast) == "" {
		bcast = "255.255.255.255"
	}
	if port <= 0 {
		port = 9
	}
	return net.JoinHostPort(bcast, strconv.Itoa(port))
}

func isHex(s string) bool {
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}
