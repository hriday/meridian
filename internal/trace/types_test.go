package trace

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const ms = time.Millisecond

func TestRTTStatsIgnoresTimeouts(t *testing.T) {
	min, avg, max := rttStats([]time.Duration{10 * ms, 0, 20 * ms, 0, 30 * ms})
	require.Equal(t, 10*ms, min)
	require.Equal(t, 20*ms, avg)
	require.Equal(t, 30*ms, max)
}

func TestRTTStatsAllTimeouts(t *testing.T) {
	min, avg, max := rttStats([]time.Duration{0, 0})
	require.Zero(t, min)
	require.Zero(t, avg)
	require.Zero(t, max)
}

func TestASPathDistinctInOrder(t *testing.T) {
	hops := []Hop{{ASN: "7922"}, {ASN: "7922"}, {ASN: "6939"}, {ASN: ""}, {ASN: "13335"}}
	p := asPath(hops)
	require.Len(t, p, 3)
	require.Equal(t, "7922", p[0].ASN)
	require.Equal(t, "6939", p[1].ASN)
	require.Equal(t, "13335", p[2].ASN)
}

func TestIsBoundary(t *testing.T) {
	hops := []Hop{{ASN: ""}, {ASN: "7922"}, {ASN: "7922"}, {ASN: ""}, {ASN: "6939"}}
	require.False(t, isBoundary(hops, 0)) // no ASN
	require.True(t, isBoundary(hops, 1))  // first annotated hop
	require.False(t, isBoundary(hops, 2)) // same AS
	require.True(t, isBoundary(hops, 4))  // 6939 vs previous annotated 7922
}

func TestVerdictReachedAndNotReached(t *testing.T) {
	r := Result{
		Target: "cloudflare.com", Addr: "104.16.123.45", Reached: true,
		Hops: []Hop{
			{ASN: "7922", ASOwner: "Comcast", RTTs: []time.Duration{8 * ms}},
			{ASN: "13335", ASOwner: "Cloudflare", RTTs: []time.Duration{62 * ms}},
		},
	}
	v := Verdict(r)
	require.Contains(t, v, "104.16.123.45")
	require.Contains(t, v, "cloudflare.com")
	require.Contains(t, v, "Cloudflare")
	require.Contains(t, v, "2 hops")

	nr := Verdict(Result{Target: "x.example", Reached: false, Hops: make([]Hop, 30)})
	require.Contains(t, strings.ToLower(nr), "did not reach")
}
