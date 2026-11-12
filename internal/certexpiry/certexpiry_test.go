package certexpiry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 12, 0, 0, 0, time.UTC)
}

func TestNormalizeHost(t *testing.T) {
	require.Equal(t, "example.com:443", NormalizeHost("example.com"))
	require.Equal(t, "example.com:443", NormalizeHost("https://example.com/path?q=1"))
	require.Equal(t, "example.com:8443", NormalizeHost("example.com:8443"))
	require.Equal(t, "example.com:8443", NormalizeHost("https://example.com:8443"))
	require.Equal(t, "", NormalizeHost("   "))
}

func TestCheckComputesDaysAndSorts(t *testing.T) {
	now := day(2026, time.January, 1)
	probe := func(ctx context.Context, hostport string) (Leaf, error) {
		switch hostport {
		case "soon.example:443":
			return Leaf{NotAfter: day(2026, time.January, 11), Subject: "soon.example", Issuer: "Let's Encrypt"}, nil
		case "later.example:443":
			return Leaf{NotAfter: day(2026, time.March, 2), Subject: "later.example", Issuer: "DigiCert"}, nil
		case "expired.example:443":
			return Leaf{NotAfter: day(2025, time.December, 20), Subject: "expired.example", Issuer: "Old CA"}, nil
		}
		return Leaf{}, errors.New("unexpected host")
	}

	res := Check(context.Background(), probe, []string{"later.example", "soon.example", "expired.example"}, now)
	require.Len(t, res, 3)

	// Sorted soonest-expiry first: expired (negative) → soon (10d) → later (60d).
	require.Equal(t, "expired.example:443", res[0].Host)
	require.True(t, res[0].Expired)
	require.Equal(t, -12, res[0].DaysLeft)

	require.Equal(t, "soon.example:443", res[1].Host)
	require.False(t, res[1].Expired)
	require.Equal(t, 10, res[1].DaysLeft)
	require.Equal(t, "Let's Encrypt", res[1].Issuer)

	require.Equal(t, "later.example:443", res[2].Host)
	require.Equal(t, 60, res[2].DaysLeft)
}

func TestCheckErrorsSortAfterSuccesses(t *testing.T) {
	now := day(2026, time.January, 1)
	probe := func(ctx context.Context, hostport string) (Leaf, error) {
		if hostport == "good.example:443" {
			return Leaf{NotAfter: day(2026, time.February, 1)}, nil
		}
		return Leaf{}, errors.New("connection refused")
	}
	res := Check(context.Background(), probe, []string{"dead.example", "good.example"}, now)
	require.Len(t, res, 2)
	require.True(t, res[0].OK, "successful probe sorts before the error")
	require.Equal(t, "good.example:443", res[0].Host)
	require.False(t, res[1].OK)
	require.Equal(t, "connection refused", res[1].Error)
}

func TestCheckDedupesHosts(t *testing.T) {
	now := day(2026, time.January, 1)
	calls := 0
	probe := func(ctx context.Context, hostport string) (Leaf, error) {
		calls++
		return Leaf{NotAfter: day(2026, time.February, 1)}, nil
	}
	res := Check(context.Background(), probe, []string{"a.example", "a.example", "https://a.example/"}, now)
	require.Len(t, res, 1, "same host collapses to one row")
	require.Equal(t, 1, calls, "probed once")
}

func TestCheckNoHosts(t *testing.T) {
	res := Check(context.Background(), nil, []string{"   ", ""}, day(2026, time.January, 1))
	require.Empty(t, res)
}

func TestSplitHosts(t *testing.T) {
	got := SplitHosts("example.com, 10.0.0.1\nhttps://foo.test:8443\n\n  bar.test  ")
	require.Equal(t, []string{"example.com", "10.0.0.1", "https://foo.test:8443", "bar.test"}, got)
}
