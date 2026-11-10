package cli

import (
	"bytes"
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

func TestRunPortScan(t *testing.T) {
	port := listenAccept(t)
	var out, errb bytes.Buffer
	code := Run([]string{"portscan", "127.0.0.1", strconv.Itoa(port), "--timeout", "500ms"}, &out, &errb)
	require.Equal(t, 0, code, errb.String())
	require.Contains(t, out.String(), strconv.Itoa(port)+"/tcp  open")
}

func TestRunPortScanNeedsPorts(t *testing.T) {
	var out, errb bytes.Buffer
	require.Equal(t, 2, Run([]string{"portscan", "127.0.0.1"}, &out, &errb))
}

type fakePingerCLI struct{}

func (fakePingerCLI) Ping(context.Context, netip.Addr, time.Duration) (time.Duration, bool, bool) {
	return 0, false, false // ICMP unavailable → forces the TCP fallback
}

func TestRunPingSweep(t *testing.T) {
	port := listenAccept(t) // on 127.0.0.1
	old := newPinger
	t.Cleanup(func() { newPinger = old })
	newPinger = func() scan.Pinger { return fakePingerCLI{} }

	var out, errb bytes.Buffer
	code := Run([]string{"pingsweep", "127.0.0.0/30", "--ports", strconv.Itoa(port), "--timeout", "500ms"}, &out, &errb)
	require.Equal(t, 0, code, errb.String())
	require.Contains(t, out.String(), "127.0.0.1")
	require.Contains(t, out.String(), "up (tcp)")
}

func TestRunPingSweepNeedsCIDR(t *testing.T) {
	var out, errb bytes.Buffer
	require.Equal(t, 2, Run([]string{"pingsweep"}, &out, &errb))
}
