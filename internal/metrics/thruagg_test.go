package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var t0 = time.Unix(1700000000, 0)

func thru(inO, outO uint64, bits int, oper string, sec int) ThruSample {
	return ThruSample{InOctets: inO, OutOctets: outO, Bits: bits, OperStatus: oper, T: t0.Add(time.Duration(sec) * time.Second)}
}

func TestFirstTickNoSpike(t *testing.T) {
	a := NewThruAgg(30, 1_000_000_000)
	s := a.Add(thru(1000, 1000, 64, "up", 0))
	require.True(t, s.Warming)
	require.Equal(t, 0.0, s.InBps)
}

func TestBpsFromDelta(t *testing.T) {
	a := NewThruAgg(30, 100_000_000)
	a.Add(thru(0, 0, 64, "up", 0))
	s := a.Add(thru(125000, 0, 64, "up", 1)) // +125000 bytes in 1s
	require.InDelta(t, 1_000_000, s.InBps, 1)   // 1 Mbps
	require.True(t, s.HasUtil)
	require.InDelta(t, 1.0, s.UtilPct, 0.01) // 1 Mbps / 100 Mbps
}

func TestCounterWrap32(t *testing.T) {
	a := NewThruAgg(30, 100_000_000)
	a.Add(thru(4_294_966_000, 0, 32, "up", 0)) // near 2^32
	s := a.Add(thru(1000, 0, 32, "up", 1))     // wrapped: delta = 296 + 1000 + 1
	require.Greater(t, s.InBps, 0.0)
	require.Less(t, s.InBps, 100_000.0) // small, not a huge/negative value
}

func TestCounterResetDropped(t *testing.T) {
	a := NewThruAgg(30, 100_000_000)
	a.Add(thru(1_000_000, 0, 32, "up", 0))
	s := a.Add(thru(500, 0, 32, "up", 1)) // big decrease, prev not near max -> reset
	require.Equal(t, 0.0, s.InBps)
}

func TestUtilNaAtZeroSpeed(t *testing.T) {
	a := NewThruAgg(30, 0) // tunnel / no speed
	a.Add(thru(0, 0, 64, "up", 0))
	s := a.Add(thru(125000, 0, 64, "up", 1))
	require.False(t, s.HasUtil)
	require.Equal(t, 0.0, s.UtilPct)
}

func TestDownState(t *testing.T) {
	a := NewThruAgg(30, 100_000_000)
	s := a.Add(thru(0, 0, 64, "down", 0))
	require.Equal(t, "down", s.State)
}

func TestSaturated(t *testing.T) {
	a := NewThruAgg(30, 100_000_000) // 100 Mbps
	var s ThruSnapshot
	in := uint64(0)
	for i := 0; i <= 8; i++ {
		s = a.Add(thru(in, 0, 64, "up", i))
		in += 12_000_000 // 12 MB/s = 96 Mbps = 96% util
	}
	require.Equal(t, "saturated", s.State)
	require.Greater(t, s.UtilPct, 90.0)
}

func TestSetBaselineFreezes(t *testing.T) {
	a := NewThruAgg(30, 1_000_000_000)
	in := uint64(0)
	for i := 0; i <= 8; i++ {
		a.Add(thru(in, 0, 64, "up", i))
		in += 1_000_000 // ~8 Mbps steady
	}
	a.SetBaseline()
	frozen := a.Add(thru(in, 0, 64, "up", 9)).BandHi
	for i := 10; i <= 14; i++ {
		in += 10_000_000 // ~80 Mbps now
		a.Add(thru(in, 0, 64, "up", i))
	}
	s := a.Add(thru(in+10_000_000, 0, 64, "up", 15))
	require.InDelta(t, frozen, s.BandHi, 0.001) // band did not drift
}
