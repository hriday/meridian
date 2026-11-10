package scan

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// HostResult is one swept host.
type HostResult struct {
	IP     string        `json:"ip"`
	Alive  bool          `json:"alive"`
	RTT    time.Duration `json:"rtt"`
	Method string        `json:"method"` // icmp | tcp
	Name   string        `json:"name"`   // reverse DNS, best-effort
}

// Pinger probes one host. available reports whether ICMP is usable at all
// (false when the raw/unprivileged socket can't be opened); ok reports whether
// this host replied.
type Pinger interface {
	Ping(ctx context.Context, ip netip.Addr, timeout time.Duration) (rtt time.Duration, ok, available bool)
}

// MaxSweepHosts caps a sweep so an over-broad prefix can't fan out unbounded.
const MaxSweepHosts = 65536

// Sweep enumerates the hosts in cidr and reports which are alive, trying ICMP
// (via p) first and a TCP connect to tcpPorts as a fallback. onResult, which may
// be called concurrently, fires per host; results are returned sorted by IP.
func Sweep(ctx context.Context, cidr string, p Pinger, tcpPorts []int, timeout time.Duration, concurrency int, onResult func(HostResult)) ([]HostResult, error) {
	pfx, err := netip.ParsePrefix(cidr)
	if err != nil {
		return nil, fmt.Errorf("not a valid CIDR: %w", err)
	}
	pfx = pfx.Masked()

	addrBits := 32
	if pfx.Addr().Is6() {
		addrBits = 128
	}
	if hostBits := addrBits - pfx.Bits(); hostBits > 20 || uint64(1)<<uint(hostBits) > MaxSweepHosts {
		return nil, fmt.Errorf("range too large: %s exceeds the %d-host sweep cap", cidr, MaxSweepHosts)
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
	results := make([]HostResult, 0, len(hosts))

Loop:
	for _, ip := range hosts {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			break Loop // stop launching promptly on cancel
		}
		wg.Add(1)
		go func(ip netip.Addr) {
			defer wg.Done()
			defer func() { <-sem }()
			r := probeHost(ctx, ip, p, tcpPorts, timeout)
			mu.Lock()
			results = append(results, r)
			mu.Unlock()
			if onResult != nil {
				onResult(r)
			}
		}(ip)
	}
	wg.Wait()

	sort.Slice(results, func(i, j int) bool {
		a, _ := netip.ParseAddr(results[i].IP)
		b, _ := netip.ParseAddr(results[j].IP)
		return a.Less(b)
	})
	return results, nil
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

func probeHost(ctx context.Context, ip netip.Addr, p Pinger, tcpPorts []int, timeout time.Duration) HostResult {
	r := HostResult{IP: ip.String()}
	if rtt, ok, available := p.Ping(ctx, ip, timeout); available && ok {
		r.Alive, r.RTT, r.Method = true, rtt, "icmp"
	} else {
		for _, port := range tcpPorts {
			d := net.Dialer{Timeout: timeout}
			c, err := d.DialContext(ctx, "tcp", net.JoinHostPort(ip.String(), strconv.Itoa(port)))
			if err == nil {
				c.Close()
				r.Alive, r.Method = true, "tcp"
				break
			}
		}
	}
	if r.Alive {
		// Best-effort reverse DNS, bounded so a slow PTR can't stall the worker.
		dnsCtx, cancel := context.WithTimeout(ctx, timeout)
		if names, err := net.DefaultResolver.LookupAddr(dnsCtx, ip.String()); err == nil && len(names) > 0 {
			r.Name = strings.TrimSuffix(names[0], ".")
		}
		cancel()
	}
	return r
}
