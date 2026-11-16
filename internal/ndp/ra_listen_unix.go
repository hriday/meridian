//go:build darwin || linux

package ndp

import (
	"context"
	"fmt"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv6"
)

func init() { raListenerFn = listenRA }

// allRouters is the IPv6 all-routers multicast address a Router Solicitation is
// sent to.
var allRouters = net.ParseIP("ff02::2")

// listenRA opens a raw ICMPv6 socket, solicits routers on each eligible
// interface, and parses the Router Advertisements that arrive within timeout.
func listenRA(ctx context.Context, iface string, timeout time.Duration) ([]RouterAdvert, error) {
	conn, err := icmp.ListenPacket("ip6:ipv6-icmp", "::")
	if err != nil {
		return nil, fmt.Errorf("raw ICMPv6 socket (re-run with sudo): %w", err)
	}
	defer conn.Close()

	pc := conn.IPv6PacketConn()
	// Receive only Router Advertisements.
	var filter ipv6.ICMPFilter
	filter.SetAll(true)
	filter.Accept(ipv6.ICMPTypeRouterAdvertisement)
	_ = pc.SetICMPFilter(&filter)
	_ = pc.SetControlMessage(ipv6.FlagInterface, true)
	_ = pc.SetHopLimit(255) // RFC 4861: RS leaves with hop limit 255

	ifaces := raInterfaces(iface)
	if len(ifaces) == 0 {
		return nil, fmt.Errorf("no IPv6-capable interface to solicit on")
	}
	rs := &icmp.Message{Type: ipv6.ICMPTypeRouterSolicitation, Code: 0, Body: &icmp.RawBody{Data: []byte{0, 0, 0, 0}}}
	wb, err := rs.Marshal(nil)
	if err != nil {
		return nil, err
	}
	for i := range ifaces {
		_ = pc.SetMulticastInterface(&ifaces[i])
		_, _ = conn.WriteTo(wb, &net.IPAddr{IP: allRouters, Zone: ifaces[i].Name})
	}

	deadline := time.Now().Add(timeout)
	if dl, ok := ctx.Deadline(); ok && dl.Before(deadline) {
		deadline = dl
	}
	_ = conn.SetReadDeadline(deadline)

	seen := map[string]bool{}
	var out []RouterAdvert
	buf := make([]byte, 1500)
	for time.Now().Before(deadline) {
		n, cm, src, rerr := pc.ReadFrom(buf)
		if rerr != nil {
			break // timeout or closed
		}
		msg, perr := icmp.ParseMessage(ipv6.ICMPTypeRouterAdvertisement.Protocol(), buf[:n])
		if perr != nil || msg.Type != ipv6.ICMPTypeRouterAdvertisement {
			continue
		}
		body, ok := msg.Body.(*icmp.RawBody)
		if !ok {
			continue
		}
		ifname := ""
		if cm != nil {
			if ifi, e := net.InterfaceByIndex(cm.IfIndex); e == nil {
				ifname = ifi.Name
			}
		}
		ra, e := parseRA(body.Data, srcString(src), ifname)
		if e != nil {
			continue
		}
		key := ra.Router + "|" + ifname
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, ra)
	}
	return out, nil
}

func srcString(src net.Addr) string {
	if ip, ok := src.(*net.IPAddr); ok {
		return ip.IP.String()
	}
	return src.String()
}

// raInterfaces returns the up, multicast-capable, non-loopback interfaces that
// have an IPv6 address — the ones worth soliciting on. An explicit name limits
// it to that interface.
func raInterfaces(name string) []net.Interface {
	all, _ := net.Interfaces()
	var out []net.Interface
	for _, ifi := range all {
		if name != "" && ifi.Name != name {
			continue
		}
		if ifi.Flags&net.FlagUp == 0 || ifi.Flags&net.FlagLoopback != 0 || ifi.Flags&net.FlagMulticast == 0 {
			continue
		}
		addrs, _ := ifi.Addrs()
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok && ipn.IP.To4() == nil && ipn.IP.To16() != nil {
				out = append(out, ifi)
				break
			}
		}
	}
	return out
}
