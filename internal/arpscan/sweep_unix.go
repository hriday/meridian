//go:build darwin || linux

package arpscan

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/hriday/meridian/internal/l2"
)

const arpSupported = true

// sweepSubnet broadcasts an ARP request for every host via a raw L2 socket and
// collects the replies. Needs elevated privileges (raw L2).
func sweepSubnet(ctx context.Context, iface string, srcMAC net.HardwareAddr, srcIP net.IP, ipnet *net.IPNet, window time.Duration, maxHosts int) (map[string]string, error) {
	c, err := l2.Open(iface)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	found := map[string]string{}
	var mu sync.Mutex
	rctx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for rctx.Err() == nil {
			frames, err := c.Read()
			if err != nil {
				if errors.Is(err, l2.ErrReadTimeout) {
					continue
				}
				return
			}
			for _, f := range frames {
				if ip, m, ok := ParseReply(f); ok {
					mu.Lock()
					found[ip] = m
					mu.Unlock()
				}
			}
		}
	}()

	for _, t := range hostsIn(ipnet, maxHosts) {
		if t.Equal(srcIP) {
			continue
		}
		_ = c.Write(BuildRequest(srcMAC, srcIP, t))
		if ctx.Err() != nil {
			break
		}
		time.Sleep(time.Millisecond) // light pacing so we don't flood
	}

	select {
	case <-time.After(window):
	case <-ctx.Done():
	}
	cancel()
	<-done

	mu.Lock()
	defer mu.Unlock()
	out := make(map[string]string, len(found))
	for k, v := range found {
		out[k] = v
	}
	return out, nil
}
