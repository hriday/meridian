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
func listenRA(ctx context.Context, iface string, timeout time.Duration) ([]RouterAdvert, int, error) {
	conn, err := icmp.ListenPacket("ip6:ipv6-icmp", "::")
	if err != nil {
		return nil, 0, fmt.Errorf("raw ICMPv6 socket (re-run with sudo): %w", err)
	}
	defer conn.Close()

	pc := conn.IPv6PacketConn()
	// No kernel ICMP type filter — it behaves inconsistently across OSes and can
	// silently drop the RAs we want; we filter for Router Advertisements in
	// userspace below instead.
	_ = pc.SetControlMessage(ipv6.FlagInterface, true)
	_ = pc.SetHopLimit(255) // RFC 4861: RS leaves with hop limit 255

	ifaces := raInterfaces(iface)
	if len(ifaces) == 0 {
		return nil, 0, fmt.Errorf("no IPv6-capable interface to solicit on")
	}
	// Routers answer a solicitation (and send periodic RAs) to the all-nodes
	// group ff02::1 — join it on each interface or the replies are dropped.
	allNodes := &net.IPAddr{IP: net.ParseIP("ff02::1")}
	sent := 0
	var lastSendErr error
	for i := range ifaces {
		if err := pc.JoinGroup(&ifaces[i], allNodes); err == nil {
			defer pc.LeaveGroup(&ifaces[i], allNodes)
		}
		_ = pc.SetMulticastInterface(&ifaces[i])
		if _, werr := conn.WriteTo(buildRS(), &net.IPAddr{IP: allRouters, Zone: ifaces[i].Name}); werr != nil {
			lastSendErr = werr
			continue
		}
		sent++
	}
	if sent == 0 {
		return nil, 0, fmt.Errorf("couldn't send a Router Solicitation: %v", lastSendErr)
	}

	deadline := time.Now().Add(timeout)
	if dl, ok := ctx.Deadline(); ok && dl.Before(deadline) {
		deadline = dl
	}
	_ = conn.SetReadDeadline(deadline)

	seen := map[string]bool{}
	var out []RouterAdvert
	icmpSeen := 0
	buf := make([]byte, 1500)
	for time.Now().Before(deadline) {
		n, cm, src, rerr := pc.ReadFrom(buf)
		if rerr != nil {
			if ne, ok := rerr.(net.Error); ok && ne.Timeout() {
				break // window elapsed
			}
			continue // a transient read error — keep listening until the deadline
		}
		icmpSeen++
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
	return out, icmpSeen, nil
}

// buildRS marshals a bare Router Solicitation (just the 4 reserved bytes, no
// options). We deliberately omit the Source Link-Layer Address option: if the
// kernel sources the packet from the unspecified address, RFC 4861 §4.1 forbids
// that option and a strict router drops the whole solicitation. The reply is
// caught via the ff02::1 group we joined regardless.
func buildRS() []byte {
	m := &icmp.Message{Type: ipv6.ICMPTypeRouterSolicitation, Code: 0, Body: &icmp.RawBody{Data: []byte{0, 0, 0, 0}}}
	wb, _ := m.Marshal(nil)
	return wb
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
