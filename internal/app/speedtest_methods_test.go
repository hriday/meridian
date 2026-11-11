package app

import (
	"context"
	"sync"
	"testing"
	"time"

	iperf3 "github.com/appshubcc/go-iperf3"
	"github.com/stretchr/testify/require"
)

func loopbackPort(t *testing.T) int {
	t.Helper()
	srv, err := iperf3.Listen(iperf3.ServerConfig{Addr: "127.0.0.1:0"})
	require.NoError(t, err)
	t.Cleanup(func() { srv.Close() })
	return srv.Port()
}

// captureSpeed swaps the emitEvent seam to collect SpeedSnapshots.
func captureSpeed(t *testing.T) (*[]SpeedSnapshot, *sync.Mutex) {
	t.Helper()
	var mu sync.Mutex
	var snaps []SpeedSnapshot
	orig := emitEvent
	emitEvent = func(_ context.Context, _ string, data ...interface{}) {
		if len(data) == 1 {
			if s, ok := data[0].(SpeedSnapshot); ok {
				mu.Lock()
				snaps = append(snaps, s)
				mu.Unlock()
			}
		}
	}
	t.Cleanup(func() { emitEvent = orig })
	return &snaps, &mu
}

func countState(snaps *[]SpeedSnapshot, mu *sync.Mutex, state string) int {
	mu.Lock()
	defer mu.Unlock()
	n := 0
	for _, s := range *snaps {
		if s.State == state {
			n++
		}
	}
	return n
}

func TestSpeedtestStreamsAndCompletes(t *testing.T) {
	port := loopbackPort(t)
	snaps, mu := captureSpeed(t)
	a := New()
	require.NoError(t, a.SpeedtestStart("s1", "127.0.0.1", port, "download", 2, 1, false))
	require.Eventually(t, func() bool { return countState(snaps, mu, "done") == 1 }, 10*time.Second, 50*time.Millisecond)
	require.Greater(t, countState(snaps, mu, "running"), 0)
	mu.Lock()
	defer mu.Unlock()
	for _, s := range *snaps {
		if s.State == "done" {
			require.Greater(t, s.AvgDownMbps, 0.0)
		}
	}
}

func TestSpeedtestNoServerErrors(t *testing.T) {
	snaps, mu := captureSpeed(t)
	a := New()
	require.NoError(t, a.SpeedtestStart("s2", "127.0.0.1", 1, "download", 1, 1, false))
	require.Eventually(t, func() bool { return countState(snaps, mu, "error") == 1 }, 10*time.Second, 50*time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	for _, s := range *snaps {
		if s.State == "error" {
			require.NotEmpty(t, s.Note)
		}
	}
}

func TestSpeedtestStopNoTerminal(t *testing.T) {
	port := loopbackPort(t)
	snaps, mu := captureSpeed(t)
	a := New()
	require.NoError(t, a.SpeedtestStart("s3", "127.0.0.1", port, "download", 2, 30, false))
	time.Sleep(300 * time.Millisecond) // let it get going
	a.SpeedtestStop("s3")              // cancels + waits for the goroutine to exit
	require.Equal(t, 0, countState(snaps, mu, "done"))
	require.Equal(t, 0, countState(snaps, mu, "error"))
}

func TestSpeedtestDuplicateStart(t *testing.T) {
	port := loopbackPort(t)
	a := New()
	require.NoError(t, a.SpeedtestStart("dup", "127.0.0.1", port, "download", 1, 10, false))
	defer a.SpeedtestStop("dup")
	require.Error(t, a.SpeedtestStart("dup", "127.0.0.1", port, "download", 1, 10, false))
}

func TestSpeedtestBadHost(t *testing.T) {
	require.Error(t, New().SpeedtestStart("s4", "  ", 5201, "download", 1, 10, false))
}
