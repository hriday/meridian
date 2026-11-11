package netflow

import (
	"net/netip"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func fr(src, dst string, dport uint16, proto uint8, bytes, pkts uint64, t time.Time) FlowRecord {
	return FlowRecord{
		Src: netip.MustParseAddr(src), Dst: netip.MustParseAddr(dst),
		DstPort: dport, Proto: proto, Bytes: bytes, Packets: pkts, T: t,
	}
}

func TestTopTalkersGroups(t *testing.T) {
	now := time.Now()
	recs := []FlowRecord{
		fr("10.0.0.9", "10.0.0.1", 443, 6, 100, 1, now.Add(-1*time.Second)),
		fr("10.0.0.9", "10.0.0.1", 443, 6, 300, 2, now.Add(-2*time.Second)),
		fr("10.0.2.5", "8.8.8.8", 53, 17, 50, 1, now.Add(-1*time.Second)),
	}
	top, _ := TopTalkers(recs, time.Minute, now, false, 10)
	require.Len(t, top, 2)
	require.Equal(t, "10.0.0.9", top[0].SrcIP) // 400 bytes beats 50
	require.EqualValues(t, 400, top[0].Bytes)
	require.Equal(t, 2, top[0].Flows)
	require.Equal(t, "TCP", top[0].Proto)
	require.Equal(t, "UDP", top[1].Proto)
}

func TestTopTalkersWindow(t *testing.T) {
	now := time.Now()
	recs := []FlowRecord{
		fr("10.0.0.9", "10.0.0.1", 443, 6, 999, 9, now.Add(-90*time.Second)), // outside 60s
		fr("10.0.2.5", "8.8.8.8", 53, 17, 50, 1, now.Add(-5*time.Second)),     // inside
	}
	top, covered := TopTalkers(recs, 60*time.Second, now, false, 10)
	require.Len(t, top, 1)
	require.Equal(t, "10.0.2.5", top[0].SrcIP)
	require.LessOrEqual(t, covered, 60)
	require.Greater(t, covered, 0)
}

func TestTopTalkersByPackets(t *testing.T) {
	now := time.Now()
	recs := []FlowRecord{
		fr("10.0.0.1", "10.0.0.2", 80, 6, 10000, 5, now),   // big bytes, few packets
		fr("10.0.0.3", "10.0.0.4", 80, 6, 500, 9999, now),  // small bytes, many packets
	}
	byBytes, _ := TopTalkers(recs, time.Minute, now, false, 10)
	require.Equal(t, "10.0.0.1", byBytes[0].SrcIP)
	byPkts, _ := TopTalkers(recs, time.Minute, now, true, 10)
	require.Equal(t, "10.0.0.3", byPkts[0].SrcIP)
}

func TestTopTalkersTopN(t *testing.T) {
	now := time.Now()
	var recs []FlowRecord
	for i := 0; i < 20; i++ {
		recs = append(recs, fr("10.0.0."+itoa(i), "10.0.0.250", 443, 6, uint64(i+1), 1, now))
	}
	top, _ := TopTalkers(recs, time.Minute, now, false, 5)
	require.Len(t, top, 5)
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}
