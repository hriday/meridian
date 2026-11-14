package dhcpscan

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"net"
	"time"
)

// DefaultWindow is how long to listen for offers after the broadcast. A couple
// of seconds is too racy — some servers take 3-4s to answer a cold DISCOVER.
const DefaultWindow = 5 * time.Second

// Supported reports whether DHCP discovery is implemented on this OS.
func Supported() bool { return supported }

// Run broadcasts a DISCOVER on the egress interface and returns the servers that
// answer. It needs elevated privileges (to bind UDP :68); without them it
// returns ErrNeedsPrivilege.
func Run(window time.Duration) (Result, error) {
	if !supported {
		return Result{}, errors.New("DHCP discovery isn't supported on this operating system yet")
	}
	if window <= 0 {
		window = DefaultWindow
	}
	iface, mac, err := egressInterface()
	if err != nil {
		return Result{}, err
	}
	t, err := newTransport(window)
	if err != nil {
		return Result{}, err
	}
	defer t.Close()
	return Scan(t, iface, mac, randomXID())
}

// egressInterface finds the interface that carries the default route (by dialing
// a public address without sending) and returns its name and MAC.
func egressInterface() (string, net.HardwareAddr, error) {
	c, err := net.Dial("udp", "8.8.8.8:9")
	if err != nil {
		return "", nil, errors.New("couldn't determine the outgoing network interface")
	}
	defer c.Close()
	la, ok := c.LocalAddr().(*net.UDPAddr)
	if !ok {
		return "", nil, errors.New("couldn't determine the local address")
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", nil, err
	}
	for _, ifc := range ifaces {
		addrs, _ := ifc.Addrs()
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok && ipn.IP.Equal(la.IP) {
				if len(ifc.HardwareAddr) == 6 {
					return ifc.Name, ifc.HardwareAddr, nil
				}
			}
		}
	}
	return "", nil, errors.New("couldn't find a usable network interface (no MAC on the egress interface)")
}

func randomXID() uint32 {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return uint32(time.Now().UnixNano())
	}
	return binary.BigEndian.Uint32(b[:])
}
