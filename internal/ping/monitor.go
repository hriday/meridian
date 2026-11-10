package ping

import (
	"context"
	"time"
)

// Monitor pings host continuously, interval apart, until ctx is cancelled,
// streaming each reply through onReply. Unlike Run it resolves host once and
// keeps no history, so it is safe to run indefinitely. It returns
// (available=false, nil) immediately if the first probe reports ICMP
// unavailable, and a non-nil error only if the host can't be resolved.
func Monitor(ctx context.Context, p Pinger, host string, interval, timeout time.Duration, onReply func(Reply)) (bool, error) {
	ip, err := resolveIP(ctx, host)
	if err != nil {
		return false, err
	}
	seq := 0
	for {
		if ctx.Err() != nil {
			return true, nil
		}
		rtt, ok, available := p.Ping(ctx, ip, timeout)
		if seq == 0 && !available {
			return false, nil
		}
		seq++
		r := Reply{Seq: seq, OK: ok}
		if ok {
			r.RTTMillis = float64(rtt) / float64(time.Millisecond)
		}
		if onReply != nil {
			onReply(r)
		}
		if interval > 0 {
			t := time.NewTimer(interval)
			select {
			case <-ctx.Done():
				t.Stop()
				return true, nil
			case <-t.C:
			}
		}
	}
}
