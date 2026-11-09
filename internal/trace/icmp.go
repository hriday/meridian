package trace

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	protoICMPv4 = 1
	protoICMPv6 = 58
)

type icmpProber struct {
	network string // "ip4" or "ip6"
	proto   int
	conn    *icmp.PacketConn
	id      int
}

// NewICMPProber opens an unprivileged ICMP datagram socket for the given
// network ("ip4" or "ip6"). It returns a clear error if the socket can't be
// opened (which on some systems means ICMP ping isn't permitted).
func NewICMPProber(network string) (Prober, error) {
	var netw, laddr string
	var proto int
	switch network {
	case "ip4":
		netw, laddr, proto = "udp4", "0.0.0.0", protoICMPv4
	case "ip6":
		netw, laddr, proto = "udp6", "::", protoICMPv6
	default:
		return nil, fmt.Errorf("traceroute: unsupported network %q (want ip4 or ip6)", network)
	}
	conn, err := icmp.ListenPacket(netw, laddr)
	if err != nil {
		return nil, fmt.Errorf("couldn't open an ICMP socket for %s — ping may not be permitted (on Linux check net.ipv4.ping_group_range): %w", network, err)
	}
	return &icmpProber{network: network, proto: proto, conn: conn, id: os.Getpid() & 0xffff}, nil
}

func (p *icmpProber) Close() error { return p.conn.Close() }

func (p *icmpProber) Probe(ctx context.Context, dst netip.Addr, ttl, seq int, timeout time.Duration) (netip.Addr, time.Duration, bool, error) {
	seq &= 0xffff
	if p.network == "ip4" {
		if err := p.conn.IPv4PacketConn().SetTTL(ttl); err != nil {
			return netip.Addr{}, 0, false, err
		}
	} else {
		if err := p.conn.IPv6PacketConn().SetHopLimit(ttl); err != nil {
			return netip.Addr{}, 0, false, err
		}
	}

	var typ icmp.Type = ipv4.ICMPTypeEcho
	if p.network == "ip6" {
		typ = ipv6.ICMPTypeEchoRequest
	}
	wb, err := (&icmp.Message{Type: typ, Code: 0, Body: &icmp.Echo{ID: p.id, Seq: seq, Data: []byte("meridian")}}).Marshal(nil)
	if err != nil {
		return netip.Addr{}, 0, false, err
	}

	deadline := time.Now().Add(timeout)
	if dl, ok := ctx.Deadline(); ok && dl.Before(deadline) {
		deadline = dl
	}
	t0 := time.Now()
	if _, err := p.conn.WriteTo(wb, &net.UDPAddr{IP: net.IP(dst.AsSlice())}); err != nil {
		return netip.Addr{}, 0, false, err
	}
	_ = p.conn.SetReadDeadline(deadline)

	rb := make([]byte, 1500)
	for {
		n, peer, err := p.conn.ReadFrom(rb)
		if err != nil {
			return netip.Addr{}, 0, false, nil // timeout -> this probe is a star
		}
		_, matched, perr := parseICMPReply(p.proto, rb[:n], seq)
		if perr != nil || !matched {
			if time.Now().After(deadline) {
				return netip.Addr{}, 0, false, nil
			}
			continue // a reply to some other probe; keep reading
		}
		kind, _, _ := parseICMPReply(p.proto, rb[:n], seq)
		return peerAddr(peer), time.Since(t0), kind == "echo-reply", nil
	}
}

// parseICMPReply classifies an ICMP reply and reports whether it carries our
// sequence number (directly for an echo reply, or inside the quoted packet of
// a time-exceeded / destination-unreachable error).
func parseICMPReply(proto int, b []byte, wantSeq int) (kind string, matched bool, err error) {
	msg, err := icmp.ParseMessage(proto, b)
	if err != nil {
		return "", false, err
	}
	switch body := msg.Body.(type) {
	case *icmp.Echo:
		return "echo-reply", body.Seq == wantSeq, nil
	case *icmp.TimeExceeded:
		seq, ok := innerEchoSeq(proto, body.Data)
		return "time-exceeded", ok && seq == wantSeq, nil
	case *icmp.DstUnreach:
		seq, ok := innerEchoSeq(proto, body.Data)
		return "dest-unreachable", ok && seq == wantSeq, nil
	default:
		return "", false, nil
	}
}

// innerEchoSeq pulls the sequence number out of the original Echo quoted inside
// an ICMP error message (after the quoted IP header).
func innerEchoSeq(proto int, data []byte) (int, bool) {
	var off int
	if proto == protoICMPv4 {
		if len(data) < 20 {
			return 0, false
		}
		off = int(data[0]&0x0f) * 4
	} else {
		off = 40 // fixed IPv6 header length
	}
	if off <= 0 || off > len(data) {
		return 0, false
	}
	inner, err := icmp.ParseMessage(proto, data[off:])
	if err != nil {
		return 0, false
	}
	if e, ok := inner.Body.(*icmp.Echo); ok {
		return e.Seq, true
	}
	return 0, false
}

func peerAddr(a net.Addr) netip.Addr {
	switch v := a.(type) {
	case *net.UDPAddr:
		if ip, ok := netip.AddrFromSlice(v.IP); ok {
			return ip.Unmap()
		}
	case *net.IPAddr:
		if ip, ok := netip.AddrFromSlice(v.IP); ok {
			return ip.Unmap()
		}
	}
	return netip.Addr{}
}
