package snmp

import (
	"context"
	"fmt"
	"net/netip"
	"sort"
	"sync"
	"time"
)

// SweepHost is one probed host of an SNMP sweep.
type SweepHost struct {
	IP          string `json:"ip"`
	Responded   bool   `json:"responded"`
	Name        string `json:"name"`
	Descr       string `json:"descr"`
	UptimeTicks uint32 `json:"uptimeTicks"`
}

// sweepDial is the dialer seam (overridden in tests).
var sweepDial = Dial

// maxSweepHosts caps a sweep so an over-broad prefix can't fan out unbounded.
const maxSweepHosts = 65536

// Sweep probes every host in cidr for an SNMP response with the given read
// community. onResult (which may be called concurrently) fires per probed host
// so the caller can show progress; the return value is the responders only,
// sorted by IP. A non-responding host costs one timeout; ctx cancellation stops
// launching promptly.
func Sweep(ctx context.Context, cidr, community string, timeout time.Duration, concurrency int, onResult func(SweepHost)) ([]SweepHost, error) {
	pfx, err := netip.ParsePrefix(cidr)
	if err != nil {
		return nil, fmt.Errorf("not a valid CIDR: %w", err)
	}
	pfx = pfx.Masked()

	addrBits := 32
	if pfx.Addr().Is6() {
		addrBits = 128
	}
	if hostBits := addrBits - pfx.Bits(); hostBits > 20 || uint64(1)<<uint(hostBits) > maxSweepHosts {
		return nil, fmt.Errorf("range too large: %s exceeds the %d-host sweep cap", cidr, maxSweepHosts)
	}

	hosts := enumerate(pfx)
	if concurrency < 1 {
		concurrency = 1
	}
	if timeout <= 0 {
		timeout = time.Second
	}

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var responders []SweepHost

Loop:
	for _, ip := range hosts {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			break Loop
		}
		wg.Add(1)
		go func(ip netip.Addr) {
			defer wg.Done()
			defer func() { <-sem }()
			h := probe(ctx, ip, community, timeout)
			if h.Responded {
				mu.Lock()
				responders = append(responders, h)
				mu.Unlock()
			}
			if onResult != nil {
				onResult(h)
			}
		}(ip)
	}
	wg.Wait()

	sort.Slice(responders, func(i, j int) bool {
		a, _ := netip.ParseAddr(responders[i].IP)
		b, _ := netip.ParseAddr(responders[j].IP)
		return a.Less(b)
	})
	return responders, nil
}

// enumerate lists the host addresses of pfx, skipping the network and broadcast
// addresses for IPv4 prefixes shorter than /31.
func enumerate(pfx netip.Prefix) []netip.Addr {
	var all []netip.Addr
	for addr := pfx.Addr(); pfx.Contains(addr); addr = addr.Next() {
		all = append(all, addr)
	}
	if pfx.Addr().Is4() && pfx.Bits() < 31 && len(all) >= 2 {
		all = all[1 : len(all)-1]
	}
	return all
}

// probe dials one host and runs the System reachability probe. A successful
// System marks the host a responder with its name/description/uptime.
func probe(ctx context.Context, ip netip.Addr, community string, timeout time.Duration) SweepHost {
	h := SweepHost{IP: ip.String()}
	if ctx.Err() != nil {
		return h
	}
	c, err := sweepDial(ip.String(), community, timeout)
	if err != nil {
		return h
	}
	defer c.Close()
	sys, err := System(c)
	if err != nil {
		return h
	}
	h.Responded = true
	h.Name, h.Descr, h.UptimeTicks = sys.Name, sys.Descr, sys.UptimeTicks
	return h
}
