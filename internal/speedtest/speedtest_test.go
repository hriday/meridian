package speedtest

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	iperf3 "github.com/appshubcc/go-iperf3"
	"github.com/stretchr/testify/require"
)

// loopbackServer starts an in-process iperf3 server on 127.0.0.1 for tests only
// (the shipped app never serves). Returns the port and a stop func.
func loopbackServer(t *testing.T) int {
	t.Helper()
	srv, err := iperf3.Listen(iperf3.ServerConfig{Addr: "127.0.0.1:0"})
	require.NoError(t, err)
	t.Cleanup(func() { srv.Close() })
	return srv.Port()
}

func collect() (func(Sample), *[]Sample, *sync.Mutex) {
	var mu sync.Mutex
	var samples []Sample
	return func(s Sample) { mu.Lock(); samples = append(samples, s); mu.Unlock() }, &samples, &mu
}

func TestRunDownloadLoopback(t *testing.T) {
	port := loopbackServer(t)
	onSample, samples, mu := collect()
	res, err := Run(context.Background(), Options{Host: "127.0.0.1", Port: port, Direction: "download", Parallel: 2, Duration: time.Second}, onSample)
	require.NoError(t, err)
	require.Greater(t, res.DownMbps, 0.0)
	mu.Lock()
	defer mu.Unlock()
	gotRunning := false
	for _, s := range *samples {
		if s.Phase == "running" && s.DownMbps > 0 {
			gotRunning = true
		}
	}
	require.True(t, gotRunning, "expected at least one running sample with download throughput")
}

func TestRunUploadLoopback(t *testing.T) {
	port := loopbackServer(t)
	res, err := Run(context.Background(), Options{Host: "127.0.0.1", Port: port, Direction: "upload", Parallel: 2, Duration: time.Second}, func(Sample) {})
	require.NoError(t, err)
	require.Greater(t, res.UpMbps, 0.0)
}

func TestRunBidirLoopback(t *testing.T) {
	port := loopbackServer(t)
	res, err := Run(context.Background(), Options{Host: "127.0.0.1", Port: port, Direction: "bidir", Parallel: 1, Duration: time.Second}, func(Sample) {})
	require.NoError(t, err)
	require.Greater(t, res.DownMbps, 0.0)
	require.Greater(t, res.UpMbps, 0.0)
}

func TestRunNoTerminalSample(t *testing.T) {
	port := loopbackServer(t)
	onSample, samples, mu := collect()
	_, err := Run(context.Background(), Options{Host: "127.0.0.1", Port: port, Direction: "download", Duration: time.Second}, onSample)
	require.NoError(t, err)
	mu.Lock()
	defer mu.Unlock()
	for _, s := range *samples {
		require.NotEqual(t, "done", s.Phase) // terminal state is the returned Result, not a sample
	}
}

func TestRunNoServer(t *testing.T) {
	_, err := Run(context.Background(), Options{Host: "127.0.0.1", Port: 1, Duration: time.Second}, func(Sample) {})
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "server")
}

func TestRunBadHost(t *testing.T) {
	_, err := Run(context.Background(), Options{Host: "no.such.host.invalid.", Port: 5201, Duration: time.Second}, func(Sample) {})
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "resolve")
}

func TestRunCancel(t *testing.T) {
	port := loopbackServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(200 * time.Millisecond); cancel() }()
	start := time.Now()
	_, err := Run(ctx, Options{Host: "127.0.0.1", Port: port, Direction: "download", Duration: 30 * time.Second}, func(Sample) {})
	require.Error(t, err)
	require.Less(t, time.Since(start), 10*time.Second) // returned promptly, didn't run 30s
}
