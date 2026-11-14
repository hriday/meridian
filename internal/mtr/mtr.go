// Package mtr turns Meridian's one-shot traceroute into a continuous
// path-quality monitor: it sweeps the route repeatedly and accumulates
// per-hop statistics (loss, last/best/worst/avg RTT), the way the classic
// `mtr` tool does. It is live-only — nothing is persisted; stop and the
// numbers are gone.
package mtr

import (
	"context"
	"sort"
	"time"

	"github.com/hriday/meridian/internal/trace"
)

// HopStat is the running statistics for one hop (one TTL) across every round
// seen so far.
type HopStat struct {
	TTL     int           `json:"ttl"`
	Addr    string        `json:"addr"`
	Host    string        `json:"host"`
	ASN     string        `json:"asn"`
	ASOwner string        `json:"asOwner"`
	Sent    int           `json:"sent"`
	Recv    int           `json:"recv"`
	LossPct float64       `json:"lossPct"`
	Last    time.Duration `json:"last"`
	Best    time.Duration `json:"best"`
	Worst   time.Duration `json:"worst"`
	Avg     time.Duration `json:"avg"`
}

// Update is a snapshot of the whole path after one completed round.
type Update struct {
	Round   int       `json:"round"`
	Target  string    `json:"target"`
	Addr    string    `json:"addr"`
	Hops    []HopStat `json:"hops"`
	Reached bool      `json:"reached"`
}

// Sweeper runs one full traceroute sweep. tracerun.Run satisfies it, and tests
// supply a scripted fake.
type Sweeper func(ctx context.Context, target string, useV6 bool, onHop func(trace.Hop)) (trace.Result, error)

// Monitor sweeps target forever, accumulating per-hop stats and emitting a
// full snapshot through onUpdate after each round, until ctx is cancelled.
//
// The first sweep's error is returned (a bad target should fail fast); later
// sweep errors are transient (a single lost round) and are skipped without
// tearing the monitor down. A cancelled context ends the monitor cleanly with
// a nil error.
func Monitor(ctx context.Context, sweep Sweeper, target string, useV6 bool, interval time.Duration, onUpdate func(Update)) error {
	acc := map[int]*HopStat{}
	var order []int
	round := 0
	var lastAddr string
	var lastReached bool

	for {
		round++
		res, err := sweep(ctx, target, useV6, func(trace.Hop) {})
		if err != nil {
			if round == 1 {
				return err
			}
			// transient round (a single lost sweep): emit the unchanged
			// snapshot so the caller still ticks, keeping the last-good
			// destination/reached instead of blanking them.
		} else {
			lastAddr = res.Addr
			lastReached = res.Reached
			for _, h := range res.Hops {
				s := acc[h.TTL]
				if s == nil {
					s = &HopStat{TTL: h.TTL}
					acc[h.TTL] = s
					order = append(order, h.TTL)
				}
				s.Sent++
				if h.Addr != "" {
					s.Addr = h.Addr
				}
				if h.Host != "" {
					s.Host = h.Host
				}
				if h.ASN != "" {
					s.ASN = h.ASN
				}
				if h.ASOwner != "" {
					s.ASOwner = h.ASOwner
				}
				if !h.Timeout && len(h.RTTs) > 0 {
					rtt := minDur(h.RTTs)
					s.Recv++
					s.Last = rtt
					if s.Recv == 1 || rtt < s.Best {
						s.Best = rtt
					}
					if rtt > s.Worst {
						s.Worst = rtt
					}
					// running mean
					s.Avg += time.Duration((int64(rtt) - int64(s.Avg)) / int64(s.Recv))
				}
				s.LossPct = float64(s.Sent-s.Recv) / float64(s.Sent) * 100
			}
		}

		sort.Ints(order)
		hops := make([]HopStat, 0, len(order))
		for _, ttl := range order {
			hops = append(hops, *acc[ttl])
		}
		onUpdate(Update{Round: round, Target: target, Addr: lastAddr, Hops: hops, Reached: lastReached})

		if interval <= 0 {
			select {
			case <-ctx.Done():
				return nil
			default:
			}
			continue
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(interval):
		}
	}
}

func minDur(ds []time.Duration) time.Duration {
	m := ds[0]
	for _, d := range ds[1:] {
		if d < m {
			m = d
		}
	}
	return m
}
