// Package ptrsweep does bulk reverse-DNS (PTR) lookups across a CIDR, mapping
// which addresses in a range have hostnames — a quick way to see what's named
// (and roughly what's live) in a subnet. It complements the ping/port sweeps:
// those probe reachability, this reads what DNS says is there.
package ptrsweep

import (
	"context"
	"fmt"
	"net/netip"
	"sort"
	"strings"
	"sync"
)

// Resolver looks up the PTR names for an IP; injected so tests stay offline.
type Resolver func(ctx context.Context, ip string) ([]string, error)

// Entry is one address that resolved to at least one name.
type Entry struct {
	IP    string   `json:"ip"`
	Names []string `json:"names"`
}

// maxHosts caps a sweep so an over-broad prefix can't fan out unbounded.
const maxHosts = 65536

// Sweep resolves the PTR records of every address in cidr (concurrently) and
// returns the ones that have a name, sorted by address. onResult (if non-nil)
// fires per named host as results arrive, so callers can show progress.
func Sweep(ctx context.Context, resolver Resolver, cidr string, concurrency int, onResult func(Entry)) ([]Entry, error) {
	hosts, err := hostsIn(cidr, maxHosts)
	if err != nil {
		return nil, err
	}
	if concurrency < 1 {
		concurrency = 1
	}

	var (
		mu      sync.Mutex
		entries []Entry
		wg      sync.WaitGroup
	)
	sem := make(chan struct{}, concurrency)
	for _, ip := range hosts {
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(ip string) {
			defer wg.Done()
			defer func() { <-sem }()
			names, err := resolver(ctx, ip)
			if err != nil || len(names) == 0 {
				return
			}
			e := Entry{IP: ip, Names: trimNames(names)}
			mu.Lock()
			entries = append(entries, e)
			mu.Unlock()
			if onResult != nil {
				onResult(e)
			}
		}(ip)
	}
	wg.Wait()

	sort.Slice(entries, func(i, j int) bool {
		a, _ := netip.ParseAddr(entries[i].IP)
		b, _ := netip.ParseAddr(entries[j].IP)
		return a.Less(b)
	})
	return entries, nil
}

// trimNames strips the trailing dot DNS PTR records carry.
func trimNames(names []string) []string {
	out := make([]string, 0, len(names))
	for _, n := range names {
		out = append(out, strings.TrimSuffix(strings.TrimSpace(n), "."))
	}
	return out
}

// hostsIn enumerates every address in cidr (network and broadcast included),
// erroring if the range exceeds max.
func hostsIn(cidr string, max int) ([]string, error) {
	p, err := netip.ParsePrefix(strings.TrimSpace(cidr))
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR %q (try 10.0.0.0/24)", cidr)
	}
	p = p.Masked()
	var out []string
	for addr := p.Addr(); p.Contains(addr); addr = addr.Next() {
		out = append(out, addr.String())
		if len(out) > max {
			return nil, fmt.Errorf("that range has more than %d addresses — narrow it to a /16 or smaller", max)
		}
		if !addr.Next().IsValid() {
			break
		}
	}
	return out, nil
}
