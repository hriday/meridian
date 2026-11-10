package netcat

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestListenTCPAccept(t *testing.T) {
	l, err := Listen(context.Background(), Config{Network: "tcp", Addr: "127.0.0.1:0"})
	require.NoError(t, err)
	defer l.Close()

	go func() {
		c, _, derr := Dial(context.Background(), Config{Network: "tcp", Addr: l.Addr()})
		if derr == nil {
			c.Send([]byte("hi"))
		}
	}()

	sess, err := l.Accept()
	require.NoError(t, err)
	defer sess.Close()

	got := make(chan string, 1)
	sess.Start(func(b []byte) { got <- string(b) }, func(error) {})
	select {
	case b := <-got:
		require.Equal(t, "hi", b)
	case <-time.After(2 * time.Second):
		t.Fatal("no data from accepted TCP peer")
	}
}

func TestListenUDPAccept(t *testing.T) {
	l, err := Listen(context.Background(), Config{Network: "udp", Addr: "127.0.0.1:0"})
	require.NoError(t, err)
	defer l.Close()

	peer, err := net.Dial("udp", l.Addr())
	require.NoError(t, err)
	defer peer.Close()
	_, err = peer.Write([]byte("hello"))
	require.NoError(t, err)

	sess, err := l.Accept()
	require.NoError(t, err)

	got := make(chan string, 1)
	sess.Start(func(b []byte) { got <- string(b) }, func(error) {})
	select {
	case b := <-got:
		require.Equal(t, "hello", b) // first datagram delivered
	case <-time.After(2 * time.Second):
		t.Fatal("first UDP datagram not delivered")
	}

	// a reply goes back to that peer
	_, err = sess.Send([]byte("world"))
	require.NoError(t, err)
	peer.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 16)
	n, err := peer.Read(buf)
	require.NoError(t, err)
	require.Equal(t, "world", string(buf[:n]))
}

// A first datagram larger than the read buffer must be delivered across reads,
// not truncated. (UDP max-datagram limits make this unreachable via a real
// socket on macOS, so exercise the adapter's retention logic directly.)
func TestUDPConnReadRetainsRemainder(t *testing.T) {
	u := &udpConn{first: []byte("hello world")}
	b := make([]byte, 5)

	n, err := u.Read(b)
	require.NoError(t, err)
	require.Equal(t, "hello", string(b[:n]))

	n, err = u.Read(b)
	require.NoError(t, err)
	require.Equal(t, " worl", string(b[:n]))

	n, err = u.Read(b)
	require.NoError(t, err)
	require.Equal(t, "d", string(b[:n])) // tail kept, nothing dropped
	require.Empty(t, u.first)
}

func TestListenCloseUnblocksAccept(t *testing.T) {
	l, err := Listen(context.Background(), Config{Network: "tcp", Addr: "127.0.0.1:0"})
	require.NoError(t, err)

	done := make(chan error, 1)
	go func() { _, aerr := l.Accept(); done <- aerr }()
	time.Sleep(50 * time.Millisecond)
	l.Close()

	select {
	case aerr := <-done:
		require.Error(t, aerr) // Accept unblocks with an error, no hang
	case <-time.After(2 * time.Second):
		t.Fatal("Accept did not unblock after Close")
	}
}
