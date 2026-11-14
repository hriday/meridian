package rtp

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/hriday/meridian/internal/l2"
)

// DefaultWindow is how long to capture by default.
const DefaultWindow = 10 * time.Second

// Supported reports whether live RTP capture works on this OS.
func Supported() bool { return l2.Supported() }

// Capture sniffs iface (empty = the egress interface) for up to window and
// returns the RTP streams seen. It needs raw-capture privileges (sudo/ChmodBPF
// on macOS, sudo on Linux, Npcap on Windows); without them it returns the
// corresponding l2 error. Note: on a switched network you only see RTP you are
// in the path of (your own calls, or a SPAN/mirror port).
func Capture(ctx context.Context, iface string, window time.Duration) ([]Stream, error) {
	if !l2.Supported() {
		return nil, errors.New("RTP capture isn't supported on this operating system yet")
	}
	if window <= 0 {
		window = DefaultWindow
	}
	if iface == "" {
		name, err := egressIface()
		if err != nil {
			return nil, err
		}
		iface = name
	}
	c, err := l2.Open(iface)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	a := NewAnalyzer()
	deadline := time.Now().Add(window)
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			break
		}
		frames, err := c.Read()
		if err != nil {
			if errors.Is(err, l2.ErrReadTimeout) {
				continue
			}
			return a.Streams(), err
		}
		now := time.Now().UnixNano()
		for _, f := range frames {
			if p, ok := ParseFrame(f, now); ok {
				a.Add(p)
			}
		}
	}
	return a.Streams(), nil
}

func egressIface() (string, error) {
	c, err := net.Dial("udp", "8.8.8.8:9")
	if err != nil {
		return "", errors.New("couldn't determine the outgoing interface")
	}
	defer c.Close()
	la, ok := c.LocalAddr().(*net.UDPAddr)
	if !ok {
		return "", errors.New("couldn't determine the local address")
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, ifc := range ifaces {
		addrs, _ := ifc.Addrs()
		for _, ad := range addrs {
			if ipn, ok := ad.(*net.IPNet); ok && ipn.IP.Equal(la.IP) {
				return ifc.Name, nil
			}
		}
	}
	return "", errors.New("couldn't find the egress interface")
}
