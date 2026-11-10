package wol

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSendDeliversMagicPacket(t *testing.T) {
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = pc.Close() })
	_, portStr, _ := net.SplitHostPort(pc.LocalAddr().String())
	port := atoiMust(t, portStr)

	done := make(chan []byte, 1)
	go func() {
		buf := make([]byte, 256)
		_ = pc.SetReadDeadline(time.Now().Add(3 * time.Second))
		n, _, _ := pc.ReadFrom(buf)
		done <- buf[:n]
	}()

	err = Send(context.Background(), "01:02:03:04:05:06", "127.0.0.1", port)
	require.NoError(t, err)

	want, _ := MagicPacket("01:02:03:04:05:06")
	select {
	case got := <-done:
		require.Equal(t, want, got)
	case <-time.After(3 * time.Second):
		t.Fatal("no datagram received")
	}
}

func TestSendRejectsBadMAC(t *testing.T) {
	// A bad MAC must error before any socket is opened — no datagram sent.
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = pc.Close() })
	_, portStr, _ := net.SplitHostPort(pc.LocalAddr().String())
	port, _ := net.LookupPort("udp", portStr)

	sendErr := Send(context.Background(), "nope", "127.0.0.1", port)
	require.Error(t, sendErr)

	_ = pc.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	buf := make([]byte, 128)
	_, _, rerr := pc.ReadFrom(buf)
	require.Error(t, rerr, "no datagram should arrive for a rejected MAC")
}

func TestTargetDefaults(t *testing.T) {
	require.Equal(t, "255.255.255.255:9", Target("", 0))
	require.Equal(t, "255.255.255.255:9", Target("   ", 9)) // whitespace normalises too
	require.Equal(t, "192.168.1.255:7", Target("192.168.1.255", 7))
}

func atoiMust(t *testing.T, s string) int {
	t.Helper()
	p, err := net.LookupPort("udp", s)
	require.NoError(t, err)
	return p
}
