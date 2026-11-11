package collector

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRingSeqAndSince(t *testing.T) {
	r := NewRing[string](10)
	r.Add("a")
	r.Add("b")
	r.Add("c")
	items, total, dropped := r.Snapshot(0)
	require.Len(t, items, 3)
	require.Equal(t, 3, total)
	require.Equal(t, 0, dropped)
	require.Equal(t, 1, items[0].Seq)
	require.Equal(t, "a", items[0].Val)
	require.Equal(t, 3, items[2].Seq)

	// since cursor 2 → only seq 3
	items, _, _ = r.Snapshot(2)
	require.Len(t, items, 1)
	require.Equal(t, 3, items[0].Seq)
}

func TestRingAddAll(t *testing.T) {
	r := NewRing[int](10)
	r.AddAll([]int{1, 2, 3})
	items, total, dropped := r.Snapshot(0)
	require.Len(t, items, 3)
	require.Equal(t, 3, total)
	require.Equal(t, 0, dropped)
	require.Equal(t, 1, items[0].Seq)
	require.Equal(t, 3, items[2].Seq)

	small := NewRing[int](2)
	small.AddAll([]int{10, 20, 30, 40}) // 4 into cap 2 → 2 dropped
	_, total, dropped = small.Snapshot(0)
	require.Equal(t, 2, total)
	require.Equal(t, 2, dropped)
}

func TestRingValuesFilter(t *testing.T) {
	r := NewRing[int](10)
	r.AddAll([]int{1, 2, 3, 4, 5})
	even := r.Values(func(v int) bool { return v%2 == 0 })
	require.Equal(t, []int{2, 4}, even)
}

func TestRingEvictsAndCountsDropped(t *testing.T) {
	r := NewRing[int](3)
	for i := 0; i < 5; i++ {
		r.Add(i)
	}
	items, total, dropped := r.Snapshot(0)
	require.Equal(t, 3, total)
	require.Equal(t, 2, dropped)
	require.Len(t, items, 3)
	require.Equal(t, 3, items[0].Seq) // oldest retained is seq 3 (seqs 1,2 evicted)
	require.Equal(t, 5, items[2].Seq)
}
