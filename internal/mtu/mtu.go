// Package mtu discovers the Path MTU to a host: the largest IP packet that can
// reach it without fragmentation. It sends unprivileged ICMP echo probes with
// the Don't-Fragment bit set at varying sizes and binary-searches the boundary
// — the classic way to find an MTU bottleneck or black hole (e.g. a tunnel that
// silently drops full-size packets).
//
// Crucially it first self-checks that the OS actually enforces Don't-Fragment:
// some systems (certain VPN / Network-Extension setups, and some macOS betas)
// accept IP_DONTFRAG but fragment anyway. On those, a measurement would be a
// lie, so the tool reports "not enforced here" instead of a bogus number.
//
// The search logic here is portable and fully unit-tested over an injectable
// Prober. The real Prober (ICMP + DF) is OS-specific and lives in the
// probe_*.go files; it is unprivileged on macOS and Linux, unsupported
// elsewhere.
package mtu

import (
	"context"
	"errors"
)

// icmpV4Overhead is the IPv4 header (20) + ICMP echo header (8) added to the
// payload to get the on-the-wire IP packet size.
const icmpV4Overhead = 28

// Outcome is what one probe of a given size produced.
type Outcome int

const (
	// Unknown means no reply and no definite reason (transient loss, or a
	// silent too-big drop — the caller retries to disambiguate).
	Unknown Outcome = iota
	// Fits means an echo reply came back: a packet this size reached the host.
	Fits
	// TooBig means the probe was rejected for being too large (a local
	// EMSGSIZE, or a frag-needed ICMP error).
	TooBig
)

// ProbeResult is one probe's outcome, with the next-hop MTU when a router
// reported one (frag-needed). NextHop is 0 when unknown.
type ProbeResult struct {
	Outcome Outcome
	NextHop int // IP bytes, from an ICMP "fragmentation needed" message
}

// Prober sends one DF ICMP echo carrying payload data bytes and reports whether
// it fit. The target and socket are bound at construction.
type Prober interface {
	Probe(ctx context.Context, payload int) (ProbeResult, error)
	Close() error
}

// Result is the discovered path MTU and the context needed to act on it.
type Result struct {
	Target     string `json:"target"`
	Addr       string `json:"addr"`
	DFEnforced bool   `json:"dfEnforced"`           // false => couldn't measure (see Note)
	Note       string `json:"note,omitempty"`       // why a measurement wasn't possible
	PathMTU    int    `json:"pathMTU"`              // largest IP packet that got through
	PayloadMax int    `json:"payloadMax"`           // = PathMTU - 28 (max ICMP data)
	LocalMTU   int    `json:"localMTU"`             // outgoing interface MTU, for comparison
	NextHopMTU int    `json:"nextHopMTU,omitempty"` // if a router reported it
	Bottleneck bool   `json:"bottleneck"`           // PathMTU < LocalMTU: a smaller link on the path
}

// Discover first self-checks that Don't-Fragment is enforced, then
// binary-searches the largest payload in [0, maxPayload] that reaches the
// target. Each size is retried up to retries times before being judged too big,
// so transient loss doesn't shrink the result. localMTU is the outgoing
// interface MTU; it anchors both the self-check and the bottleneck flag.
func Discover(ctx context.Context, p Prober, target, addr string, localMTU, maxPayload, retries int) (Result, error) {
	if maxPayload < 0 {
		maxPayload = 0
	}

	// Baseline first: a minimal probe proves the host answers ICMP and, as a
	// side effect, warms the route/ARP so later probes aren't lost to cold-start
	// latency. It gets extra retries because the very first packet to a host can
	// take a second or two.
	base, _, err := probe(ctx, p, 0, retries+3)
	if err != nil {
		return Result{}, err
	}
	if !base {
		return Result{}, errors.New("no reply even to a minimal ping — the host may not answer ICMP echo, or it's unreachable")
	}

	// Self-check: one byte over the local link MTU, with DF, can never
	// legitimately get through (path MTU <= local MTU). If it does, the OS isn't
	// enforcing DF and any measurement would be fiction. The path is warm now,
	// so a reply here is a real signal, not a cold-start artifact.
	if localMTU > icmpV4Overhead {
		over := localMTU - icmpV4Overhead + 1
		fits, _, err := probe(ctx, p, over, 1)
		if err != nil {
			return Result{}, err
		}
		if fits {
			return Result{
				Target: target, Addr: addr, LocalMTU: localMTU, DFEnforced: false,
				Note: "this system isn't enforcing Don't-Fragment — a packet larger than the link MTU still got through (common with some VPNs / Network Extensions and on some macOS betas), so the path MTU can't be measured here",
			}, nil
		}
	}

	lo, hi := 0, maxPayload
	best := 0
	nextHop := 0
	for lo <= hi {
		mid := int(uint(lo+hi) >> 1)
		fits, nh, err := probe(ctx, p, mid, retries)
		if err != nil {
			return Result{}, err
		}
		if fits {
			best = mid
			lo = mid + 1
			continue
		}
		if nh > 0 {
			nextHop = nh
			if capped := nh - icmpV4Overhead; capped < hi {
				hi = capped
				continue
			}
		}
		hi = mid - 1
	}

	res := Result{
		Target: target, Addr: addr, DFEnforced: true,
		PayloadMax: best,
		PathMTU:    best + icmpV4Overhead,
		LocalMTU:   localMTU,
		NextHopMTU: nextHop,
	}
	res.Bottleneck = localMTU > 0 && res.PathMTU < localMTU
	return res, nil
}

// probe sends one size up to retries+1 times; it fits if any attempt replies.
// A definitive next-hop MTU short-circuits the retries.
func probe(ctx context.Context, p Prober, payload, retries int) (fits bool, nextHop int, err error) {
	for i := 0; i <= retries; i++ {
		if err := ctx.Err(); err != nil {
			return false, 0, err
		}
		r, err := p.Probe(ctx, payload)
		if err != nil {
			return false, 0, err
		}
		if r.Outcome == Fits {
			return true, 0, nil
		}
		if r.NextHop > 0 {
			return false, r.NextHop, nil
		}
	}
	return false, 0, nil
}
