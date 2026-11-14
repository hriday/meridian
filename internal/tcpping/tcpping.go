// Package tcpping measures TCP connect latency and loss to a host:port — a
// "ping" that works where ICMP is filtered (which is exactly the paths that
// matter when a firewall is in the way), and needs no elevated privileges. It
// times the TCP handshake (SYN → SYN-ACK → connected) to each attempt; a
// refused or timed-out connection counts as a loss.
package tcpping

import (
	"context"
	"fmt"
	"math"
	"time"
)

// Probe is one connect attempt.
type Probe struct {
	Seq   int     `json:"seq"`
	RTTMs float64 `json:"rttMs"`
	OK    bool    `json:"ok"`
	Err   string  `json:"err,omitempty"`
}

// Result is the summary plus every probe.
type Result struct {
	Host     string  `json:"host"`
	Port     int     `json:"port"`
	Address  string  `json:"address"` // resolved remote address actually reached
	Sent     int     `json:"sent"`
	Received int     `json:"received"`
	LossPct  float64 `json:"lossPct"`
	MinMs    float64 `json:"minMs"`
	AvgMs    float64 `json:"avgMs"`
	MaxMs    float64 `json:"maxMs"`
	JitterMs float64 `json:"jitterMs"`
	Verdict  string  `json:"verdict"`
	Probes   []Probe `json:"probes"`
}

// Connector establishes a TCP connection to addr and returns the time taken
// and the resolved remote address. Injected so tests run without a network.
type Connector func(ctx context.Context, addr string) (time.Duration, string, error)

// Ping makes count connect attempts to host:port, interval apart, and
// summarizes the latency and loss.
func Ping(ctx context.Context, conn Connector, host string, port, count int, interval time.Duration) (Result, error) {
	if count < 1 {
		return Result{}, fmt.Errorf("count must be at least 1")
	}
	addr := fmt.Sprintf("%s:%d", host, port)
	r := Result{Host: host, Port: port}
	for i := 0; i < count; i++ {
		if i > 0 && interval > 0 {
			select {
			case <-ctx.Done():
				return r, ctx.Err()
			case <-time.After(interval):
			}
		}
		d, remote, err := conn(ctx, addr)
		p := Probe{Seq: i + 1}
		if err != nil {
			p.OK = false
			p.Err = err.Error()
		} else {
			p.OK = true
			p.RTTMs = round2(float64(d) / float64(time.Millisecond))
			if remote != "" {
				r.Address = remote
			}
		}
		r.Probes = append(r.Probes, p)
	}
	summarize(&r)
	return r, nil
}

func summarize(r *Result) {
	r.Sent = len(r.Probes)
	var rtts []float64
	for _, p := range r.Probes {
		if p.OK {
			rtts = append(rtts, p.RTTMs)
		}
	}
	r.Received = len(rtts)
	if r.Sent > 0 {
		r.LossPct = round2(float64(r.Sent-r.Received) / float64(r.Sent) * 100)
	}
	if r.Received > 0 {
		min, max, sum := rtts[0], rtts[0], 0.0
		for _, v := range rtts {
			if v < min {
				min = v
			}
			if v > max {
				max = v
			}
			sum += v
		}
		r.MinMs, r.MaxMs = min, max
		r.AvgMs = round2(sum / float64(r.Received))
		r.JitterMs = round2(meanAbsDiff(rtts))
	}
	r.Verdict = verdict(r)
}

// meanAbsDiff is the mean absolute difference between consecutive samples —
// a simple jitter estimate.
func meanAbsDiff(xs []float64) float64 {
	if len(xs) < 2 {
		return 0
	}
	sum := 0.0
	for i := 1; i < len(xs); i++ {
		sum += math.Abs(xs[i] - xs[i-1])
	}
	return sum / float64(len(xs)-1)
}

func verdict(r *Result) string {
	switch {
	case r.Received == 0:
		return "No TCP connection on any attempt — the port is filtered or closed, or the host is unreachable."
	case r.LossPct > 0:
		return fmt.Sprintf("Port open, but %.0f%% of connections failed — intermittent reachability (avg %.1f ms).", r.LossPct, r.AvgMs)
	case r.AvgMs < 30:
		return fmt.Sprintf("Port open and fast — avg %.1f ms, no loss.", r.AvgMs)
	case r.AvgMs < 150:
		return fmt.Sprintf("Port open — avg %.1f ms, no loss.", r.AvgMs)
	default:
		return fmt.Sprintf("Port open but slow to connect — avg %.1f ms, no loss.", r.AvgMs)
	}
}

func round2(f float64) float64 { return math.Round(f*100) / 100 }
