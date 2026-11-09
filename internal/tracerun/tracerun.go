// Package tracerun orchestrates a full AS-aware traceroute for the adapters
// (CLI and Wails): it resolves the target, opens the ICMP prober, runs the
// probe loop, and annotates each hop with its AS and reverse-DNS name as it
// streams. Every external dependency is an exported seam so tests inject fakes.
package tracerun

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/asnmap"
	"github.com/hriday/meridian/internal/dnsx"
	"github.com/hriday/meridian/internal/trace"
)

// Seams (overridable in tests).
var (
	NewProber     = trace.NewICMPProber
	Query         asnmap.QueryFunc = dnsx.Query
	ResolveTarget                  = defaultResolveTarget
	ResolveHost                    = defaultResolveHost
)

// Run traces target (IPv4 unless useV6), streaming each annotated hop to onHop
// and returning the final annotated Result.
func Run(ctx context.Context, target string, useV6 bool, onHop func(trace.Hop)) (trace.Result, error) {
	network := "ip4"
	if useV6 {
		network = "ip6"
	}
	dst, err := ResolveTarget(ctx, target, useV6)
	if err != nil {
		return trace.Result{Target: target}, err
	}
	p, err := NewProber(network)
	if err != nil {
		return trace.Result{Target: target}, err
	}
	defer p.Close()

	cache := map[string]asnmap.ASInfo{}
	annotate := func(h *trace.Hop) {
		if h.Addr == "" {
			return
		}
		info, ok := cache[h.Addr]
		if !ok {
			info, _ = asnmap.Lookup(ctx, Query, h.Addr)
			cache[h.Addr] = info
		}
		h.ASN, h.ASOwner = info.ASN, info.Owner
		h.Host = ResolveHost(ctx, h.Addr)
	}

	var hops []trace.Hop
	r, err := trace.Run(ctx, dst, p, trace.Options{}, func(h trace.Hop) {
		annotate(&h)
		hops = append(hops, h)
		if onHop != nil {
			onHop(h)
		}
	})
	r.Hops = hops
	r.Target = target
	// A user-initiated Stop cancels the context; that is a clean stop, not an
	// error — return the partial result so the UI shows what was found.
	if errors.Is(err, context.Canceled) {
		err = nil
	}
	r.Verdict = trace.Verdict(r)
	return r, err
}

func defaultResolveTarget(ctx context.Context, target string, useV6 bool) (netip.Addr, error) {
	if a, err := netip.ParseAddr(target); err == nil {
		if useV6 && !a.Is6() {
			return netip.Addr{}, fmt.Errorf("%s is not an IPv6 address", target)
		}
		if !useV6 && !a.Is4() {
			return netip.Addr{}, fmt.Errorf("%s is not an IPv4 address", target)
		}
		return a, nil
	}
	network := "ip4"
	if useV6 {
		network = "ip6"
	}
	ips, err := net.DefaultResolver.LookupNetIP(ctx, network, target)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("couldn't resolve %s: %w", target, err)
	}
	if len(ips) == 0 {
		return netip.Addr{}, fmt.Errorf("no %s address for %s", network, target)
	}
	return ips[0].Unmap(), nil
}

func defaultResolveHost(ctx context.Context, ip string) string {
	c, cancel := context.WithTimeout(ctx, 400*time.Millisecond)
	defer cancel()
	names, err := net.DefaultResolver.LookupAddr(c, ip)
	if err != nil || len(names) == 0 {
		return ""
	}
	return strings.TrimSuffix(names[0], ".")
}
