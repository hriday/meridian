//go:build darwin || linux

package trace

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"syscall"
	"time"

	"github.com/hriday/meridian/internal/l2"
	"golang.org/x/net/icmp"
	"golang.org/x/sys/unix"
)

type tcpProber struct {
	port     int
	conn     *icmp.PacketConn // raw ICMP socket: hears Time-Exceeded for our SYNs
	basePort int
}

// NewTCPProber opens the ICMP socket a TCP traceroute reads its per-hop replies
// on, then probes dst:port with TCP SYNs at increasing TTL — the SYN itself is
// sent by the kernel via a normal connect() with IP_TTL set, so no raw TCP send
// is needed. Intermediate routers answer with ICMP Time Exceeded (read here);
// the destination answers SYN-ACK or RST (seen via connect). This reaches
// through middleboxes that drop UDP/ICMP traceroute but allow TCP to the port.
// On macOS this runs unprivileged; on Linux it uses a raw ICMP socket (root).
// IPv4 only.
func NewTCPProber(network string, port int) (Prober, error) {
	if network != "ip4" {
		return nil, fmt.Errorf("TCP traceroute is IPv4-only for now")
	}
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("invalid destination port %d", port)
	}
	conn, err := icmp.ListenPacket(tcpICMPNetwork, "0.0.0.0")
	if err != nil {
		if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) {
			return nil, l2.ErrNeedsPrivilege
		}
		return nil, fmt.Errorf("couldn't open an ICMP socket for TCP traceroute: %w", err)
	}
	base := 33000 + int(time.Now().UnixNano()%10000)
	return &tcpProber{port: port, conn: conn, basePort: base}, nil
}

func (p *tcpProber) Close() error { return p.conn.Close() }

func (p *tcpProber) Probe(ctx context.Context, dst netip.Addr, ttl, seq int, timeout time.Duration) (netip.Addr, time.Duration, bool, error) {
	localPort := p.basePort + seq
	for localPort > 60000 {
		localPort -= 27000
	}

	deadline := time.Now().Add(timeout)
	if dl, ok := ctx.Deadline(); ok && dl.Before(deadline) {
		deadline = dl
	}

	dstHostPort := net.JoinHostPort(dst.String(), strconv.Itoa(p.port))
	d := net.Dialer{
		LocalAddr: &net.TCPAddr{Port: localPort},
		Control: func(_, _ string, c syscall.RawConn) error {
			var serr error
			if e := c.Control(func(fd uintptr) {
				serr = unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_TTL, ttl)
			}); e != nil {
				return e
			}
			return serr
		},
	}

	t0 := time.Now()
	connErr := make(chan error, 1)
	go func() {
		cctx, cancel := context.WithDeadline(ctx, deadline)
		defer cancel()
		c, err := d.DialContext(cctx, "tcp4", dstHostPort)
		if c != nil {
			c.Close() // we only wanted the handshake result, not a session
		}
		connErr <- err
	}()

	rb := make([]byte, 1500)
	for {
		if !time.Now().Before(deadline) {
			return netip.Addr{}, 0, false, nil // timed out -> a star
		}
		// A conclusive connect result means we reached the destination.
		select {
		case e := <-connErr:
			if e == nil || errors.Is(e, syscall.ECONNREFUSED) {
				return dst, time.Since(t0), true, nil
			}
			// Otherwise (unreachable/timeout) keep listening for a late ICMP.
		default:
		}

		rd := time.Now().Add(40 * time.Millisecond)
		if rd.After(deadline) {
			rd = deadline
		}
		_ = p.conn.SetReadDeadline(rd)
		n, peer, err := p.conn.ReadFrom(rb)
		if err != nil {
			continue // read deadline hit; re-check connect + overall deadline
		}
		kind, matched := matchTCPICMP(rb[:n], localPort, p.port)
		if !matched {
			continue
		}
		from := peerAddr(peer)
		reached := kind == "dest-unreachable" && from == dst
		return from, time.Since(t0), reached, nil
	}
}

// matchTCPICMP reports whether an ICMP error quotes our TCP probe (by source
// and destination port) and what kind of error it is.
func matchTCPICMP(b []byte, wantSrc, wantDst int) (kind string, matched bool) {
	msg, err := icmp.ParseMessage(protoICMPv4, b)
	if err != nil {
		return "", false
	}
	var data []byte
	switch body := msg.Body.(type) {
	case *icmp.TimeExceeded:
		data, kind = body.Data, "time-exceeded"
	case *icmp.DstUnreach:
		data, kind = body.Data, "dest-unreachable"
	default:
		return "", false
	}
	src, dst, ok := quotedTCPPorts(data)
	return kind, ok && int(src) == wantSrc && int(dst) == wantDst
}
