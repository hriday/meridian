package app

import (
	"context"
	"testing"
	"time"

	"github.com/hriday/meridian/internal/whois"
	"github.com/stretchr/testify/require"
)

func TestWhoisMethod(t *testing.T) {
	old := whoisQuery
	whoisQuery = func(_ context.Context, object string, _ time.Duration) (whois.Result, error) {
		return whois.Result{Object: object, Server: "registry.test", Chain: []string{"whois.iana.org", "registry.test"}, Raw: "domain: " + object}, nil
	}
	t.Cleanup(func() { whoisQuery = old })

	r, err := New().Whois("example.com")
	require.NoError(t, err)
	require.Equal(t, "registry.test", r.Server)
	require.Contains(t, r.Raw, "example.com")
	require.Len(t, r.Chain, 2)
}
