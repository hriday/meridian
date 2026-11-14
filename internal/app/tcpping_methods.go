package app

import (
	"context"
	"net"
	"time"

	"github.com/hriday/meridian/internal/tcpping"
)

// TCPPing measures TCP connect latency and loss to host:port — a "ping" that
// works where ICMP is filtered, with no elevated privileges. count is clamped
// to a sane range; probes are 400ms apart.
func (a *App) TCPPing(host string, port, count int) (tcpping.Result, error) {
	if count < 1 {
		count = 5
	}
	if count > 50 {
		count = 50
	}
	ctx, cancel := context.WithTimeout(a.lookupCtx(), time.Duration(count)*4*time.Second+5*time.Second)
	defer cancel()
	return tcpping.Ping(ctx, tcpConnect, host, port, count, 400*time.Millisecond)
}

// tcpConnect dials addr and returns the handshake duration and the resolved
// remote address.
func tcpConnect(ctx context.Context, addr string) (time.Duration, string, error) {
	d := net.Dialer{Timeout: 3 * time.Second}
	start := time.Now()
	c, err := d.DialContext(ctx, "tcp", addr)
	dur := time.Since(start)
	if err != nil {
		return dur, "", err
	}
	remote := c.RemoteAddr().String()
	_ = c.Close()
	return dur, remote, nil
}
