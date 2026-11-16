package dnsx

import (
	"crypto"
	"testing"
	"time"

	mdns "github.com/miekg/dns"
	"github.com/stretchr/testify/require"
)

// signedZone generates a KSK for example.test., signs the DNSKEY set, and
// returns the key, a matching DS, and the RRSIG over the key set valid at now.
func signedZone(t *testing.T, now time.Time) (*mdns.DNSKEY, *mdns.DS, *mdns.RRSIG) {
	t.Helper()
	key := &mdns.DNSKEY{
		Hdr:   mdns.RR_Header{Name: "example.test.", Rrtype: mdns.TypeDNSKEY, Class: mdns.ClassINET, Ttl: 3600},
		Flags: 257, Protocol: 3, Algorithm: mdns.ECDSAP256SHA256,
	}
	priv, err := key.Generate(256)
	require.NoError(t, err)
	sig := &mdns.RRSIG{
		Hdr:         mdns.RR_Header{Name: "example.test.", Rrtype: mdns.TypeRRSIG, Class: mdns.ClassINET, Ttl: 3600},
		KeyTag:      key.KeyTag(),
		SignerName:  "example.test.",
		Algorithm:   key.Algorithm,
		TypeCovered: mdns.TypeDNSKEY,
		Labels:      2,
		OrigTtl:     3600,
		Inception:   uint32(now.Add(-time.Hour).Unix()),
		Expiration:  uint32(now.Add(24 * time.Hour).Unix()),
	}
	require.NoError(t, sig.Sign(priv.(crypto.Signer), []mdns.RR{key}))
	return key, key.ToDS(mdns.SHA256), sig
}

func TestClassifyDNSSEC(t *testing.T) {
	now := time.Now()
	key, ds, sig := signedZone(t, now)

	t.Run("fully valid", func(t *testing.T) {
		r := classifyDNSSEC([]*mdns.DNSKEY{key}, []*mdns.DS{ds}, []*mdns.RRSIG{sig}, now)
		require.True(t, r.Signed)
		require.True(t, r.HasDS)
		require.True(t, r.SecureDelegation)
		require.True(t, r.SignaturesValid)
		require.Equal(t, "ECDSAP256SHA256", r.Algorithm)
	})

	t.Run("unsigned", func(t *testing.T) {
		r := classifyDNSSEC(nil, nil, nil, now)
		require.False(t, r.Signed)
	})

	t.Run("signed but no DS at parent", func(t *testing.T) {
		r := classifyDNSSEC([]*mdns.DNSKEY{key}, nil, []*mdns.RRSIG{sig}, now)
		require.True(t, r.Signed)
		require.False(t, r.HasDS)
		require.False(t, r.SecureDelegation)
		require.True(t, r.SignaturesValid)
	})

	t.Run("bogus DS does not match a key", func(t *testing.T) {
		bogus := &mdns.DS{KeyTag: key.KeyTag(), Algorithm: key.Algorithm, DigestType: mdns.SHA256, Digest: "00" + ds.Digest[2:]}
		r := classifyDNSSEC([]*mdns.DNSKEY{key}, []*mdns.DS{bogus}, []*mdns.RRSIG{sig}, now)
		require.True(t, r.HasDS)
		require.False(t, r.SecureDelegation, "a DS with the wrong digest must not count as secure")
	})

	t.Run("expired signature", func(t *testing.T) {
		_, _, old := signedZone(t, now.Add(-72*time.Hour)) // signed in the past, already expired
		r := classifyDNSSEC([]*mdns.DNSKEY{key}, []*mdns.DS{ds}, []*mdns.RRSIG{old}, now)
		require.False(t, r.SignaturesValid, "an expired RRSIG must not validate")
		require.Contains(t, r.Detail, "validity")
	})
}
