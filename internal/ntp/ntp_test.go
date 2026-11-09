package ntp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// buildNTP constructs a 48-byte NTP v4 server reply. origin is echoed into the
// origin-timestamp field (what a well-behaved server copies from our request's
// transmit timestamp); serverTime fills receive+transmit. putNTPTime is the
// package's own helper, reused here.
func buildNTP(t *testing.T, li, stratum int, refID string, origin, serverTime time.Time) []byte {
	t.Helper()
	b := make([]byte, 48)
	b[0] = byte(li<<6) | (4 << 3) | 4 // LI | VN=4 | Mode=4 (server)
	b[1] = byte(stratum)
	b[3] = 0xE9 // precision = -23
	copy(b[12:16], []byte(refID))
	putNTPTime(b[24:32], origin)     // origin (echo of our transmit)
	putNTPTime(b[32:40], serverTime) // receive
	putNTPTime(b[40:48], serverTime) // transmit
	return b
}

func TestDecodeStratum1(t *testing.T) {
	t1 := time.Unix(1762700000, 0).UTC()
	t4 := t1.Add(18 * time.Millisecond)
	resp := buildNTP(t, 0, 1, "GPS", t1, t1.Add(8*time.Millisecond))

	r, err := decode(resp, t1, t4)
	require.NoError(t, err)
	require.Equal(t, 1, r.Stratum)
	require.Equal(t, "GPS", r.RefID)
	require.Equal(t, "none", r.LeapIndicator)
	require.False(t, r.KissOfDeath)
	require.InDelta(t, 18*time.Millisecond, r.RTT, float64(5*time.Millisecond))
}

func TestDecodeShortPacketErrors(t *testing.T) {
	_, err := decode(make([]byte, 20), time.Now(), time.Now())
	require.Error(t, err)
}

func TestDecodeKissOfDeath(t *testing.T) {
	t1 := time.Unix(1762700000, 0).UTC()
	resp := buildNTP(t, 0, 0, "RATE", t1, t1) // stratum 0 = kiss-o'-death
	r, err := decode(resp, t1, t1.Add(5*time.Millisecond))
	require.NoError(t, err)
	require.True(t, r.KissOfDeath)
	require.Equal(t, 0, r.Stratum)
}

func TestDecodeRejectsWrongMode(t *testing.T) {
	t1 := time.Unix(1762700000, 0).UTC()
	resp := buildNTP(t, 0, 1, "GPS", t1, t1.Add(8*time.Millisecond))
	resp[0] = (resp[0] &^ 0x07) | 3 // mode 3 (client) — not a valid server reply
	_, err := decode(resp, t1, t1.Add(18*time.Millisecond))
	require.Error(t, err)
}

func TestDecodeRejectsOriginMismatch(t *testing.T) {
	t1 := time.Unix(1762700000, 0).UTC()
	// Origin echoes a different time than our request's transmit (t1).
	resp := buildNTP(t, 0, 1, "GPS", t1.Add(time.Second), t1.Add(8*time.Millisecond))
	_, err := decode(resp, t1, t1.Add(18*time.Millisecond))
	require.Error(t, err)
}

func TestDecodeRejectsZeroTimestamps(t *testing.T) {
	t1 := time.Unix(1762700000, 0).UTC()
	resp := buildNTP(t, 0, 1, "GPS", t1, t1.Add(8*time.Millisecond))
	for i := 32; i < 48; i++ { // zero the receive + transmit timestamps
		resp[i] = 0
	}
	_, err := decode(resp, t1, t1.Add(18*time.Millisecond))
	require.Error(t, err)
}
