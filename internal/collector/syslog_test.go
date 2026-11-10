package collector

import (
	"net/netip"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var src = netip.MustParseAddr("10.0.0.3")

func TestParseSyslogPRI(t *testing.T) {
	m, ok := ParseSyslog([]byte("<34>Oct 11 22:14:15 host app: su root failed"), src)
	require.True(t, ok)
	require.Equal(t, 4, m.Facility) // 34/8
	require.Equal(t, 2, m.Severity) // 34%8
	require.Equal(t, "crit", m.SeverityName)
	require.Contains(t, m.Message, "su root failed")
	require.Equal(t, "10.0.0.3", m.Source)
	require.False(t, m.T.IsZero())
}

func TestParseSyslogSeverityNames(t *testing.T) {
	want := []string{"emerg", "alert", "crit", "err", "warning", "notice", "info", "debug"}
	for sev, name := range want {
		m, ok := ParseSyslog([]byte("<"+itoaSmall(sev)+">a message"), src)
		require.True(t, ok)
		require.Equal(t, sev, m.Severity)
		require.Equal(t, name, m.SeverityName)
	}
}

func TestParseSyslogNoPRI(t *testing.T) {
	m, ok := ParseSyslog([]byte("just a line, no priority"), src)
	require.True(t, ok)
	require.Equal(t, 6, m.Severity) // default info
	require.Contains(t, m.Message, "just a line")
}

func TestParseSyslogTruncatesAndHex(t *testing.T) {
	big := "<13>" + strings.Repeat("a", 5000)
	m, ok := ParseSyslog([]byte(big), src)
	require.True(t, ok)
	require.LessOrEqual(t, len(m.Message), 2048)

	// a binary payload after the PRI → hex, no raw NUL
	bin := append([]byte("<13>"), 0x00, 0xde, 0xad)
	m2, ok := ParseSyslog(bin, src)
	require.True(t, ok)
	require.NotContains(t, m2.Message, "\x00")
}

func TestParseSyslogEmpty(t *testing.T) {
	_, ok := ParseSyslog(nil, src)
	require.False(t, ok)
	_, ok = ParseSyslog([]byte(""), src)
	require.False(t, ok)
}

func itoaSmall(n int) string { return string(rune('0' + n)) }
