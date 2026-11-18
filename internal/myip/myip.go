// Package myip answers the most basic question a network tool should: what are
// my addresses? It composes the host's active interfaces, its primary local
// IPv4, and the default gateway into one small, local, read-only snapshot. The
// public IP and NAT type are deliberately left to the STUN path — they need the
// network and can be slow, so a caller fetches them separately and the local
// facts render instantly.
package myip

import (
	"net"
	"strings"

	"github.com/hriday/meridian/internal/netif"
	"github.com/hriday/meridian/internal/netroute"
)

// InterfaceIP is one active interface's name and its IPv4 addresses (no mask).
type InterfaceIP struct {
	Name string   `json:"name"`
	IPv4 []string `json:"ipv4"`
}

// Info is the local network identity snapshot.
type Info struct {
	PrimaryIP  string        `json:"primaryIP"`         // the source IP for the default route
	PrimaryIf  string        `json:"primaryIf"`         // the interface that owns PrimaryIP
	Gateway    string        `json:"gateway,omitempty"` // default gateway, "" if unknown
	Interfaces []InterfaceIP `json:"interfaces"`        // active, non-loopback, with an IPv4
}

// Local gathers the snapshot. It never fails on a missing gateway or primary
// IP — those are simply left blank.
func Local() (Info, error) {
	ifaces, err := netif.List()
	if err != nil {
		return Info{}, err
	}
	info := Info{Gateway: netroute.DefaultGateway()}
	for _, f := range ifaces {
		if f.Loopback || !f.Up || len(f.IPv4) == 0 {
			continue
		}
		info.Interfaces = append(info.Interfaces, InterfaceIP{Name: f.Name, IPv4: stripMasks(f.IPv4)})
	}
	info.PrimaryIP = outboundIP()
	info.PrimaryIf = ifaceForIP(info.Interfaces, info.PrimaryIP)
	// If the outbound probe couldn't pick a source (e.g. no default route),
	// fall back to the first active interface's first address.
	if info.PrimaryIP == "" && len(info.Interfaces) > 0 && len(info.Interfaces[0].IPv4) > 0 {
		info.PrimaryIP = info.Interfaces[0].IPv4[0]
		info.PrimaryIf = info.Interfaces[0].Name
	}
	return info, nil
}

// outboundIP returns the source IPv4 the kernel would use to reach the internet.
// A UDP "connect" sends no packets — it only makes the OS select a route and a
// source address — so this leaks nothing and works offline-ish. 192.0.2.1 is
// RFC 5737 TEST-NET, guaranteed never to be a real destination.
func outboundIP() string {
	c, err := net.Dial("udp4", "192.0.2.1:9")
	if err != nil {
		return ""
	}
	defer c.Close()
	if ua, ok := c.LocalAddr().(*net.UDPAddr); ok && ua.IP != nil {
		return ua.IP.String()
	}
	return ""
}

// ifaceForIP returns the name of the interface whose IPv4 list contains ip, or "".
func ifaceForIP(ifaces []InterfaceIP, ip string) string {
	if ip == "" {
		return ""
	}
	for _, f := range ifaces {
		for _, a := range f.IPv4 {
			if a == ip {
				return f.Name
			}
		}
	}
	return ""
}

// stripMasks turns "192.168.1.82/24" into "192.168.1.82"; it leaves bare
// addresses untouched.
func stripMasks(cidrs []string) []string {
	out := make([]string, 0, len(cidrs))
	for _, c := range cidrs {
		if i := strings.IndexByte(c, '/'); i >= 0 {
			out = append(out, c[:i])
		} else {
			out = append(out, c)
		}
	}
	return out
}
