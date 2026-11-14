//go:build darwin || linux

package mtu

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/sys/unix"
)

const supported = true

// dfProber sends DF ICMP echo probes over an unprivileged SOCK_DGRAM ICMP
// socket. The socket is opened by hand so Don't-Fragment can be set, then handed
// to Go's net poller (via FilePacketConn) for reliable, deadline-driven reads —
// the same read path the rest of the app uses. No elevated privileges needed.
//
// Replies are matched by sequence number; the kernel can rewrite the ICMP id on
// datagram sockets, so id is not reliable.
type dfProber struct {
	conn    net.PacketConn
	dst     net.Addr
	id      int
	seq     int
	timeout time.Duration
}

func newProber(ip netip.Addr, timeout time.Duration) (Prober, error) {
	if !ip.Is4() {
		return nil, fmt.Errorf("path MTU discovery currently supports IPv4 only")
	}
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, unix.IPPROTO_ICMP)
	if err != nil {
		return nil, fmt.Errorf("couldn't open an ICMP socket — ping may not be permitted (on Linux check net.ipv4.ping_group_range): %w", err)
	}
	if err := setDontFragment(fd); err != nil {
		unix.Close(fd)
		return nil, fmt.Errorf("couldn't set the Don't-Fragment bit: %w", err)
	}
	if err := unix.SetNonblock(fd, true); err != nil {
		unix.Close(fd)
		return nil, err
	}
	f := os.NewFile(uintptr(fd), "icmp-df")
	conn, err := net.FilePacketConn(f) // dups fd into a poller-managed socket
	f.Close()                          // the dup is independent; drop the original wrapper
	if err != nil {
		return nil, fmt.Errorf("couldn't wrap the ICMP socket: %w", err)
	}
	dst := &net.UDPAddr{IP: net.IP(ip.AsSlice())} // unprivileged ICMP uses UDP-style addresses
	return &dfProber{conn: conn, dst: dst, id: os.Getpid() & 0xffff, timeout: timeout}, nil
}

func (p *dfProber) Close() error { return p.conn.Close() }

func (p *dfProber) Probe(ctx context.Context, payload int) (ProbeResult, error) {
	p.seq = (p.seq + 1) & 0xffff
	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{ID: p.id, Seq: p.seq, Data: make([]byte, payload)},
	}
	b, err := msg.Marshal(nil)
	if err != nil {
		return ProbeResult{}, err
	}
	if _, err := p.conn.WriteTo(b, p.dst); err != nil {
		// The local link can't carry this un-fragmented with DF set: a
		// definitive too-big (and, for the self-check, proof DF is enforced).
		if errors.Is(err, unix.EMSGSIZE) {
			return ProbeResult{Outcome: TooBig}, nil
		}
		return ProbeResult{}, err
	}

	deadline := time.Now().Add(p.timeout)
	_ = p.conn.SetReadDeadline(deadline)
	rb := make([]byte, 1600)
	for {
		if err := ctx.Err(); err != nil {
			return ProbeResult{}, err
		}
		n, _, err := p.conn.ReadFrom(rb)
		if err != nil {
			var nerr net.Error
			if errors.As(err, &nerr) && nerr.Timeout() {
				return ProbeResult{Outcome: Unknown}, nil // no reply for this size
			}
			return ProbeResult{}, err
		}
		out, matched := classifyReply(rb[:n], p.seq)
		if matched {
			return out, nil
		}
		if time.Now().After(deadline) { // stray replies only; give up on this size
			return ProbeResult{Outcome: Unknown}, nil
		}
	}
}

// classifyReply interprets one received datagram. matched is false when the
// datagram isn't a reply to our probe (caller keeps reading). Some kernels
// prepend the IPv4 header to datagram-ICMP reads; we strip it when present.
func classifyReply(b []byte, wantSeq int) (ProbeResult, bool) {
	// An IPv4 header starts with version nibble 4; ICMP reply/error types
	// (0, 3, 11) never do, so a high nibble of 4 means the header is present.
	if len(b) > 0 && b[0]>>4 == 4 {
		ihl := int(b[0]&0x0f) * 4
		if ihl >= 20 && len(b) >= ihl {
			b = b[ihl:]
		}
	}
	m, err := icmp.ParseMessage(1 /* IPv4 ICMP */, b)
	if err != nil {
		return ProbeResult{}, false
	}
	switch body := m.Body.(type) {
	case *icmp.Echo:
		if body.Seq == wantSeq {
			return ProbeResult{Outcome: Fits}, true
		}
	case *icmp.DstUnreach:
		// Fragmentation needed (type 3, code 4): the next-hop MTU is in the low
		// 16 bits of the ICMP header's "unused" word (bytes 6-7).
		if m.Code == 4 {
			nh := 0
			if len(b) >= 8 {
				nh = int(binary.BigEndian.Uint16(b[6:8]))
			}
			return ProbeResult{Outcome: TooBig, NextHop: nh}, true
		}
	}
	return ProbeResult{}, false
}
