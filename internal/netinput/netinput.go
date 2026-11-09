// Package netinput recognizes what a free-text value is — an IP address,
// a CIDR prefix, an AS number, a domain, or a hostname — and returns its
// canonical form. It is the front door for Meridian's any-value search box
// and CLI, so every tool agrees on what the user typed.
package netinput

import (
	"fmt"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
)

// Kind is the category a value was recognized as.
type Kind int

const (
	KindUnknown Kind = iota
	KindIPv4
	KindIPv6
	KindPrefixV4
	KindPrefixV6
	KindASN
	KindDomain
	KindHostname
)

func (k Kind) String() string {
	switch k {
	case KindIPv4:
		return "IPv4"
	case KindIPv6:
		return "IPv6"
	case KindPrefixV4:
		return "PrefixV4"
	case KindPrefixV6:
		return "PrefixV6"
	case KindASN:
		return "ASN"
	case KindDomain:
		return "Domain"
	case KindHostname:
		return "Hostname"
	default:
		return "Unknown"
	}
}

// Result is a recognized value and its canonical form.
type Result struct {
	Kind       Kind
	Normalized string
}

const maxASN = 4294967295 // 2^32-1, the 32-bit ASN ceiling

var (
	asnRe   = regexp.MustCompile(`^(?i:as)?(\d+)$`)
	labelRe = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9-]*[A-Za-z0-9])?$`)
)

// Classify recognizes input after trimming surrounding whitespace, or
// returns an error if it matches none of the known kinds.
func Classify(input string) (Result, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return Result{}, fmt.Errorf("empty input")
	}

	if strings.Contains(s, "/") {
		p, err := netip.ParsePrefix(s)
		if err != nil {
			return Result{}, fmt.Errorf("not a valid prefix: %w", err)
		}
		kind := KindPrefixV4
		if p.Addr().Is6() {
			kind = KindPrefixV6
		}
		return Result{Kind: kind, Normalized: p.String()}, nil
	}

	if addr, err := netip.ParseAddr(s); err == nil {
		kind := KindIPv4
		if addr.Is6() {
			kind = KindIPv6
		}
		return Result{Kind: kind, Normalized: addr.String()}, nil
	}

	// A bare "AS"/"as" with no digits reads as a half-typed AS number, not a
	// one-label hostname — reject it rather than send it to DNS.
	if strings.EqualFold(s, "as") {
		return Result{}, fmt.Errorf("incomplete AS number: %q", s)
	}

	if m := asnRe.FindStringSubmatch(s); m != nil {
		n, err := strconv.ParseUint(m[1], 10, 64)
		if err != nil || n > maxASN {
			return Result{}, fmt.Errorf("AS number out of range: %s", s)
		}
		return Result{Kind: KindASN, Normalized: fmt.Sprintf("AS%d", n)}, nil
	}

	if name, ok := classifyName(s); ok {
		kind := KindHostname
		if strings.Contains(name, ".") {
			kind = KindDomain
		}
		return Result{Kind: kind, Normalized: name}, nil
	}

	return Result{}, fmt.Errorf("unrecognized value: %q", s)
}

// classifyName lower-cases a DNS name, drops a single trailing dot, and
// confirms every label is well-formed. Returns false if it is not a name.
func classifyName(s string) (string, bool) {
	name := strings.ToLower(strings.TrimSuffix(s, "."))
	if name == "" {
		return "", false
	}
	for _, label := range strings.Split(name, ".") {
		if label == "" || !labelRe.MatchString(label) {
			return "", false
		}
	}
	return name, true
}
