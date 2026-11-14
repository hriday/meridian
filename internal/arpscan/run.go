package arpscan

import (
	"context"
	"encoding/binary"
	"errors"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/hriday/meridian/internal/l2"
	"github.com/hriday/meridian/internal/macvendor"
)

const (
	// DefaultWindow is how long to keep listening after the last request.
	DefaultWindow = 2 * time.Second
	// DefaultMaxHosts caps the sweep so a wide netmask can't flood the segment.
	DefaultMaxHosts = 1024
)

// Supported reports whether ARP scanning is implemented on this OS.
func Supported() bool { return l2.Supported() }

// Run sweeps the local subnet with ARP requests and returns the hosts that
// answer (IP + MAC; vendors are added separately). It needs elevated privileges
// (raw L2); without them it returns l2.ErrNeedsPrivilege.
func Run(ctx context.Context, window time.Duration, maxHosts int) (Result, error) {
	if !l2.Supported() {
		return Result{}, errors.New("ARP scanning isn't supported on this operating system yet")
	}
	if window <= 0 {
		window = DefaultWindow
	}
	if maxHosts <= 0 {
		maxHosts = DefaultMaxHosts
	}
	iface, mac, srcIP, ipnet, err := egressInfo()
	if err != nil {
		return Result{}, err
	}
	c, err := l2.Open(iface)
	if err != nil {
		return Result{}, err
	}
	defer c.Close()

	found := map[string]string{}
	var mu sync.Mutex
	rctx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for rctx.Err() == nil {
			frames, err := c.Read()
			if err != nil {
				if errors.Is(err, l2.ErrReadTimeout) {
					continue
				}
				return
			}
			for _, f := range frames {
				if ip, m, ok := ParseReply(f); ok {
					mu.Lock()
					found[ip] = m
					mu.Unlock()
				}
			}
		}
	}()

	for _, t := range hostsIn(ipnet, maxHosts) {
		if t.Equal(srcIP) {
			continue
		}
		_ = c.Write(BuildRequest(mac, srcIP, t))
		if ctx.Err() != nil {
			break
		}
		time.Sleep(time.Millisecond) // light pacing so we don't flood
	}

	select {
	case <-time.After(window):
	case <-ctx.Done():
	}
	cancel()
	<-done

	mu.Lock()
	hosts := make([]Host, 0, len(found))
	for ip, m := range found {
		hosts = append(hosts, Host{IP: ip, MAC: m})
	}
	mu.Unlock()
	sort.Slice(hosts, func(i, j int) bool { return ipLess(hosts[i].IP, hosts[j].IP) })

	return Result{Interface: iface, Subnet: ipnet.String(), Hosts: hosts}, nil
}

// LookupVendors fills in each host's Vendor from its OUI, deduped so each OUI is
// queried once. Failures leave the vendor blank. Best-effort.
func LookupVendors(ctx context.Context, d macvendor.Doer, hosts []Host) {
	cache := map[string]string{}
	for i := range hosts {
		oui := hosts[i].MAC
		if len(oui) >= 8 {
			oui = oui[:8] // first three octets
		}
		if v, ok := cache[oui]; ok {
			hosts[i].Vendor = v
			continue
		}
		v := ""
		if res, err := macvendor.Lookup(ctx, d, hosts[i].MAC); err == nil && res.Found {
			v = res.Vendor
		}
		cache[oui] = v
		hosts[i].Vendor = v
	}
}

// egressInfo returns the egress interface's name, MAC, our IPv4, and its subnet.
func egressInfo() (string, net.HardwareAddr, net.IP, *net.IPNet, error) {
	c, err := net.Dial("udp", "8.8.8.8:9")
	if err != nil {
		return "", nil, nil, nil, errors.New("couldn't determine the outgoing network interface")
	}
	defer c.Close()
	la, ok := c.LocalAddr().(*net.UDPAddr)
	if !ok {
		return "", nil, nil, nil, errors.New("couldn't determine the local address")
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", nil, nil, nil, err
	}
	for _, ifc := range ifaces {
		if len(ifc.HardwareAddr) != 6 {
			continue
		}
		addrs, _ := ifc.Addrs()
		for _, a := range addrs {
			ipn, ok := a.(*net.IPNet)
			if !ok || ipn.IP.To4() == nil {
				continue
			}
			if ipn.IP.Equal(la.IP) {
				return ifc.Name, ifc.HardwareAddr, la.IP.To4(), &net.IPNet{IP: ipn.IP.To4(), Mask: ipn.Mask}, nil
			}
		}
	}
	return "", nil, nil, nil, errors.New("couldn't find a usable IPv4 interface with a MAC")
}

func ipLess(a, b string) bool {
	ai, bi := net.ParseIP(a).To4(), net.ParseIP(b).To4()
	if ai == nil || bi == nil {
		return a < b
	}
	return binary.BigEndian.Uint32(ai) < binary.BigEndian.Uint32(bi)
}
