package collector

import (
	"encoding/hex"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// SyslogMsg is one parsed syslog datagram. The sequence number lives on the
// ring's Stamped wrapper, not here.
type SyslogMsg struct {
	T            time.Time `json:"t"`
	Source       string    `json:"source"`
	Facility     int       `json:"facility"`
	Severity     int       `json:"severity"`
	SeverityName string    `json:"severityName"`
	Tag          string    `json:"tag"`
	Message      string    `json:"message"`
}

const maxSyslogMsg = 2048

var severityNames = [8]string{"emerg", "alert", "crit", "err", "warning", "notice", "info", "debug"}

var (
	tsRe  = regexp.MustCompile(`^[A-Z][a-z]{2} [ 0-9]\d \d\d:\d\d:\d\d `)
	tagRe = regexp.MustCompile(`^([A-Za-z0-9._\-/]+)(?:\[\d+\])?:\s?(.*)$`)
)

// ParseSyslog parses an RFC3164-ish datagram best-effort. T is the receive time
// (device clocks drift and RFC3164 carries no year/zone). ok is false only for
// an empty datagram. The source IP is authoritative; the message is bounded and
// hex-rendered if it isn't printable text.
func ParseSyslog(datagram []byte, src netip.Addr) (SyslogMsg, bool) {
	if len(datagram) == 0 {
		return SyslogMsg{}, false
	}
	m := SyslogMsg{T: time.Now(), Source: src.String(), Facility: 1, Severity: 6}

	s := string(datagram)
	// <PRI>
	if strings.HasPrefix(s, "<") {
		if end := strings.IndexByte(s, '>'); end > 1 && end <= 4 {
			if pri, err := strconv.Atoi(s[1:end]); err == nil && pri >= 0 && pri < 1024 {
				m.Facility = pri / 8
				m.Severity = pri % 8
				s = s[end+1:]
			}
		}
	}
	m.SeverityName = severityNames[m.Severity&7]

	rest := strings.TrimSpace(s)
	if loc := tsRe.FindStringIndex(rest); loc != nil { // drop the RFC3164 timestamp
		rest = rest[loc[1]:]
	}
	// Optional "host tag: message"; else "tag: message"; else the whole line.
	tag, msg := "", rest
	if i := strings.IndexByte(rest, ' '); i > 0 {
		if g := tagRe.FindStringSubmatch(rest[i+1:]); g != nil {
			tag, msg = g[1], g[2] // first token was the hostname
		} else if g := tagRe.FindStringSubmatch(rest); g != nil {
			tag, msg = g[1], g[2]
		}
	} else if g := tagRe.FindStringSubmatch(rest); g != nil {
		tag, msg = g[1], g[2]
	}
	m.Tag = tag
	m.Message = clip(msg)
	return m, true
}

// clip bounds and renders a message: it first caps the raw bytes (so a huge
// binary datagram isn't hex-expanded in full), then renders printable text
// as-is or non-text as hex, bounding the result and trimming any partial
// trailing UTF-8 rune left by the byte truncation.
func clip(s string) string {
	if len(s) > maxSyslogMsg {
		s = s[:maxSyslogMsg]
	}
	if !printable(s) {
		s = hex.EncodeToString([]byte(s))
		if len(s) > maxSyslogMsg {
			s = s[:maxSyslogMsg]
		}
		return s // hex is pure ASCII
	}
	for len(s) > 0 && !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s
}

func printable(s string) bool {
	if !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if r == '\t' {
			continue
		}
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
