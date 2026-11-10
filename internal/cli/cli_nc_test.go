package cli

import (
	"bytes"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func echoServerCLI(t *testing.T) string {
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

func TestRunNCConnect(t *testing.T) {
	host, port, _ := net.SplitHostPort(echoServerCLI(t))

	oldIn, oldGrace := ncStdin, ncGrace
	t.Cleanup(func() { ncStdin, ncGrace = oldIn, oldGrace })
	ncStdin = strings.NewReader("hi\n")
	ncGrace = 300 * time.Millisecond

	var out, errb bytes.Buffer
	code := Run([]string{"nc", host, port}, &out, &errb)
	require.Equal(t, 0, code, errb.String())
	require.Contains(t, out.String(), "hi") // echoed back
}

func TestRunNCNeedsHostPort(t *testing.T) {
	var out, errb bytes.Buffer
	require.Equal(t, 2, Run([]string{"nc", "127.0.0.1"}, &out, &errb))
}

type blockingReader struct{}

func (blockingReader) Read([]byte) (int, error) { select {} } // never returns

func TestRunNCExitsOnRemoteClose(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() })
	go func() {
		if c, err := ln.Accept(); err == nil {
			c.Close() // accept then immediately close
		}
	}()
	host, port, _ := net.SplitHostPort(ln.Addr().String())

	oldIn := ncStdin
	t.Cleanup(func() { ncStdin = oldIn })
	ncStdin = blockingReader{} // stdin never EOFs, so only remote-close can end it

	done := make(chan int, 1)
	go func() {
		var o, e bytes.Buffer
		done <- Run([]string{"nc", host, port}, &o, &e)
	}()
	select {
	case <-done: // exited on remote close, did not hang
	case <-time.After(3 * time.Second):
		t.Fatal("nc hung after the remote closed the connection")
	}
}
