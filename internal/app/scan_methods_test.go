package app

import (
	"context"
	"net"
	"net/netip"
	"strconv"
	"testing"
	"time"

	"github.com/hriday/meridian/internal/scan"
	"github.com/stretchr/testify/require"
)

func listenAccept(t *testing.T) int {
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

func TestAppPortScan(t *testing.T) {
	port := listenAccept(t)
	a := New()
	res, err := a.PortScan("127.0.0.1", []int{port})
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.Equal(t, "open", res[0].State)
}

type fakePinger struct{}

func (fakePinger) Ping(context.Context, netip.Addr, time.Duration) (time.Duration, bool, bool) {
	return 0, false, false // force the TCP fallback
}

func TestAppPingSweep(t *testing.T) {
	port := listenAccept(t)
	a := New()
	a.newPinger = func() scan.Pinger { return fakePinger{} }
	res, err := a.PingSweep("127.0.0.0/30", []int{port})
	require.NoError(t, err)
	byIP := map[string]scan.HostResult{}
	for _, r := range res {
		byIP[r.IP] = r
	}
	require.True(t, byIP["127.0.0.1"].Alive)
	require.Equal(t, "tcp", byIP["127.0.0.1"].Method)
}
