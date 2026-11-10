package scan

import (
	"context"
	"net"
	"net/netip"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakePinger struct {
	available bool
	reply     map[string]time.Duration // ip -> rtt for hosts that "answer" ICMP
	mu        sync.Mutex
	inflight  int
	maxFlight int
}

func (f *fakePinger) Ping(_ context.Context, ip netip.Addr, _ time.Duration) (time.Duration, bool, bool) {
	if !f.available {
		return 0, false, false
	}
	f.mu.Lock()
	f.inflight++
	if f.inflight > f.maxFlight {
		f.maxFlight = f.inflight
	}
	f.mu.Unlock()
	time.Sleep(3 * time.Millisecond)
	f.mu.Lock()
	f.inflight--
	f.mu.Unlock()
	rtt, ok := f.reply[ip.String()]
	return rtt, ok, true
}

func tcpListener(t *testing.T) int {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()
	_, p, _ := net.SplitHostPort(ln.Addr().String())
	n, _ := strconv.Atoi(p)
	return n
}

func TestSweepICMP(t *testing.T) {
	fp := &fakePinger{available: true, reply: map[string]time.Duration{"127.0.0.1": 2 * time.Millisecond}}
	res, err := Sweep(context.Background(), "127.0.0.0/30", fp, nil, 300*time.Millisecond, 8, nil)
	require.NoError(t, err)
	byIP := map[string]HostResult{}
	for _, r := range res {
		byIP[r.IP] = r
	}
	require.True(t, byIP["127.0.0.1"].Alive)
	require.Equal(t, "icmp", byIP["127.0.0.1"].Method)
}

func TestSweepTCPFallback(t *testing.T) {
	port := tcpListener(t) // listens on 127.0.0.1
	fp := &fakePinger{available: true, reply: map[string]time.Duration{}} // ICMP up but nobody answers
	res, err := Sweep(context.Background(), "127.0.0.0/30", fp, []int{port}, 300*time.Millisecond, 8, nil)
	require.NoError(t, err)
	byIP := map[string]HostResult{}
	for _, r := range res {
		byIP[r.IP] = r
	}
	require.True(t, byIP["127.0.0.1"].Alive) // found via the TCP fallback
	require.Equal(t, "tcp", byIP["127.0.0.1"].Method)
}

func TestSweepICMPUnavailable(t *testing.T) {
	port := tcpListener(t)
	fp := &fakePinger{available: false} // no ICMP at all
	res, err := Sweep(context.Background(), "127.0.0.0/30", fp, []int{port}, 300*time.Millisecond, 8, nil)
	require.NoError(t, err)
	byIP := map[string]HostResult{}
	for _, r := range res {
		byIP[r.IP] = r
	}
	require.True(t, byIP["127.0.0.1"].Alive)
	require.Equal(t, "tcp", byIP["127.0.0.1"].Method)
}

func TestSweepRejectsHugeRange(t *testing.T) {
	_, err := Sweep(context.Background(), "10.0.0.0/8", &fakePinger{available: true}, nil, time.Second, 8, nil)
	require.Error(t, err)
}

func TestSweepHostEnumeration(t *testing.T) {
	// /30 → 2 usable hosts (skip network + broadcast)
	res, err := Sweep(context.Background(), "192.0.2.0/30", &fakePinger{available: true}, nil, 50*time.Millisecond, 8, nil)
	require.NoError(t, err)
	require.Len(t, res, 2)
	ips := []string{res[0].IP, res[1].IP}
	require.ElementsMatch(t, []string{"192.0.2.1", "192.0.2.2"}, ips)

	// /32 → the single host
	res, err = Sweep(context.Background(), "192.0.2.5/32", &fakePinger{available: true}, nil, 50*time.Millisecond, 8, nil)
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.Equal(t, "192.0.2.5", res[0].IP)
}

func TestSweepBoundedConcurrency(t *testing.T) {
	fp := &fakePinger{available: true, reply: map[string]time.Duration{}}
	_, err := Sweep(context.Background(), "192.0.2.0/28", fp, nil, 200*time.Millisecond, 4, nil)
	require.NoError(t, err)
	require.LessOrEqual(t, fp.maxFlight, 4) // worker pool honoured
}
