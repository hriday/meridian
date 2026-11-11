package tftp

import (
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"

	ptftp "github.com/pin/tftp/v3"
	"github.com/stretchr/testify/require"
)

func client(t *testing.T, port int) *ptftp.Client {
	t.Helper()
	c, err := ptftp.NewClient(fmt.Sprintf("127.0.0.1:%d", port))
	require.NoError(t, err)
	c.SetTimeout(time.Second)
	return c
}

func TestTftpReceivesFile(t *testing.T) {
	s := New()
	require.NoError(t, s.Start(0))
	defer s.Stop()
	_, port, _ := s.List()
	require.NotZero(t, port)

	cfg := []byte("hostname sw1\ninterface GigabitEthernet0/1\n no shutdown\n")
	rf, err := client(t, port).Send("sw1.cfg", "octet")
	require.NoError(t, err)
	n, err := rf.ReadFrom(bytes.NewReader(cfg))
	require.NoError(t, err)
	require.EqualValues(t, len(cfg), n)

	require.Eventually(t, func() bool {
		_, _, files := s.List()
		return len(files) == 1 && files[0].Name == "sw1.cfg"
	}, 2*time.Second, 10*time.Millisecond)

	_, _, files := s.List()
	require.Contains(t, files[0].Source, "127.0.0.1")
	require.Equal(t, len(cfg), files[0].Size)
	content, ok := s.Content(files[0].ID)
	require.True(t, ok)
	require.Equal(t, cfg, content)
	prev, ok := s.Preview(files[0].ID, 4096)
	require.True(t, ok)
	require.Contains(t, prev, "hostname sw1")
}

func TestTftpDeniesRead(t *testing.T) {
	s := New()
	require.NoError(t, s.Start(0))
	defer s.Stop()
	_, port, _ := s.List()
	wt, err := client(t, port).Receive("anything", "octet")
	if err == nil {
		_, err = wt.WriteTo(&bytes.Buffer{})
	}
	require.Error(t, err) // reads refused
}

func TestTftpCapsSize(t *testing.T) {
	old := maxTftpFile
	maxTftpFile = 100
	defer func() { maxTftpFile = old }()
	s := New()
	require.NoError(t, s.Start(0))
	defer s.Stop()
	_, port, _ := s.List()
	rf, err := client(t, port).Send("big", "octet")
	require.NoError(t, err)
	_, _ = rf.ReadFrom(bytes.NewReader(make([]byte, 500))) // exceeds cap → aborted
	time.Sleep(50 * time.Millisecond)
	_, _, files := s.List()
	require.Empty(t, files) // not stored
}

func TestTftpStopClears(t *testing.T) {
	s := New()
	require.NoError(t, s.Start(0))
	_, port, _ := s.List()
	rf, _ := client(t, port).Send("f", "octet")
	_, _ = rf.ReadFrom(bytes.NewReader([]byte("x")))
	require.Eventually(t, func() bool { _, _, f := s.List(); return len(f) == 1 }, 2*time.Second, 10*time.Millisecond)
	s.Stop()
	running, _, files := s.List()
	require.False(t, running)
	require.Empty(t, files)
}

func TestTftpBindInUse(t *testing.T) {
	held, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	require.NoError(t, err)
	defer held.Close()
	require.Error(t, New().Start(held.LocalAddr().(*net.UDPAddr).Port))
}
