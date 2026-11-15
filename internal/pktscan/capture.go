package pktscan

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/l2"
)

// DefaultWindow is how long to capture by default.
const DefaultWindow = 8 * time.Second

// Supported reports whether live packet capture works on this OS.
func Supported() bool { return l2.Supported() }

// Run captures on iface for window and returns a traffic summary. With iface ""
// it picks the egress interface. It needs raw-capture privileges (sudo /
// ChmodBPF on macOS, sudo on Linux, Npcap on Windows); without them it returns
// the l2 privilege error. On a switched network you only see traffic you're in
// the path of (your own, broadcast/multicast, or a SPAN/mirror port).
func Run(ctx context.Context, iface string, window time.Duration, topN int) (Summary, error) {
	if !l2.Supported() {
		return Summary{}, errors.New("packet capture isn't supported on this operating system yet")
	}
	if window <= 0 {
		window = DefaultWindow
	}
	if topN <= 0 {
		topN = 10
	}
	if iface == "" {
		name, err := egressIface()
		if err != nil {
			return Summary{}, err
		}
		iface = name
	}

	c, err := l2.Open(iface)
	if err != nil {
		return Summary{}, err
	}
	defer c.Close()

	s := newSummarizer()
	start := time.Now()
	deadline := start.Add(window)
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			break
		}
		frames, err := c.Read()
		if err != nil {
			if errors.Is(err, l2.ErrReadTimeout) {
				continue
			}
			return s.finalize(iface, time.Since(start).Seconds(), topN), err
		}
		for _, f := range frames {
			if p, ok := parseFrame(f); ok {
				s.ingest(p)
			}
		}
	}
	return s.finalize(iface, time.Since(start).Seconds(), topN), nil
}

// egressIface picks an up, non-loopback interface with an IPv4 address, skipping
// VPN/AWDL virtual interfaces.
func egressIface() (string, error) {
	ifs, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, ifi := range ifs {
		if ifi.Flags&net.FlagUp == 0 || ifi.Flags&net.FlagLoopback != 0 {
			continue
		}
		name := strings.ToLower(ifi.Name)
		if strings.HasPrefix(name, "utun") || strings.HasPrefix(name, "awdl") || strings.HasPrefix(name, "llw") {
			continue
		}
		addrs, _ := ifi.Addrs()
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok && ipn.IP.To4() != nil && !ipn.IP.IsLoopback() {
				return ifi.Name, nil
			}
		}
	}
	return "", errors.New("no suitable network interface found")
}
