package app

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSyslogStartPollStop(t *testing.T) {
	a := New()
	require.NoError(t, a.SyslogStart(0)) // 0 → free port
	defer a.SyslogStop()
	port := a.SyslogPoll(0).Port
	require.NotZero(t, port)

	c, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: port})
	require.NoError(t, err)
	_, err = c.Write([]byte("<11>hello from device"))
	require.NoError(t, err)
	c.Close()

	require.Eventually(t, func() bool {
		return len(a.SyslogPoll(0).Items) >= 1
	}, 2*time.Second, 10*time.Millisecond)
	require.Equal(t, "err", a.SyslogPoll(0).Items[0].Val.SeverityName) // <11> → severity 3

	a.SyslogStop()
	require.False(t, a.SyslogPoll(0).Running)
}

func TestSyslogStartBindInUse(t *testing.T) {
	held, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	require.NoError(t, err)
	defer held.Close()
	port := held.LocalAddr().(*net.UDPAddr).Port
	require.Error(t, New().SyslogStart(port))
}

func TestSyslogRestartEpoch(t *testing.T) {
	a := New()
	require.NoError(t, a.SyslogStart(0))
	defer a.SyslogStop()
	e1 := a.SyslogPoll(0).Epoch
	require.NoError(t, a.SyslogStart(0))
	require.Greater(t, a.SyslogPoll(0).Epoch, e1)
}
