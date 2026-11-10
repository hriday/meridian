package app

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWakeOnLANMethod(t *testing.T) {
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = pc.Close() })
	_, portStr, _ := net.SplitHostPort(pc.LocalAddr().String())
	port, _ := net.LookupPort("udp", portStr)

	got := make(chan int, 1)
	go func() {
		buf := make([]byte, 256)
		_ = pc.SetReadDeadline(time.Now().Add(3 * time.Second))
		n, _, _ := pc.ReadFrom(buf)
		got <- n
	}()

	msg, err := New().WakeOnLAN("01:02:03:04:05:06", "127.0.0.1", port)
	require.NoError(t, err)
	require.NotEmpty(t, msg)

	select {
	case n := <-got:
		require.Equal(t, 102, n)
	case <-time.After(3 * time.Second):
		t.Fatal("no datagram received")
	}
}

func TestWakeOnLANMethodBadMAC(t *testing.T) {
	_, err := New().WakeOnLAN("nope", "127.0.0.1", 9)
	require.Error(t, err)
}
