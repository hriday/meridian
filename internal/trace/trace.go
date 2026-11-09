package trace

import (
	"context"
	"net/netip"
	"time"
)

// Prober sends one probe to dst at the given TTL and sequence, returning the
// responding address, the round-trip time, and whether the reply came from the
// destination itself (reached). A timed-out probe returns an invalid address
// with a nil error; err is reserved for unrecoverable socket failures.
type Prober interface {
	Probe(ctx context.Context, dst netip.Addr, ttl, seq int, timeout time.Duration) (from netip.Addr, rtt time.Duration, reached bool, err error)
	Close() error
}

// Options tune a trace. Zero values take the defaults below.
type Options struct {
	MaxHops int
	Probes  int
	Timeout time.Duration
}

func (o Options) withDefaults() Options {
	if o.MaxHops <= 0 {
		o.MaxHops = 30
	}
	if o.Probes <= 0 {
		o.Probes = 3
	}
	if o.Timeout <= 0 {
		o.Timeout = 1500 * time.Millisecond
	}
	return o
}

// Run drives the trace: for each TTL from 1 up to MaxHops it sends Probes
// probes, builds a Hop, emits it via onHop, and stops after the hop whose
// reply reached the destination. It returns the assembled Result.
func Run(ctx context.Context, dst netip.Addr, p Prober, opts Options, onHop func(Hop)) (Result, error) {
	opts = opts.withDefaults()
	start := time.Now()
	r := Result{Addr: dst.String()}
	seq := 0

	for ttl := 1; ttl <= opts.MaxHops; ttl++ {
		if err := ctx.Err(); err != nil {
			return r, err
		}
		hop := Hop{TTL: ttl, RTTs: make([]time.Duration, 0, opts.Probes)}
		var hopAddr netip.Addr
		reached := false

		for i := 0; i < opts.Probes; i++ {
			if err := ctx.Err(); err != nil {
				return r, err
			}
			from, rtt, got, err := p.Probe(ctx, dst, ttl, seq, opts.Timeout)
			seq++
			if err != nil {
				return r, err
			}
			if from.IsValid() {
				hop.RTTs = append(hop.RTTs, rtt)
				if !hopAddr.IsValid() {
					hopAddr = from
				}
				if got {
					reached = true
				}
			} else {
				hop.RTTs = append(hop.RTTs, 0) // timed-out probe
			}
		}

		if hopAddr.IsValid() {
			hop.Addr = hopAddr.String()
		} else {
			hop.Timeout = true
		}
		r.Hops = append(r.Hops, hop)
		if onHop != nil {
			onHop(hop)
		}
		if reached {
			r.Reached = true
			break
		}
	}

	r.Elapsed = time.Since(start)
	return r, nil
}
