// Package ping measures how well one host responds to ICMP echo: it pings a
// host a bounded number of times and reports min/avg/max/stddev round-trip
// latency, jitter, and packet loss, streaming each reply as it arrives. It
// reuses the unprivileged ICMP prober (scan.Pinger) that the ping sweep uses, so
// it introduces no new privilege. summarize is pure; Run drives the pinger. The
// CLI and Wails controller are thin adapters over Run.
package ping

import "math"

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
