package snmp

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// communityProbe tests read access for one community via a single sysDescr GET.
// It is a package var so tests can inject a fake without touching the network.
var communityProbe = func(device, community string, timeout time.Duration) bool {
	// No retry: a wrong community is silent, so one timeout is enough — a
	// retransmit would double a brute audit's time and packet count (same reason
	// the discovery sweep dials with 0 retries).
	c, err := dialRetries(device, community, timeout, 0)
	if err != nil {
		return false
	}
	defer c.Close()
	pdus, err := c.Get([]string{oidSysDescr + ".0"})
	return err == nil && len(pdus) > 0 && pdus[0].Value != nil
}

// BruteCommunities tries each community string against device over SNMPv2c and
// returns those that grant read access (a successful sysDescr GET), in input
// order with duplicates removed. This is an authorization-gated audit tool for
// your OWN gear: it reads only sysDescr and never writes. The caller is
// responsible for confirming authorization before calling.
func BruteCommunities(ctx context.Context, device string, communities []string, timeout time.Duration, concurrency int) ([]string, error) {
	device = strings.TrimSpace(device)
	if device == "" {
		return nil, fmt.Errorf("enter the device IP or hostname")
	}
	seen := map[string]bool{}
	var list []string
	for _, c := range communities {
		c = strings.TrimSpace(c)
		if c == "" || seen[c] {
			continue
		}
		seen[c] = true
		list = append(list, c)
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("no community strings to try")
	}
	if concurrency < 1 {
		concurrency = 1
	}
	if timeout <= 0 {
		timeout = time.Second
	}

	hits := make([]bool, len(list))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for i, comm := range list {
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, comm string) {
			defer wg.Done()
			defer func() { <-sem }()
			if ctx.Err() != nil {
				return
			}
			if communityProbe(device, comm, timeout) {
				hits[i] = true
			}
		}(i, comm)
	}
	wg.Wait()

	var out []string
	for i, h := range hits {
		if h {
			out = append(out, list[i])
		}
	}
	return out, nil
}
