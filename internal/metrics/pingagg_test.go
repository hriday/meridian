package metrics

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWarmingThenSteady(t *testing.T) {
	a := NewAggregator(30)
	var s Snapshot
	for i := 0; i < 5; i++ {
		s = a.Add(10, true)
		require.True(t, s.Warming)
		require.Equal(t, "warming", s.State)
	}
	for i := 0; i < 10; i++ {
		s = a.Add(10+float64(i%3-1), true) // 9,10,11,…
	}
	require.False(t, s.Warming)
	require.Equal(t, "steady", s.State)
	require.InDelta(t, 10, s.AvgMs, 1.5)
	require.Equal(t, 0.0, s.LossPct)
	require.LessOrEqual(t, s.BandLo, 10.0)
	require.GreaterOrEqual(t, s.BandHi, 10.0)
}

func TestAnomalyOnSpike(t *testing.T) {
	a := NewAggregator(30)
	for i := 0; i < 15; i++ {
		a.Add(10, true)
	}
	s := a.Add(40, true)
	require.Equal(t, "anomaly", s.State)
	require.Greater(t, s.RTTMs, s.BandHi)
	require.True(t, s.HasRTT)
}

func TestLossStates(t *testing.T) {
	a := NewAggregator(30)
	for i := 0; i < 15; i++ {
		a.Add(10, true)
	}
	// latest probe lost → loss state, no RTT
	s := a.Add(0, false)
	require.Equal(t, "loss", s.State)
	require.False(t, s.HasRTT)
	require.Greater(t, s.LossPct, 0.0)

	// a modest loss fraction with an in-band latest responding sample → degraded
	a2 := NewAggregator(30)
	for i := 0; i < 19; i++ {
		a2.Add(10, true)
	}
	a2.Add(0, false) // one loss in 20 = 5%
	s2 := a2.Add(10, true)
	require.Equal(t, "degraded", s2.State)
}

func TestJitter(t *testing.T) {
	a := NewAggregator(30)
	for i := 0; i < 20; i++ {
		if i%2 == 0 {
			a.Add(10, true)
		} else {
			a.Add(20, true)
		}
	}
	s := a.Add(10, true)
	require.InDelta(t, 10, s.JitterMs, 1.5)
}

func TestSetBaselineFreezesBand(t *testing.T) {
	a := NewAggregator(30)
	for i := 0; i < 15; i++ {
		a.Add(10, true)
	}
	a.SetBaseline()
	frozenHi := a.Add(10, true).BandHi
	// now feed a sustained 18ms — without freezing, the band would drift up to
	// include it; frozen, 18ms stays out of band → anomaly.
	var s Snapshot
	for i := 0; i < 10; i++ {
		s = a.Add(18, true)
	}
	require.Equal(t, "anomaly", s.State)
	require.InDelta(t, frozenHi, s.BandHi, 0.001) // band did not drift
}
