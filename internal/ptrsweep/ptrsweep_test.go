package ptrsweep

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func fakeResolver(m map[string][]string) Resolver {
	return func(ctx context.Context, ip string) ([]string, error) {
		return m[ip], nil
	}
}

func TestSweepReturnsNamedHostsSorted(t *testing.T) {
	res := fakeResolver(map[string][]string{
		"10.0.0.1": {"gw.local."},
		"10.0.0.5": {"host5.local.", "alias5.local."},
		"10.0.0.3": {}, // no PTR
	})
	entries, err := Sweep(context.Background(), res, "10.0.0.0/29", 8, nil)
	require.NoError(t, err)
	require.Len(t, entries, 2, "only IPs with a PTR record are returned")
	require.Equal(t, "10.0.0.1", entries[0].IP, "sorted by IP")
	require.Equal(t, "10.0.0.5", entries[1].IP)
	require.Equal(t, "gw.local", entries[0].Names[0], "trailing dot trimmed")
	require.Len(t, entries[1].Names, 2)
}

func TestSweepStreamsResults(t *testing.T) {
	res := fakeResolver(map[string][]string{"192.168.1.1": {"r1."}})
	var streamed []Entry
	_, err := Sweep(context.Background(), res, "192.168.1.0/30", 4, func(e Entry) {
		streamed = append(streamed, e)
	})
	require.NoError(t, err)
	require.Len(t, streamed, 1)
	require.Equal(t, "192.168.1.1", streamed[0].IP)
}

func TestSweepInvalidCIDR(t *testing.T) {
	_, err := Sweep(context.Background(), fakeResolver(nil), "not-a-cidr", 4, nil)
	require.Error(t, err)
}

func TestSweepRejectsHugeRange(t *testing.T) {
	_, err := Sweep(context.Background(), fakeResolver(nil), "10.0.0.0/8", 4, nil)
	require.Error(t, err, "a /8 is too large to sweep")
}

func TestHostsInCount(t *testing.T) {
	hosts, err := hostsIn("10.0.0.0/30", 1000)
	require.NoError(t, err)
	require.Equal(t, []string{"10.0.0.0", "10.0.0.1", "10.0.0.2", "10.0.0.3"}, hosts)
}
