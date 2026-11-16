// Package ndp explores the IPv6 link: the neighbor cache (the v6 counterpart to
// ARP — every on-link host the OS has talked to, with its MAC and reachability)
// and, by soliciting and parsing ICMPv6 Router Advertisements, what the routers
// announce: prefixes and their SLAAC/on-link flags, the managed/other-config
// flags, recursive DNS servers (RDNSS), MTU and lifetimes. Reading the neighbor
// cache is unprivileged and cross-platform; listening for Router Advertisements
// needs a raw ICMPv6 socket (root), so it is a Unix-only, elevated operation.
//
// The parsers — the neighbor-cache text per OS and the RA option TLVs — are pure
// and unit-tested; only the socket send/receive touches the network.
package ndp

import (
	"context"
	"errors"
	"time"
)

// Neighbor is one entry in the IPv6 neighbor cache.
type Neighbor struct {
	IP        string `json:"ip"`
	MAC       string `json:"mac,omitempty"`
	Interface string `json:"interface"`
	State     string `json:"state,omitempty"`  // reachable, stale, incomplete, …
	Router    bool   `json:"router,omitempty"` // the neighbor is acting as a router
}

// Prefix is one Prefix Information option from a Router Advertisement.
type Prefix struct {
	Prefix            string `json:"prefix"` // e.g. 2001:db8::/64
	OnLink            bool   `json:"onLink"`
	Autonomous        bool   `json:"autonomous"` // hosts may form SLAAC addresses from it
	ValidLifetime     uint32 `json:"validLifetime"`
	PreferredLifetime uint32 `json:"preferredLifetime"`
}

// RouterAdvert is a parsed ICMPv6 Router Advertisement.
type RouterAdvert struct {
	Router         string   `json:"router"` // source address of the RA
	Interface      string   `json:"interface,omitempty"`
	SourceMAC      string   `json:"sourceMAC,omitempty"`
	HopLimit       uint8    `json:"hopLimit"`
	Managed        bool     `json:"managed"`        // M flag — addresses via DHCPv6
	OtherConfig    bool     `json:"otherConfig"`    // O flag — other config via DHCPv6
	RouterLifetime uint16   `json:"routerLifetime"` // seconds; 0 = not a default router
	ReachableTime  uint32   `json:"reachableTime"`
	RetransTimer   uint32   `json:"retransTimer"`
	MTU            uint32   `json:"mtu,omitempty"`
	Prefixes       []Prefix `json:"prefixes,omitempty"`
	DNS            []string `json:"dns,omitempty"`       // RDNSS
	DNSSearch      []string `json:"dnsSearch,omitempty"` // DNSSL
}

// Result bundles both views of the link.
type Result struct {
	OS        string         `json:"os"`
	Neighbors []Neighbor     `json:"neighbors"`
	Routers   []RouterAdvert `json:"routers"`
	RANote    string         `json:"raNote,omitempty"` // why RA listening was skipped/failed, if it was
}

// neighborsFn reads the OS neighbor cache, set per-OS at build time.
var neighborsFn func(context.Context) ([]Neighbor, error)

// Neighbors returns the IPv6 neighbor cache (unprivileged).
func Neighbors(ctx context.Context) ([]Neighbor, error) {
	if neighborsFn == nil {
		return nil, errUnsupported
	}
	return neighborsFn(ctx)
}

// raListenerFn solicits and parses Router Advertisements, set only on platforms
// with a raw ICMPv6 socket (macOS/Linux).
var raListenerFn func(ctx context.Context, iface string, timeout time.Duration) ([]RouterAdvert, error)

// errRAUnsupported is returned when RA listening isn't built for this OS.
var errRAUnsupported = errors.New("listening for Router Advertisements needs a raw ICMPv6 socket (macOS or Linux, as root)")

// ListenRA sends a Router Solicitation and collects the Router Advertisements
// that arrive within timeout. iface optionally restricts it to one interface.
// It needs a raw ICMPv6 socket, so it requires root and is macOS/Linux only.
func ListenRA(ctx context.Context, iface string, timeout time.Duration) ([]RouterAdvert, error) {
	if raListenerFn == nil {
		return nil, errRAUnsupported
	}
	return raListenerFn(ctx, iface, timeout)
}

// RASupported reports whether this build can listen for Router Advertisements.
func RASupported() bool { return raListenerFn != nil }
