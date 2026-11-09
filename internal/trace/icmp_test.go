package trace

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

func TestParseICMPReplyEchoReply(t *testing.T) {
	wm := icmp.Message{Type: ipv4.ICMPTypeEchoReply, Code: 0, Body: &icmp.Echo{ID: 1, Seq: 42, Data: []byte("x")}}
	b, err := wm.Marshal(nil)
	require.NoError(t, err)

	kind, matched, perr := parseICMPReply(protoICMPv4, b, 42)
	require.NoError(t, perr)
	require.True(t, matched)
	require.Equal(t, "echo-reply", kind)

	_, m2, _ := parseICMPReply(protoICMPv4, b, 7) // different seq
	require.False(t, m2)
}

func TestParseICMPReplyTimeExceeded(t *testing.T) {
	inner := icmp.Message{Type: ipv4.ICMPTypeEcho, Code: 0, Body: &icmp.Echo{ID: 1, Seq: 42, Data: []byte("x")}}
	ib, err := inner.Marshal(nil)
	require.NoError(t, err)
	iph := make([]byte, 20)
	iph[0] = 0x45 // IPv4, IHL=5 (20 bytes)
	data := append(iph, ib...)

	te := icmp.Message{Type: ipv4.ICMPTypeTimeExceeded, Code: 0, Body: &icmp.TimeExceeded{Data: data}}
	b, err := te.Marshal(nil)
	require.NoError(t, err)

	kind, matched, perr := parseICMPReply(protoICMPv4, b, 42)
	require.NoError(t, perr)
	require.True(t, matched)
	require.Equal(t, "time-exceeded", kind)
}

func TestNewICMPProberErrorIsClear(t *testing.T) {
	_, err := NewICMPProber("nope")
	require.Error(t, err)
	require.Contains(t, err.Error(), "ip4")
}
