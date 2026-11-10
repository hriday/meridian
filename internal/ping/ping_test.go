package ping

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func ok(seq int, rtt float64) Reply { return Reply{Seq: seq, RTTMillis: rtt, OK: true} }
func lost(seq int) Reply            { return Reply{Seq: seq, OK: false} }

func TestSummarizeBasic(t *testing.T) {
	s := summarize("h", "1.2.3.4", []Reply{ok(1, 10), ok(2, 20), ok(3, 30)}, true)
	require.Equal(t, 3, s.Sent)
	require.Equal(t, 3, s.Received)
	require.InDelta(t, 0, s.LossPct, 1e-9)
	require.InDelta(t, 10, s.MinMs, 1e-9)
	require.InDelta(t, 20, s.AvgMs, 1e-9)
	require.InDelta(t, 30, s.MaxMs, 1e-9)
	require.InDelta(t, 8.16496580927726, s.StddevMs, 1e-6) // population stddev of 10,20,30
	require.Equal(t, "1.2.3.4", s.IP)
	require.True(t, s.Available)
}

func TestSummarizeJitterConsecutive(t *testing.T) {
	// Jitter is the mean absolute difference of consecutive RTTs, not the range.
	s := summarize("h", "", []Reply{ok(1, 10), ok(2, 40), ok(3, 10)}, true)
	require.InDelta(t, 30, s.JitterMs, 1e-9) // mean(|40-10|, |10-40|) = 30
	s2 := summarize("h", "", []Reply{ok(1, 10), ok(2, 40), ok(3, 20)}, true)
	require.InDelta(t, 25, s2.JitterMs, 1e-9) // mean(|30|, |20|) = 25 (range would be 30)
}

func TestSummarizeLoss(t *testing.T) {
	s := summarize("h", "", []Reply{ok(1, 10), lost(2), lost(3), lost(4)}, true)
	require.Equal(t, 4, s.Sent)
	require.Equal(t, 1, s.Received)
	require.InDelta(t, 75, s.LossPct, 1e-9)
	require.InDelta(t, 10, s.MinMs, 1e-9)
	require.InDelta(t, 10, s.AvgMs, 1e-9)
	require.InDelta(t, 10, s.MaxMs, 1e-9)
	require.InDelta(t, 0, s.StddevMs, 1e-9)
	require.InDelta(t, 0, s.JitterMs, 1e-9)
}

func TestSummarizeAllLoss(t *testing.T) {
	s := summarize("h", "", []Reply{lost(1), lost(2), lost(3)}, true)
	require.Equal(t, 0, s.Received)
	require.InDelta(t, 100, s.LossPct, 1e-9)
	for _, v := range []float64{s.MinMs, s.AvgMs, s.MaxMs, s.StddevMs, s.JitterMs} {
		require.False(t, math.IsNaN(v))
		require.InDelta(t, 0, v, 1e-9)
	}
}

func TestSummarizeSingle(t *testing.T) {
	s := summarize("h", "", []Reply{ok(1, 15)}, true)
	require.Equal(t, 1, s.Received)
	require.InDelta(t, 15, s.MinMs, 1e-9)
	require.InDelta(t, 15, s.AvgMs, 1e-9)
	require.InDelta(t, 15, s.MaxMs, 1e-9)
	require.InDelta(t, 0, s.StddevMs, 1e-9)
	require.InDelta(t, 0, s.JitterMs, 1e-9)
}
