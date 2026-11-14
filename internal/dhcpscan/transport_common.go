//go:build darwin || linux || windows

package dhcpscan

import (
	"net"
	"time"
)

// udpTransport broadcasts the DISCOVER from UDP :68 and collects replies for a
// fixed window. The socket is created per-OS (newTransport); the send/collect
// loop below is shared.
type udpTransport struct {
	conn   net.PacketConn
	window time.Duration
}

func (u *udpTransport) Discover(disco []byte) ([][]byte, error) {
	dst := &net.UDPAddr{IP: net.IPv4bcast, Port: 67}
	if _, err := u.conn.WriteTo(disco, dst); err != nil {
		return nil, err
	}
	_ = u.conn.SetReadDeadline(time.Now().Add(u.window))
	var offers [][]byte
	buf := make([]byte, 1500)
	for {
		n, _, err := u.conn.ReadFrom(buf)
		if err != nil {
			break // deadline reached: done collecting
		}
		pkt := make([]byte, n)
		copy(pkt, buf[:n])
		offers = append(offers, pkt)
	}
	return offers, nil
}

func (u *udpTransport) Close() error { return u.conn.Close() }
