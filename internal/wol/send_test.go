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
	err := Send(context.Background(), "nope", "127.0.0.1", 9)
	require.Error(t, err)
}

func TestSendDefaults(t *testing.T) {
	require.Equal(t, "255.255.255.255:9", target("", 0))
	require.Equal(t, "192.168.1.255:7", target("192.168.1.255", 7))
}

func atoiMust(t *testing.T, s string) int {
	t.Helper()
	p, err := net.LookupPort("udp", s)
	require.NoError(t, err)
	return p
}
