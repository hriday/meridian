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
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/hriday/meridian/internal/asnmap"
	"github.com/hriday/meridian/internal/dnsx"
	"github.com/hriday/meridian/internal/lookup"
	"github.com/hriday/meridian/internal/trace"
)

// Seams (overridable in tests).
var (
	NewProber       = trace.NewICMPProber
	NewTCPProber    = trace.NewTCPProber
	Query           asnmap.QueryFunc = dnsx.Query
	ResolveTarget                    = defaultResolveTarget
	ResolveHost                      = defaultResolveHost
	EnrichNetworks                   = defaultEnrichNetworks // set nil to skip PeeringDB enrichment
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
	return drive(ctx, target, dst, p, onHop)
}

// RunTCP traces target over TCP to port (IPv4), reaching through middleboxes
// that drop UDP/ICMP traceroute. It needs elevated privileges (raw sockets).
func RunTCP(ctx context.Context, target string, port int, onHop func(trace.Hop)) (trace.Result, error) {
	dst, err := ResolveTarget(ctx, target, false)
	if err != nil {
		return trace.Result{Target: target}, err
	}
	p, err := NewTCPProber("ip4", port)
	if err != nil {
		return trace.Result{Target: target}, err
	}
	defer p.Close()
	return drive(ctx, target, dst, p, onHop)
}

// drive runs the probe loop with an opened prober and annotates each hop.
func drive(ctx context.Context, target string, dst netip.Addr, p trace.Prober, onHop func(trace.Hop)) (trace.Result, error) {
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
	// Enrich the networks in the path with their PeeringDB view (best-effort).
	// Skipped on a failed/cancelled trace, whose context is already done.
	if err == nil && EnrichNetworks != nil {
		r.Networks = EnrichNetworks(ctx, uniqueNetworks(hops))
	}
	// A user-initiated Stop cancels the context; that is a clean stop, not an
	// error — return the partial result so the UI shows what was found.
	if errors.Is(err, context.Canceled) {
		err = nil
	}
	r.Verdict = trace.Verdict(r)
	return r, err
}

// uniqueNetworks lists the distinct ASes in path order as NetInfo skeletons
// (ASN + owner), ready for PeeringDB enrichment.
func uniqueNetworks(hops []trace.Hop) []trace.NetInfo {
	seen := map[string]bool{}
	var out []trace.NetInfo
	for _, h := range hops {
		if h.ASN == "" || seen[h.ASN] {
			continue
		}
		seen[h.ASN] = true
		out = append(out, trace.NetInfo{ASN: "AS" + strings.TrimPrefix(h.ASN, "AS"), Owner: h.ASOwner})
	}
	return out
}

// defaultEnrichNetworks fills each network's PeeringDB view (policy, IX/facility
// presence, IRR macro) concurrently and best-effort — a miss leaves the AS
// listed with just its number and owner.
func defaultEnrichNetworks(ctx context.Context, nets []trace.NetInfo) []trace.NetInfo {
	if len(nets) == 0 {
		return nets
	}
	cctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	cl := lookup.NewClient(&http.Client{Timeout: 5 * time.Second}, 10*time.Minute)

	var wg sync.WaitGroup
	sem := make(chan struct{}, 6)
	for i := range nets {
		i := i
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			p, err := cl.Peering(cctx, nets[i].ASN)
			if err != nil || p == nil {
				return
			}
			nets[i].Policy = p.Policy
			nets[i].Ratio = p.Ratio
			nets[i].IXCount = p.IXCount
			nets[i].FacCount = p.FacCount
			nets[i].IRRASSet = p.IRRASSet
			for _, ix := range p.IXPs {
				nets[i].IXPs = append(nets[i].IXPs, ix.Name)
				if len(nets[i].IXPs) >= 5 {
					break
				}
			}
		}()
	}
	wg.Wait()
	return nets
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
