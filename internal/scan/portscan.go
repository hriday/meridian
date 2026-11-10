// Package scan provides local-network recon for networks the operator
// administers: a TCP-connect port scanner and a CIDR ping sweep. No stealth,
// evasion, or raw-socket scanning — plain connect/echo diagnostics.
package scan

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// PortResult is one scanned port's state.
type PortResult struct {
	Port    int    `json:"port"`
	State   string `json:"state"` // open | closed | filtered
	Service string `json:"service"`
}

var services = map[int]string{
	21: "ftp", 22: "ssh", 23: "telnet", 25: "smtp", 53: "dns", 80: "http",
	110: "pop3", 111: "rpcbind", 135: "msrpc", 139: "netbios", 143: "imap",
	161: "snmp", 389: "ldap", 443: "https", 445: "smb", 465: "smtps",
	587: "submission", 631: "ipp", 636: "ldaps", 993: "imaps", 995: "pop3s",
	1433: "mssql", 1521: "oracle", 2049: "nfs", 3306: "mysql", 3389: "rdp",
	5432: "postgres", 5900: "vnc", 6379: "redis", 8080: "http-alt",
	8443: "https-alt", 9200: "elasticsearch", 27017: "mongodb",
}

// CommonPorts is a preset of frequently-interesting ports.
func CommonPorts() []int {
	ports := make([]int, 0, len(services))
	for p := range services {
		ports = append(ports, p)
	}
	sort.Ints(ports)
	return ports
}

// ParsePorts parses a spec like "22,80,443,8000-8100" into a port list.
func ParsePorts(spec string) ([]int, error) {
	var out []int
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if lo, hi, ok := strings.Cut(part, "-"); ok {
			a, err1 := strconv.Atoi(strings.TrimSpace(lo))
			b, err2 := strconv.Atoi(strings.TrimSpace(hi))
			if err1 != nil || err2 != nil || a < 1 || b > 65535 || a > b {
				return nil, fmt.Errorf("invalid port range %q", part)
			}
			for p := a; p <= b; p++ {
				out = append(out, p)
			}
			continue
		}
		p, err := strconv.Atoi(part)
		if err != nil || p < 1 || p > 65535 {
			return nil, fmt.Errorf("invalid port %q", part)
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no ports in %q", spec)
	}
	return out, nil
}

// scanPort classifies one port: open (connected), closed (refused), or
// filtered (timeout / cancelled / unreachable).
func scanPort(ctx context.Context, host string, port int, timeout time.Duration) string {
	d := net.Dialer{Timeout: timeout}
	c, err := d.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err == nil {
		c.Close()
		return "open"
	}
	// ECONNREFUSED is portable (Windows maps WSAECONNREFUSED to it); the string
	// check is a fallback for odd wrappers.
	if errors.Is(err, syscall.ECONNREFUSED) || strings.Contains(err.Error(), "refused") {
		return "closed"
	}
	return "filtered"
}

// ScanPorts TCP-connect-scans host's ports with bounded concurrency, streaming
// each result and returning them sorted by port. onResult may be called
// concurrently from worker goroutines, so it must be safe for concurrent use.
func ScanPorts(ctx context.Context, host string, ports []int, timeout time.Duration, concurrency int, onResult func(PortResult)) ([]PortResult, error) {
	if concurrency < 1 {
		concurrency = 1
	}
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []PortResult

Loop:
	for _, port := range ports {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			break Loop // stop launching promptly on cancel
		}
		wg.Add(1)
		go func(port int) {
			defer wg.Done()
			defer func() { <-sem }()
			r := PortResult{Port: port, State: scanPort(ctx, host, port, timeout), Service: services[port]}
			mu.Lock()
			results = append(results, r)
			mu.Unlock()
			if onResult != nil {
				onResult(r)
			}
		}(port)
	}
	wg.Wait()
	sort.Slice(results, func(i, j int) bool { return results[i].Port < results[j].Port })
	return results, nil
}
