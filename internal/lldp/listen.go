package lldp

import (
	"context"
	"errors"
	"net"
	"sort"
	"time"
)

// DefaultWindow is how long to listen. Switches send LLDP every ~30s and CDP
// every ~60s, so a short listen can miss the only advertisement.
const DefaultWindow = 35 * time.Second

// capture is a raw L2 frame source. ReadFrames returns zero or more frames and
// returns promptly (within its internal read timeout) so Listen can poll the
// deadline/context.
type capture interface {
	ReadFrames() ([][]byte, error)
	Close() error
}

// Supported reports whether LLDP/CDP capture is implemented on this OS.
func Supported() bool { return supported }

// Listen captures on iface (empty = the egress interface) for up to window and
// returns the distinct LLDP/CDP neighbors heard. It needs elevated privileges;
// without them it returns ErrNeedsPrivilege.
func Listen(ctx context.Context, iface string, window time.Duration) ([]Neighbor, error) {
	if !supported {
		return nil, errors.New("LLDP/CDP capture isn't supported on this operating system yet")
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

	c, err := newCapture(iface)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	deadline := time.Now().Add(window)
	seen := map[string]Neighbor{}
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			break
		}
		frames, err := c.ReadFrames()
		if err != nil {
			if errors.Is(err, errReadTimeout) {
				continue // no frames this interval; keep waiting
			}
			return neighborSlice(seen), err
		}
		for _, f := range frames {
			if n, ok := Classify(f); ok {
				seen[n.Key()] = n
			}
		}
	}
	return neighborSlice(seen), nil
}

func neighborSlice(m map[string]Neighbor) []Neighbor {
	out := make([]Neighbor, 0, len(m))
	for _, n := range m {
		out = append(out, n)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SystemName != out[j].SystemName {
			return out[i].SystemName < out[j].SystemName
		}
		return out[i].PortID < out[j].PortID
	})
	return out
}

// errReadTimeout is what a capture returns when its read window elapsed with no
// frames — not a real error, just "nothing yet".
var errReadTimeout = errors.New("read timeout")

// egressIface finds the interface carrying the default route.
func egressIface() (string, error) {
	c, err := net.Dial("udp", "8.8.8.8:9")
	if err != nil {
		return "", errors.New("couldn't determine the outgoing network interface")
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
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok && ipn.IP.Equal(la.IP) {
				return ifc.Name, nil
			}
		}
	}
	return "", errors.New("couldn't find the egress network interface")
}
