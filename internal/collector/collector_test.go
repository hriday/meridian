package collector

import (
	"net"
	"net/netip"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// echoParse keeps every datagram as a string message.
func echoParse(b []byte, _ netip.Addr) (string, bool) {
	if len(b) == 0 {
		return "", false
	}
	return string(b), true
}

func sendUDP(t *testing.T, port int, s string) {
	t.Helper()
	c, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: port})
	require.NoError(t, err)
	defer c.Close()
	_, err = c.Write([]byte(s))
	require.NoError(t, err)
}

func TestCollectorReceivesAndPolls(t *testing.T) {
	c := New[string](100, echoParse)
	require.NoError(t, c.Start(0)) // 0 → OS picks a free port
	defer c.Stop()
	port := c.PollSince(0).Port
	require.NotZero(t, port)
	sendUDP(t, port, "hello")
	sendUDP(t, port, "world")
	require.Eventually(t, func() bool {
		return len(c.PollSince(0).Items) >= 2
	}, 2*time.Second, 10*time.Millisecond)
	p := c.PollSince(0)
	require.True(t, p.Running)
}

func TestCollectorUnmapsSource(t *testing.T) {
	// a datagram from loopback arrives as ::ffff:127.0.0.1 on a dual-stack
	// socket; the source handed to parse must be the unmapped IPv4.
	c := New[string](10, func(_ []byte, src netip.Addr) (string, bool) { return src.String(), true })
	require.NoError(t, c.Start(0))
	defer c.Stop()
	sendUDP(t, c.PollSince(0).Port, "x")
	require.Eventually(t, func() bool {
		items := c.PollSince(0).Items
		return len(items) == 1 && items[0].Val == "127.0.0.1"
	}, 2*time.Second, 10*time.Millisecond)
}

func TestCollectorStopNoLeak(t *testing.T) {
	c := New[string](100, echoParse)
	require.NoError(t, c.Start(0))
	c.Stop() // must block until the read loop has exited
	p := c.PollSince(0)
	require.False(t, p.Running)
	require.Empty(t, p.Items)
}

func TestCollectorRestartBumpsEpoch(t *testing.T) {
	c := New[string](100, echoParse)
	require.NoError(t, c.Start(0))
	defer c.Stop()
	e1 := c.PollSince(0).Epoch
	require.NoError(t, c.Start(0)) // restart
	e2 := c.PollSince(0).Epoch
	require.Greater(t, e2, e1)
}

func TestCollectorConcurrentStartNoLeak(t *testing.T) {
	base := runtime.NumGoroutine()
	for round := 0; round < 20; round++ {
		c := New[string](10, echoParse)
		var wg sync.WaitGroup
		for i := 0; i < 4; i++ {
			wg.Add(1)
			go func() { defer wg.Done(); _ = c.Start(0) }()
		}
		wg.Wait()
		c.Stop()
	}
	// read-loop goroutines of leaked binds would never exit; allow settle.
	require.Eventually(t, func() bool {
		return runtime.NumGoroutine() <= base+3
	}, 2*time.Second, 50*time.Millisecond)
}

func TestCollectorBindInUse(t *testing.T) {
	// hold a UDP port, then try to bind it.
	held, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	require.NoError(t, err)
	defer held.Close()
	port := held.LocalAddr().(*net.UDPAddr).Port

	c := New[string](100, echoParse)
	err = c.Start(port)
	require.Error(t, err)
	require.False(t, c.PollSince(0).Running)
}
