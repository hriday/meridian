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
