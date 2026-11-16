package dnsx

import (
	"context"
	"strings"
	"time"

	mdns "github.com/miekg/dns"
)

// DNSSECResult is the outcome of validating a zone's DNSSEC, independent of
// whether the asking resolver happens to validate (the AD flag). It reports what
// was actually found and checked, so a caller can grade it honestly rather than
// trusting a single bit.
type DNSSECResult struct {
	Signed           bool   `json:"signed"`           // the zone publishes DNSKEY record(s)
	HasDS            bool   `json:"hasDS"`            // the parent publishes a DS record
	SecureDelegation bool   `json:"secureDelegation"` // a parent DS matches a published DNSKEY (chain is linked)
	SignaturesValid  bool   `json:"signaturesValid"`  // an RRSIG over the DNSKEY set verifies and is within its validity window
	Algorithm        string `json:"algorithm"`        // signing algorithm, for display
	Detail           string `json:"detail,omitempty"` // extra context for a broken or unverifiable state
}

// ValidateDNSSEC fetches domain's DNSKEY and DS records from server and checks
// the zone's self-consistency: that the parent DS matches a DNSKEY (secure
// delegation) and that the DNSKEY set's own RRSIG verifies and is current. It is
// not a full walk to the root trust anchor — it validates the zone and its
// delegation, which is what catches the common failures (unsigned, unanchored,
// bogus, expired). A query error is returned so the caller can say "couldn't
// check" rather than invent a verdict.
func ValidateDNSSEC(ctx context.Context, domain, server string) (DNSSECResult, error) {
	keyMsg, err := QueryMsg(ctx, domain, "DNSKEY", server)
	if err != nil {
		return DNSSECResult{}, err
	}
	dsMsg, err := QueryMsg(ctx, domain, "DS", server)
	if err != nil {
		return DNSSECResult{}, err
	}

	var keys []*mdns.DNSKEY
	var sigs []*mdns.RRSIG
	for _, rr := range keyMsg.Answer {
		switch v := rr.(type) {
		case *mdns.DNSKEY:
			keys = append(keys, v)
		case *mdns.RRSIG:
			if v.TypeCovered == mdns.TypeDNSKEY {
				sigs = append(sigs, v)
			}
		}
	}
	var dss []*mdns.DS
	for _, rr := range dsMsg.Answer {
		if ds, ok := rr.(*mdns.DS); ok {
			dss = append(dss, ds)
		}
	}
	return classifyDNSSEC(keys, dss, sigs, time.Now()), nil
}

// classifyDNSSEC is the pure core of ValidateDNSSEC: given the fetched records
// and the current time, it decides what was proven. Separated out so it can be
// unit-tested with generated keys and signatures, no network.
func classifyDNSSEC(keys []*mdns.DNSKEY, dss []*mdns.DS, sigs []*mdns.RRSIG, now time.Time) DNSSECResult {
	if len(keys) == 0 {
		return DNSSECResult{} // unsigned
	}
	res := DNSSECResult{Signed: true, HasDS: len(dss) > 0}
	res.Algorithm = mdns.AlgorithmToString[keys[0].Algorithm]

	// Secure delegation: some parent DS is the digest of a published DNSKEY.
	for _, ds := range dss {
		for _, k := range keys {
			if k.KeyTag() != ds.KeyTag || k.Algorithm != ds.Algorithm {
				continue
			}
			if computed := k.ToDS(ds.DigestType); computed != nil && strings.EqualFold(computed.Digest, ds.Digest) {
				res.SecureDelegation = true
			}
		}
	}

	// Signatures: an RRSIG over the DNSKEY set verifies against one of the keys
	// and is inside its validity window. The signed rrset is the DNSKEY set.
	keyset := make([]mdns.RR, len(keys))
	for i, k := range keys {
		keyset[i] = k
	}
	for _, sig := range sigs {
		if !sig.ValidityPeriod(now) {
			res.Detail = "the DNSKEY signature is outside its validity window (expired or not yet valid)"
			continue
		}
		for _, k := range keys {
			if k.KeyTag() != sig.KeyTag {
				continue
			}
			if err := sig.Verify(k, keyset); err == nil {
				res.SignaturesValid = true
				res.Detail = ""
				break
			} else if res.Detail == "" {
				res.Detail = "the DNSKEY signature could not be verified (" + err.Error() + ")"
			}
		}
		if res.SignaturesValid {
			break
		}
	}
	return res
}
