package scan

import (
	"context"
	"net"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParsePorts(t *testing.T) {
	got, err := ParsePorts("22,80,8000-8002")
	require.NoError(t, err)
	require.Equal(t, []int{22, 80, 8000, 8001, 8002}, got)

	_, err = ParsePorts("bad")
	require.Error(t, err)
	_, err = ParsePorts("9999999")
	require.Error(t, err)
}

func freePort(t *testing.T) int {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	_, p, _ := net.SplitHostPort(ln.Addr().String())
	ln.Close() // now free (very likely)
	n, _ := strconv.Atoi(p)
	return n
}

func TestScanPortStates(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	_, op, _ := net.SplitHostPort(ln.Addr().String())
	openPort, _ := strconv.Atoi(op)
	closedPort := freePort(t)

	var fired atomic.Int32
	res, err := ScanPorts(context.Background(), "127.0.0.1",
		[]int{openPort, closedPort}, 500*time.Millisecond, 8,
		func(PortResult) { fired.Add(1) }) // onResult may fire concurrently
	require.NoError(t, err)
	require.Equal(t, int32(2), fired.Load())

	states := map[int]string{}
	for _, r := range res {
		states[r.Port] = r.State
	}
	require.Equal(t, "open", states[openPort])
	require.Equal(t, "closed", states[closedPort])
	// results sorted by port
	require.True(t, res[0].Port <= res[1].Port)
}

func TestScanPortFilteredOnCancel(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	_, p, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(p)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := ScanPorts(ctx, "127.0.0.1", []int{port}, time.Second, 8, nil)
	require.NoError(t, err)
	require.Equal(t, "filtered", res[0].State) // cancelled → filtered, not open/closed
}

func TestScanPortsService(t *testing.T) {
	res, err := ScanPorts(context.Background(), "127.0.0.1",
		[]int{22, 443}, 300*time.Millisecond, 8, nil)
	require.NoError(t, err)
	svc := map[int]string{}
	for _, r := range res {
		svc[r.Port] = r.Service
	}
	require.Equal(t, "ssh", svc[22])
	require.Equal(t, "https", svc[443])
}
