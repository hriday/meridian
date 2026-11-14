package mtu

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"time"
)

// Defaults for a discovery run.
const (
	defaultMaxPayload = 1472 // a full 1500-byte Ethernet MTU minus IP+ICMP headers
	defaultRetries    = 1
	defaultTimeout    = 2 * time.Second // absorbs first-probe cold-start latency
)

// Options controls a discovery run. Zero fields take the defaults above.
type Options struct {
	Target  string
	Max     int           // max payload bytes to try
	Retries int           // per-size retries before judging a size too big
	Timeout time.Duration // per-probe receive timeout
}

// Supported reports whether DF-based path-MTU discovery is implemented on this
// OS (macOS and Linux). Elsewhere DiscoverHost returns an explanatory error.
func Supported() bool { return supported }

// DiscoverHost resolves opt.Target to an IPv4 address, opens an unprivileged
// DF-setting ICMP prober, and binary-searches the path MTU (after self-checking
// that the OS enforces Don't-Fragment).
func DiscoverHost(ctx context.Context, opt Options) (Result, error) {
	if !supported {
		return Result{}, errors.New("path MTU discovery isn't supported on this operating system yet")
	}
	host := opt.Target
	if host == "" {
		return Result{}, errors.New("enter a host")
	}
	ip, err := resolveV4(ctx, host)
	if err != nil {
		return Result{}, err
	}

	max := opt.Max
	if max <= 0 {
		max = defaultMaxPayload
	}
	retries := opt.Retries
	if retries <= 0 {
		retries = defaultRetries
	}
	timeout := opt.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	lmtu := localMTU(ip)
	// Allow measuring up to the local link when it carries jumbo frames.
	if lmtu > 0 && lmtu-icmpV4Overhead > max {
		max = lmtu - icmpV4Overhead
	}

	p, err := newProber(ip, timeout)
	if err != nil {
		return Result{}, err
	}
	defer p.Close()

	return Discover(ctx, p, host, ip.String(), lmtu, max, retries)
}

// resolveV4 returns the first IPv4 address for host (host may already be a
// literal IP).
func resolveV4(ctx context.Context, host string) (netip.Addr, error) {
	if ip, err := netip.ParseAddr(host); err == nil {
		if ip.Is4() {
			return ip, nil
		}
		return netip.Addr{}, errors.New("path MTU discovery currently supports IPv4 only")
	}
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip4", host)
	if err != nil || len(ips) == 0 {
		return netip.Addr{}, errors.New("couldn't resolve " + host + " to an IPv4 address")
	}
	addr, ok := netip.AddrFromSlice(ips[0].To4())
	if !ok {
		return netip.Addr{}, errors.New("couldn't resolve " + host + " to an IPv4 address")
	}
	return addr, nil
}

// localMTU returns the MTU of the interface that routes to dst, or 0 if it can't
// be determined. The UDP "dial" selects a route without sending packets.
func localMTU(dst netip.Addr) int {
	c, err := net.Dial("udp", net.JoinHostPort(dst.String(), "9"))
	if err != nil {
		return 0
	}
	defer c.Close()
	la, ok := c.LocalAddr().(*net.UDPAddr)
	if !ok {
		return 0
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return 0
	}
	for _, ifc := range ifaces {
		addrs, _ := ifc.Addrs()
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok && ipn.IP.Equal(la.IP) {
				return ifc.MTU
			}
		}
	}
	return 0
}
