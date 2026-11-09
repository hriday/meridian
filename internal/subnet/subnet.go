// Package subnet computes the facts about an IPv4 or IPv6 CIDR block —
// its range, mask, host counts, and usable addresses — and splits or
// summarizes prefixes. Counts cross the Go/JS boundary as decimal strings
// so IPv6 magnitudes survive intact.
package subnet

import (
	"fmt"
	"math/big"
	"net/netip"

	"go4.org/netipx"
)

// maxSplitBits caps how far Split will expand a prefix: at most 2^16 = 65536
// subnets per call, which keeps the result enumerable without blocking.
const maxSplitBits = 16

// Info is the full description of one prefix.
type Info struct {
	Prefix      string // normalized CIDR (host bits cleared)
	Version     int    // 4 or 6
	Network     string
	Broadcast   string // v4 only; "" for v6
	Netmask     string // v4 dotted-decimal; "" for v6
	Wildcard    string // v4 only; "" for v6
	PrefixLen   int
	TotalCount  string // decimal string (may be huge for v6)
	UsableFirst string
	UsableLast  string
	UsableCount string // decimal string
}

// Describe returns the Info for cidr, accepting input whose host bits are
// set (they are cleared to the network) and rejecting non-CIDR input.
func Describe(cidr string) (Info, error) {
	p, err := netip.ParsePrefix(cidr)
	if err != nil {
		return Info{}, fmt.Errorf("not a valid CIDR: %w", err)
	}
	p = p.Masked()
	bits := p.Addr().BitLen() // 32 or 128
	plen := p.Bits()

	total := new(big.Int).Lsh(big.NewInt(1), uint(bits-plen))
	r := netipx.RangeOfPrefix(p)
	first, last := r.From(), r.To()

	info := Info{
		Prefix:     p.String(),
		Version:    4,
		Network:    first.String(),
		PrefixLen:  plen,
		TotalCount: total.String(),
	}
	if p.Addr().Is6() {
		info.Version = 6
		// For IPv6 the whole block is usable; there is no broadcast.
		info.UsableFirst = first.String()
		info.UsableLast = last.String()
		info.UsableCount = total.String()
		return info, nil
	}

	// IPv4.
	info.Broadcast = last.String()
	info.Netmask = netmaskV4(plen)
	info.Wildcard = wildcardV4(plen)
	switch {
	case plen == 32: // single host, no broadcast
		info.Broadcast = ""
		info.UsableFirst = first.String()
		info.UsableLast = first.String()
		info.UsableCount = "1"
	case plen == 31: // RFC 3021 point-to-point: both addresses usable
		info.UsableFirst = first.String()
		info.UsableLast = last.String()
		info.UsableCount = "2"
	default:
		info.UsableFirst = first.Next().String()
		info.UsableLast = last.Prev().String()
		info.UsableCount = new(big.Int).Sub(total, big.NewInt(2)).String()
	}
	return info, nil
}

// Split divides cidr into equal subnets of length newPrefixLen, which must
// be longer than cidr's own prefix length.
func Split(cidr string, newPrefixLen int) ([]string, error) {
	p, err := netip.ParsePrefix(cidr)
	if err != nil {
		return nil, fmt.Errorf("not a valid CIDR: %w", err)
	}
	p = p.Masked()
	if newPrefixLen <= p.Bits() || newPrefixLen > p.Addr().BitLen() {
		return nil, fmt.Errorf("new prefix length %d must be between %d and %d", newPrefixLen, p.Bits()+1, p.Addr().BitLen())
	}
	// Guard against a request that would enumerate an unreasonable number of
	// subnets (e.g. /8 → /32 is ~16M), which would exhaust memory and block
	// the UI. 2^maxSplitBits is the ceiling on how many we will produce.
	if newPrefixLen-p.Bits() > maxSplitBits {
		return nil, fmt.Errorf("splitting /%d into /%d would make %d×2^%d subnets; split into a shorter prefix (at most /%d)",
			p.Bits(), newPrefixLen, 1, newPrefixLen-p.Bits(), p.Bits()+maxSplitBits)
	}

	var out []string
	addr := p.Addr()
	end := netipx.RangeOfPrefix(p).To()
	for {
		sub := netip.PrefixFrom(addr, newPrefixLen)
		out = append(out, sub.String())
		next := netipx.RangeOfPrefix(sub).To().Next()
		if !next.IsValid() || next.Compare(end) > 0 {
			break
		}
		addr = next
	}
	return out, nil
}

// Summarize aggregates cidrs into the minimal set of prefixes covering
// exactly the same addresses. All inputs must be the same IP family.
func Summarize(cidrs []string) ([]string, error) {
	var b netipx.IPSetBuilder
	var sawV4, sawV6 bool
	for _, c := range cidrs {
		p, err := netip.ParsePrefix(c)
		if err != nil {
			return nil, fmt.Errorf("not a valid CIDR %q: %w", c, err)
		}
		if p.Addr().Is6() {
			sawV6 = true
		} else {
			sawV4 = true
		}
		b.AddPrefix(p.Masked())
	}
	if sawV4 && sawV6 {
		return nil, fmt.Errorf("cannot summarize mixed IPv4 and IPv6 prefixes")
	}
	set, err := b.IPSet()
	if err != nil {
		return nil, err
	}
	// Prefixes() already returns blocks in numeric order; keep it (a string
	// sort would order 100.0.0.0/8 before 9.0.0.0/8).
	prefixes := set.Prefixes()
	out := make([]string, 0, len(prefixes))
	for _, p := range prefixes {
		out = append(out, p.String())
	}
	return out, nil
}

func netmaskV4(plen int) string {
	mask := uint32(0xffffffff) << (32 - plen)
	if plen == 0 {
		mask = 0
	}
	return netip.AddrFrom4([4]byte{byte(mask >> 24), byte(mask >> 16), byte(mask >> 8), byte(mask)}).String()
}

func wildcardV4(plen int) string {
	wild := ^(uint32(0xffffffff) << (32 - plen))
	if plen == 0 {
		wild = 0xffffffff
	}
	return netip.AddrFrom4([4]byte{byte(wild >> 24), byte(wild >> 16), byte(wild >> 8), byte(wild)}).String()
}
