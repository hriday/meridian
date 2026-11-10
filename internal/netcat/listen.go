package netcat

import (
	"context"
	"net"
	"time"
)

// Listener accepts one peer session over TCP or UDP.
type Listener struct {
	network string
	tcp     net.Listener
	udp     net.PacketConn
	addr    net.Addr
}

// Listen binds a local port per cfg (TCP or UDP).
func Listen(ctx context.Context, cfg Config) (*Listener, error) {
	if cfg.Network == "udp" {
		var lc net.ListenConfig
		pc, err := lc.ListenPacket(ctx, "udp", cfg.Addr)
		if err != nil {
			return nil, err
		}
		return &Listener{network: "udp", udp: pc, addr: pc.LocalAddr()}, nil
	}
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", cfg.Addr)
	if err != nil {
		return nil, err
	}
	return &Listener{network: "tcp", tcp: ln, addr: ln.Addr()}, nil
}

// Addr is the bound local address (useful when the port was :0).
func (l *Listener) Addr() string { return l.addr.String() }

// Accept blocks for the first peer and returns a session bound to it. For UDP,
// the first datagram's payload is delivered on the session's first read.
func (l *Listener) Accept() (*Session, error) {
	if l.network == "udp" {
		buf := make([]byte, 64*1024)
		n, peer, err := l.udp.ReadFrom(buf)
		if err != nil {
			return nil, err
		}
		first := make([]byte, n)
		copy(first, buf[:n])
		return newSession(&udpConn{pc: l.udp, peer: peer, first: first}), nil
	}
	c, err := l.tcp.Accept()
	if err != nil {
		return nil, err
	}
	return newSession(c), nil
}

// Close stops the listener, unblocking a pending Accept.
func (l *Listener) Close() error {
	if l.network == "udp" {
		return l.udp.Close()
	}
	return l.tcp.Close()
}

// udpConn adapts a UDP packet conn bound to one peer into a net.Conn, so a
// listened UDP session reads from and writes to that single peer.
type udpConn struct {
	pc    net.PacketConn
	peer  net.Addr
	first []byte // unread remainder of the first datagram
}

func (u *udpConn) Read(b []byte) (int, error) {
	if len(u.first) > 0 {
		n := copy(b, u.first)
		u.first = u.first[n:] // keep the tail for the next read (no truncation)
		return n, nil
	}
	for {
		n, addr, err := u.pc.ReadFrom(b)
		if err != nil {
			return n, err
		}
		if addr.String() == u.peer.String() {
			return n, nil
		}
		// datagram from another peer: ignore in this single-peer session
	}
}

func (u *udpConn) Write(b []byte) (int, error)        { return u.pc.WriteTo(b, u.peer) }
func (u *udpConn) Close() error                       { return u.pc.Close() }
func (u *udpConn) LocalAddr() net.Addr                { return u.pc.LocalAddr() }
func (u *udpConn) RemoteAddr() net.Addr               { return u.peer }
func (u *udpConn) SetDeadline(t time.Time) error      { return u.pc.SetDeadline(t) }
func (u *udpConn) SetReadDeadline(t time.Time) error  { return u.pc.SetReadDeadline(t) }
func (u *udpConn) SetWriteDeadline(t time.Time) error { return u.pc.SetWriteDeadline(t) }
