package app

import (
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"

	ptftp "github.com/pin/tftp/v3"
	"github.com/stretchr/testify/require"
)

func TestTftpStartListStop(t *testing.T) {
	a := New()
	require.NoError(t, a.TftpStart(0))
	defer a.TftpStop()
	port := a.TftpList().Port
	require.NotZero(t, port)

	c, err := ptftp.NewClient(fmt.Sprintf("127.0.0.1:%d", port))
	require.NoError(t, err)
	c.SetTimeout(time.Second)
	rf, err := c.Send("router.cfg", "octet")
	require.NoError(t, err)
	_, err = rf.ReadFrom(bytes.NewReader([]byte("boot system flash\n")))
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		v := a.TftpList()
		return len(v.Files) == 1 && v.Files[0].Name == "router.cfg"
	}, 2*time.Second, 10*time.Millisecond)
	require.Contains(t, a.TftpPreview(a.TftpList().Files[0].ID), "boot system")

	a.TftpStop()
	require.False(t, a.TftpList().Running)
}

func TestTftpStartBindInUse(t *testing.T) {
	held, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	require.NoError(t, err)
	defer held.Close()
	require.Error(t, New().TftpStart(held.LocalAddr().(*net.UDPAddr).Port))
}
