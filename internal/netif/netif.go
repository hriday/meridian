// Package netif lists the host's network interfaces — name, MAC, MTU, flags, and
// IPv4/IPv6 addresses — from the Go standard library. It is entirely local and
// read-only: it opens no connection and sends nothing. List is the one entry
// point; the CLI and Wails controller are thin adapters over it.
package netif

import (
	"net"
	"sort"
)

// Iface is one network interface in a UI-friendly shape.
type Iface struct {
	Index    int      `json:"index"`
	Name     string   `json:"name"`
	MAC      string   `json:"mac"`
	MTU      int      `json:"mtu"`
	Up       bool     `json:"up"`
	Loopback bool     `json:"loopback"`
	Flags    []string `json:"flags"`
	IPv4     []string `json:"ipv4"`
	IPv6     []string `json:"ipv6"`
}

// Seams so tests can supply interfaces and addresses without touching the host.
var (
	listInterfaces = net.Interfaces
	interfaceAddrs = func(i net.Interface) ([]net.Addr, error) { return i.Addrs() }
)

// List returns the host's interfaces, sorted by index. An interface whose
// address lookup fails is still listed (with empty addresses) rather than
// failing the whole call.
func List() ([]Iface, error) {
	raw, err := listInterfaces()
	if err != nil {
		return nil, err
	}
	out := make([]Iface, 0, len(raw))
	for _, in := range raw {
		iface := Iface{
			Index:    in.Index,
			Name:     in.Name,
			MTU:      in.MTU,
			Up:       in.Flags&net.FlagUp != 0,
			Loopback: in.Flags&net.FlagLoopback != 0,
			Flags:    flagWords(in.Flags),
		}
		if len(in.HardwareAddr) > 0 {
			iface.MAC = in.HardwareAddr.String()
		}
		if addrs, aerr := interfaceAddrs(in); aerr == nil {
			for _, a := range addrs {
				ipnet, ok := a.(*net.IPNet)
				if !ok {
					continue
				}
				if ipnet.IP.To4() != nil {
					iface.IPv4 = append(iface.IPv4, ipnet.String())
				} else {
					iface.IPv6 = append(iface.IPv6, ipnet.String())
				}
			}
		}
		out = append(out, iface)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Index < out[j].Index })
	return out, nil
}

// flagWords renders the net.Flags bits as plain lowercase words.
func flagWords(f net.Flags) []string {
	var w []string
	for _, fl := range []struct {
		bit   net.Flags
		label string
	}{
		{net.FlagUp, "up"},
		{net.FlagBroadcast, "broadcast"},
		{net.FlagLoopback, "loopback"},
		{net.FlagPointToPoint, "point-to-point"},
		{net.FlagMulticast, "multicast"},
		{net.FlagRunning, "running"},
	} {
		if f&fl.bit != 0 {
			w = append(w, fl.label)
		}
	}
	return w
}
