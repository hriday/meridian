package emailauth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeLookup serves TXT records from a map keyed by exact name.
func fakeLookup(m map[string][]string) TXTLookup {
	return func(ctx context.Context, name string) ([]string, error) {
		return m[name], nil
	}
}

func TestCheckFullStack(t *testing.T) {
	lk := fakeLookup(map[string][]string{
		"example.com":                      {"v=spf1 include:_spf.google.com -all"},
		"_dmarc.example.com":               {"v=DMARC1; p=reject; pct=100; rua=mailto:dmarc@example.com"},
		"google._domainkey.example.com":    {"v=DKIM1; k=rsa; p=MIGfMA0G..."},
		"selector1._domainkey.example.com": {},
	})
	r := Check(context.Background(), lk, "example.com", []string{"google", "selector1"})

	require.Equal(t, StatusPass, r.SPF.Status)
	require.Contains(t, r.SPF.Record, "-all")

	require.Equal(t, StatusPass, r.DMARC.Status)
	require.Equal(t, "reject", r.DMARC.Policy)
	require.Contains(t, r.DMARC.RUA, "dmarc@example.com")

	require.True(t, r.DKIMFound, "at least one selector present")
	require.Len(t, r.DKIM, 2)
	require.True(t, r.DKIM[0].Found, "google selector present")
	require.False(t, r.DKIM[1].Found, "selector1 absent")
}

func TestSPFMissingAndSoftfail(t *testing.T) {
	missing := Check(context.Background(), fakeLookup(nil), "nospf.test", nil)
	require.Equal(t, StatusFail, missing.SPF.Status, "no SPF is spoofable")

	soft := Check(context.Background(), fakeLookup(map[string][]string{
		"soft.test": {"v=spf1 ~all"},
	}), "soft.test", nil)
	require.Equal(t, StatusWarn, soft.SPF.Status, "~all is softer than -all")
}

func TestSPFDuplicateInvalid(t *testing.T) {
	r := Check(context.Background(), fakeLookup(map[string][]string{
		"dup.test": {"v=spf1 -all", "v=spf1 include:x ~all"},
	}), "dup.test", nil)
	require.Equal(t, StatusFail, r.SPF.Status, "two SPF records make SPF invalid")
}

func TestDMARCPolicies(t *testing.T) {
	none := Check(context.Background(), fakeLookup(map[string][]string{
		"_dmarc.n.test": {"v=DMARC1; p=none"},
	}), "n.test", nil)
	require.Equal(t, StatusWarn, none.DMARC.Status, "p=none only monitors")
	require.Equal(t, "none", none.DMARC.Policy)

	quar := Check(context.Background(), fakeLookup(map[string][]string{
		"_dmarc.q.test": {"v=DMARC1; p=quarantine"},
	}), "q.test", nil)
	require.Equal(t, StatusPass, quar.DMARC.Status)

	missing := Check(context.Background(), fakeLookup(nil), "m.test", nil)
	require.Equal(t, StatusFail, missing.DMARC.Status)
}

func TestDefaultSelectorsUsedWhenNoneGiven(t *testing.T) {
	// "google" is in the default set; serve it and confirm Check probes it.
	r := Check(context.Background(), fakeLookup(map[string][]string{
		"google._domainkey.d.test": {"v=DKIM1; p=abc"},
	}), "d.test", nil)
	require.True(t, r.DKIMFound)
	require.NotEmpty(t, r.DKIM)
}

func TestNormalizeDomainStripsSchemeAndMailto(t *testing.T) {
	require.Equal(t, "example.com", normalizeDomain("https://example.com/path"))
	require.Equal(t, "example.com", normalizeDomain("user@example.com"))
	require.Equal(t, "example.com", normalizeDomain("  Example.com.  "))
}
