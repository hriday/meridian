package app

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

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

func TestAppNetcatConnectSendRecv(t *testing.T) {
	a := New()
	got := make(chan string, 1)
	a.ncRecvHook = func(b []byte) { got <- string(b) }

	cert, err := a.NetcatConnect("tcp", echoServer(t), false, false)
	require.NoError(t, err)
	require.False(t, cert.Verified) // non-TLS: zero CertInfo

	require.NoError(t, a.NetcatSend("ping"))
	select {
	case b := <-got:
		require.Equal(t, "ping", b)
	case <-time.After(2 * time.Second):
		t.Fatal("no echoed data")
	}
	a.NetcatClose()
	require.Error(t, a.NetcatSend("after-close")) // session cleared
}
