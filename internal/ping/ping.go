// Package ping measures how well one host responds to ICMP echo: it pings a
// host a bounded number of times and reports min/avg/max/stddev round-trip
// latency, jitter, and packet loss, streaming each reply as it arrives. It
// reuses the unprivileged ICMP prober (scan.Pinger) that the ping sweep uses, so
// it introduces no new privilege. summarize is pure; Run drives the pinger. The
// CLI and Wails controller are thin adapters over Run.
package ping

import (
	"context"
	"math"
	"net"
	"net/netip"
	"time"
)

// Pinger sends one ICMP echo. It matches scan.Pinger so the real unprivileged
// prober can be passed directly, and a fake can be used in tests.
type Pinger interface {
	Ping(ctx context.Context, ip netip.Addr, timeout time.Duration) (rtt time.Duration, ok, available bool)
}

// resolveIP turns a host (IP literal or name) into an address. Package var so
// tests stay offline. An IP literal is used as-is; a name resolves via the
// system resolver (first address).
var resolveIP = func(ctx context.Context, host string) (netip.Addr, error) {
	if a, err := netip.ParseAddr(host); err == nil {
		return a, nil
	}
	addrs, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return netip.Addr{}, err
	}
	if len(addrs) == 0 {
		return netip.Addr{}, &net.DNSError{Err: "no addresses", Name: host}
	}
	return addrs[0].Unmap(), nil
}

// Run resolves host and pings it count times, interval apart, each with the
// given per-ping timeout, streaming each reply through onReply (may be nil). If
// the first probe reports ICMP unavailable, it returns immediately with
// Available=false rather than looping. Context cancellation ends the run early
// and returns the partial stats gathered.
func Run(ctx context.Context, p Pinger, host string, count int, interval, timeout time.Duration, onReply func(Reply)) (Stats, error) {
	if count <= 0 {
		count = 5
	}
	ip, err := resolveIP(ctx, host)
	if err != nil {
		return Stats{Host: host}, err
	}
	ipStr := ip.String()

	replies := make([]Reply, 0, count)
	for i := 0; i < count; i++ {
		if ctx.Err() != nil {
			break
		}
		rtt, okReply, available := p.Ping(ctx, ip, timeout)
		if i == 0 && !available {
			return summarize(host, ipStr, nil, false), nil
		}
		r := Reply{Seq: i + 1, OK: okReply}
		if okReply {
			r.RTTMillis = float64(rtt) / float64(time.Millisecond)
		}
		replies = append(replies, r)
		if onReply != nil {
			onReply(r)
		}
		if i < count-1 && interval > 0 {
			t := time.NewTimer(interval)
			select {
			case <-ctx.Done():
				t.Stop()
				return summarize(host, ipStr, replies, true), nil
			case <-t.C:
			}
		}
	}
	return summarize(host, ipStr, replies, true), nil
}

// Reply is one echo attempt: its sequence number, round-trip time in
// milliseconds (meaningful only when OK), and whether a reply came back.
type Reply struct {
	Seq       int     `json:"seq"`
	RTTMillis float64 `json:"rttMillis"`
	OK        bool    `json:"ok"`
}

// Stats is the aggregate result of a ping run.
type Stats struct {
	Host      string  `json:"host"`
	IP        string  `json:"ip"`
	Sent      int     `json:"sent"`
	Received  int     `json:"received"`
	LossPct   float64 `json:"lossPct"`
	MinMs     float64 `json:"minMs"`
	AvgMs     float64 `json:"avgMs"`
	MaxMs     float64 `json:"maxMs"`
	StddevMs  float64 `json:"stddevMs"`
	JitterMs  float64 `json:"jitterMs"`
	Available bool    `json:"available"`
	Replies   []Reply `json:"replies"`
}

// summarize computes the aggregate statistics over the replies. Loss is the
// fraction that didn't come back; the latency figures are over the ones that
// did. Jitter is the mean absolute difference between consecutive received RTTs.
// Zero-received and single-reply cases yield zeros, never NaN.
func summarize(host, ip string, replies []Reply, available bool) Stats {
	s := Stats{Host: host, IP: ip, Sent: len(replies), Available: available, Replies: replies}

	var rtts []float64
	for _, r := range replies {
		if r.OK {
			rtts = append(rtts, r.RTTMillis)
		}
	}
	s.Received = len(rtts)
	if s.Sent > 0 {
		s.LossPct = float64(s.Sent-s.Received) / float64(s.Sent) * 100
	}
	if s.Received == 0 {
		return s
	}

	s.MinMs, s.MaxMs = rtts[0], rtts[0]
	var sum float64
	for _, v := range rtts {
		sum += v
		if v < s.MinMs {
			s.MinMs = v
		}
		if v > s.MaxMs {
			s.MaxMs = v
		}
	}
	s.AvgMs = sum / float64(s.Received)

	var sq float64
	for _, v := range rtts {
		d := v - s.AvgMs
		sq += d * d
	}
	s.StddevMs = math.Sqrt(sq / float64(s.Received))

	if s.Received > 1 {
		var jsum float64
		for i := 1; i < len(rtts); i++ {
			jsum += math.Abs(rtts[i] - rtts[i-1])
		}
		s.JitterMs = jsum / float64(len(rtts)-1)
	}
	return s
}
