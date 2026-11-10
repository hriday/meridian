package netcat

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// echoServer starts a TCP echo listener and returns its address.
func echoServer(t *testing.T) string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go io.Copy(c, c)
		}
	}()
	return ln.Addr().String()
}

func TestDialSendRecv(t *testing.T) {
	s, cert, err := Dial(context.Background(), Config{Network: "tcp", Addr: echoServer(t)})
	require.NoError(t, err)
	require.Nil(t, cert) // non-TLS: no cert info
	defer s.Close()

	got := make(chan string, 1)
	s.Start(func(b []byte) { got <- string(b) }, func(error) {})
	_, err = s.Send([]byte("ping"))
	require.NoError(t, err)

	select {
	case b := <-got:
		require.Equal(t, "ping", b)
	case <-time.After(2 * time.Second):
		t.Fatal("no data received")
	}
}

func TestSessionCloseStopsLoop(t *testing.T) {
	s, _, err := Dial(context.Background(), Config{Network: "tcp", Addr: echoServer(t)})
	require.NoError(t, err)

	closed := make(chan error, 1)
	s.Start(func([]byte) {}, func(e error) { closed <- e })
	s.Close()

	select {
	case <-closed: // onClose fired exactly once, no panic
	case <-time.After(2 * time.Second):
		t.Fatal("onClose not called after Close")
	}
}

func TestDialTLSVerifies(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()
	addr := strings.TrimPrefix(srv.URL, "https://")

	// Default verifies → the test server's self-signed cert is rejected.
	_, _, err := Dial(context.Background(), Config{Network: "tcp", Addr: addr, TLS: true})
	require.Error(t, err)

	// Insecure connects and reports the cert as unverified.
	s, cert, err := Dial(context.Background(), Config{Network: "tcp", Addr: addr, TLS: true, Insecure: true})
	require.NoError(t, err)
	require.NotNil(t, cert)
	require.False(t, cert.Verified)
	require.NotEmpty(t, cert.Subject)
	s.Close()
}

func TestHexDump(t *testing.T) {
	out := HexDump([]byte{0x00, 'A', 0x7f, 'z'})
	require.Contains(t, out, "00000000")    // offset
	require.Contains(t, out, "00 41 7f 7a") // hex bytes
	require.Contains(t, out, ".A.z")        // ascii, non-printable as '.'
}
