// Package ntp is a minimal SNTP (NTP v4, client mode) query client. It asks a
// time server for the current time and reports stratum, clock offset, and
// round-trip delay — it never sets the system clock. Query does the UDP
// exchange; decode is the pure parser, tested with canned packets.
package ntp

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"strings"
	"time"
)

const ntpEpochOffset = 2208988800 // seconds between 1900-01-01 and 1970-01-01

// Reply is the decoded result of one NTP query.
type Reply struct {
	Server         string        `json:"server"`
	Stratum        int           `json:"stratum"`
	RefID          string        `json:"refId"`
	LeapIndicator  string        `json:"leapIndicator"`
	Precision      float64       `json:"precisionSeconds"`
	RootDelay      time.Duration `json:"rootDelay"`
	RootDispersion time.Duration `json:"rootDispersion"`
	ServerTime     time.Time     `json:"serverTime"`
	Offset         time.Duration `json:"offset"`
	RTT            time.Duration `json:"rtt"`
	KissOfDeath    bool          `json:"kissOfDeath"`
}

// Query sends a client-mode NTP request to server (a host or host:port; port
// 123 is assumed if absent) and returns the decoded Reply. The deadline comes
// from ctx, else a 5s default.
func Query(ctx context.Context, server string) (Reply, error) {
	addr := withPort(server, "123")
	var d net.Dialer
	conn, err := d.DialContext(ctx, "udp", addr)
	if err != nil {
		return Reply{}, err
	}
	defer conn.Close()

	deadline := time.Now().Add(5 * time.Second)
	if dl, ok := ctx.Deadline(); ok {
		deadline = dl
	}
	_ = conn.SetDeadline(deadline)

	req := make([]byte, 48)
	req[0] = (4 << 3) | 3 // LI=0, VN=4, Mode=3 (client)
	t1 := time.Now()
	putNTPTime(req[40:48], t1) // transmit timestamp

	if _, err := conn.Write(req); err != nil {
		return Reply{}, err
	}
	resp := make([]byte, 48)
	n, err := conn.Read(resp)
	if err != nil {
		return Reply{}, err
	}
	t4 := time.Now()
	r, err := decode(resp[:n], t1, t4)
	if err != nil {
		return Reply{}, err
	}
	r.Server = server
	return r, nil
}

// decode parses a 48-byte NTP reply and computes offset/RTT from the four
// timestamps (t1 request-sent, t4 reply-received are measured locally; t2
// receive and t3 transmit come from the packet).
func decode(resp []byte, t1, t4 time.Time) (Reply, error) {
	if len(resp) < 48 {
		return Reply{}, fmt.Errorf("short NTP reply: %d bytes", len(resp))
	}
	if mode := resp[0] & 0x07; mode != 4 {
		return Reply{}, fmt.Errorf("not an NTP server reply (mode %d)", mode)
	}
	// The origin timestamp must echo the transmit timestamp we sent (putNTPTime(t1)),
	// which rejects stale, duplicated, or spoofed replies (RFC 4330).
	var wantOrigin [8]byte
	putNTPTime(wantOrigin[:], t1)
	if !bytes.Equal(resp[24:32], wantOrigin[:]) {
		return Reply{}, fmt.Errorf("origin timestamp mismatch (stale or spoofed reply)")
	}
	stratum := int(resp[1])
	r := Reply{
		Stratum:        stratum,
		LeapIndicator:  leapText(resp[0] >> 6),
		Precision:      math.Pow(2, float64(int8(resp[3]))),
		RootDelay:      ntpShort(resp[4:8]),
		RootDispersion: ntpShort(resp[8:12]),
		RefID:          refIDText(resp[12:16], stratum),
		KissOfDeath:    stratum == 0,
	}
	t2 := ntpTime(resp[32:40])
	t3 := ntpTime(resp[40:48])
	if t2.IsZero() || t3.IsZero() {
		return Reply{}, fmt.Errorf("server returned zero timestamps")
	}
	r.ServerTime = t3
	r.Offset = ((t2.Sub(t1)) + (t3.Sub(t4))) / 2
	r.RTT = (t4.Sub(t1)) - (t3.Sub(t2))
	return r, nil
}

func leapText(li byte) string {
	switch li {
	case 0:
		return "none"
	case 1:
		return "last minute has 61 seconds"
	case 2:
		return "last minute has 59 seconds"
	default:
		return "unsynchronized"
	}
}

func refIDText(b []byte, stratum int) string {
	if stratum <= 1 {
		return strings.TrimRight(string(b), "\x00")
	}
	return fmt.Sprintf("%d.%d.%d.%d", b[0], b[1], b[2], b[3])
}

func ntpTime(b []byte) time.Time {
	sec := binary.BigEndian.Uint32(b[0:4])
	frac := binary.BigEndian.Uint32(b[4:8])
	if sec == 0 && frac == 0 {
		return time.Time{}
	}
	nsec := (int64(frac) * 1e9) >> 32
	return time.Unix(int64(sec)-ntpEpochOffset, nsec).UTC()
}

func ntpShort(b []byte) time.Duration {
	sec := binary.BigEndian.Uint16(b[0:2])
	frac := binary.BigEndian.Uint16(b[2:4])
	return time.Duration(sec)*time.Second + time.Duration(int64(frac)*1e9/65536)*time.Nanosecond
}

func putNTPTime(b []byte, t time.Time) {
	binary.BigEndian.PutUint32(b[0:4], uint32(t.Unix()+ntpEpochOffset))
	binary.BigEndian.PutUint32(b[4:8], uint32((int64(t.Nanosecond())<<32)/1e9))
}

func withPort(server, port string) string {
	if _, _, err := net.SplitHostPort(server); err == nil {
		return server // already has a port
	}
	return net.JoinHostPort(server, port)
}
