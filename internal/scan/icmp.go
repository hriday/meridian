package scan

import (
	"context"
	"net/netip"
	"time"

	"github.com/hriday/meridian/internal/trace"
)

// ICMPPinger pings hosts with unprivileged ICMP echo, reusing the traceroute
// prober. A fresh socket is opened per ping so the sweep can probe concurrently;
// availability is checked once at construction.
type ICMPPinger struct {
	v4ok, v6ok bool
}

var _ Pinger = (*ICMPPinger)(nil)

// NewICMPPinger probes whether unprivileged ICMP sockets can be opened.
func NewICMPPinger() *ICMPPinger {
	p := &ICMPPinger{}
	if c, err := trace.NewICMPProber("ip4"); err == nil {
		c.Close()
		p.v4ok = true
	}
	if c, err := trace.NewICMPProber("ip6"); err == nil {
		c.Close()
		p.v6ok = true
	}
	return p
}

func (p *ICMPPinger) Ping(ctx context.Context, ip netip.Addr, timeout time.Duration) (time.Duration, bool, bool) {
	network, available := "ip4", p.v4ok
	if ip.Is6() {
		network, available = "ip6", p.v6ok
	}
	if !available {
		return 0, false, false
	}
	pr, err := trace.NewICMPProber(network)
	if err != nil {
		return 0, false, false
	}
	defer pr.Close()
	_, rtt, reached, perr := pr.Probe(ctx, ip, 64, 1, timeout)
	if perr != nil {
		return 0, false, true // ICMP is available, this probe just errored
	}
	return rtt, reached, true
}
