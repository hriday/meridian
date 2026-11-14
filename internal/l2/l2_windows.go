//go:build windows

package l2

import (
	"fmt"
	"net"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const supported = true

// On Windows there is no native raw-L2 API, so capture goes through Npcap's
// wpcap.dll (the WinPcap-compatible library). It's loaded dynamically — no cgo,
// so the build stays pure-Go and cross-compilable — and if Npcap isn't
// installed, Open returns ErrNeedsNpcap.
//
// This file is written against the documented libpcap/Npcap API but has not been
// run on Windows; it needs verification there.

var (
	wpcap              *windows.LazyDLL
	procFindAllDevs    *windows.LazyProc
	procFreeAllDevs    *windows.LazyProc
	procOpenLive       *windows.LazyProc
	procNextEx         *windows.LazyProc
	procSendPacket     *windows.LazyProc
	procClose          *windows.LazyProc
)

// loadWpcap finds wpcap.dll. Npcap installs it under System32\Npcap unless
// "WinPcap API-compatible mode" was chosen (then it's in System32), so try both.
func loadWpcap() error {
	if wpcap != nil {
		return wpcap.Load()
	}
	for _, name := range []string{"wpcap.dll", `C:\Windows\System32\Npcap\wpcap.dll`} {
		d := windows.NewLazyDLL(name)
		if err := d.Load(); err == nil {
			wpcap = d
			procFindAllDevs = d.NewProc("pcap_findalldevs")
			procFreeAllDevs = d.NewProc("pcap_freealldevs")
			procOpenLive = d.NewProc("pcap_open_live")
			procNextEx = d.NewProc("pcap_next_ex")
			procSendPacket = d.NewProc("pcap_sendpacket")
			procClose = d.NewProc("pcap_close")
			return nil
		}
	}
	return ErrNeedsNpcap
}

// pcapIf mirrors struct pcap_if (x64).
type pcapIf struct {
	next        uintptr
	name        uintptr
	description uintptr
	addresses   uintptr
	flags       uint32
}

// pcapAddr mirrors struct pcap_addr (x64).
type pcapAddr struct {
	next      uintptr
	addr      uintptr
	netmask   uintptr
	broadaddr uintptr
	dstaddr   uintptr
}

// Conn is a wpcap capture handle bound to an interface.
type Conn struct {
	handle uintptr
}

// Open finds the Npcap device whose IPv4 address matches the egress interface
// and opens it for capture + injection.
func Open(iface string) (*Conn, error) {
	if err := loadWpcap(); err != nil {
		return nil, err
	}
	egress, err := egressIPv4()
	if err != nil {
		return nil, err
	}
	devName, err := deviceForIP(egress)
	if err != nil {
		return nil, err
	}

	nameC, err := windows.BytePtrFromString(devName)
	if err != nil {
		return nil, err
	}
	errbuf := make([]byte, 256)
	h, _, _ := procOpenLive.Call(
		uintptr(unsafe.Pointer(nameC)),
		65536,                                  // snaplen
		1,                                      // promiscuous (see multicast LLDP/CDP)
		1000,                                   // read timeout 1s so callers poll
		uintptr(unsafe.Pointer(&errbuf[0])),
	)
	if h == 0 {
		return nil, fmt.Errorf("pcap_open_live failed: %s", cstr(uintptr(unsafe.Pointer(&errbuf[0]))))
	}
	return &Conn{handle: h}, nil
}

func (c *Conn) Close() error {
	if c.handle != 0 {
		procClose.Call(c.handle)
		c.handle = 0
	}
	return nil
}

// Write injects one complete Ethernet frame.
func (c *Conn) Write(frame []byte) error {
	if len(frame) == 0 {
		return nil
	}
	ret, _, _ := procSendPacket.Call(c.handle, uintptr(unsafe.Pointer(&frame[0])), uintptr(len(frame)))
	if int32(ret) != 0 {
		return fmt.Errorf("pcap_sendpacket failed (%d)", int32(ret))
	}
	return nil
}

// Read returns one frame, or ErrReadTimeout if the poll interval elapsed.
func (c *Conn) Read() ([][]byte, error) {
	var hdr uintptr // *pcap_pkthdr
	var data uintptr // *u_char
	ret, _, _ := procNextEx.Call(c.handle, uintptr(unsafe.Pointer(&hdr)), uintptr(unsafe.Pointer(&data)))
	switch int32(ret) {
	case 1: // a frame was read
		// pcap_pkthdr{ ts timeval(8 on Win), caplen uint32, len uint32 }; caplen at offset 8.
		caplen := *(*uint32)(unsafe.Pointer(hdr + 8))
		f := make([]byte, caplen)
		copy(f, unsafe.Slice((*byte)(unsafe.Pointer(data)), caplen))
		return [][]byte{f}, nil
	case 0: // timeout
		return nil, ErrReadTimeout
	default: // -1 error, -2 EOF
		return nil, fmt.Errorf("pcap_next_ex failed (%d)", int32(ret))
	}
}

// deviceForIP walks pcap_findalldevs and returns the device whose addresses
// include ip.
func deviceForIP(ip net.IP) (string, error) {
	var alldevs uintptr
	errbuf := make([]byte, 256)
	ret, _, _ := procFindAllDevs.Call(uintptr(unsafe.Pointer(&alldevs)), uintptr(unsafe.Pointer(&errbuf[0])))
	if int32(ret) != 0 || alldevs == 0 {
		return "", fmt.Errorf("pcap_findalldevs failed: %s", cstr(uintptr(unsafe.Pointer(&errbuf[0]))))
	}
	defer procFreeAllDevs.Call(alldevs)

	want := ip.To4()
	for dev := alldevs; dev != 0; {
		d := (*pcapIf)(unsafe.Pointer(dev))
		for a := d.addresses; a != 0; {
			addr := (*pcapAddr)(unsafe.Pointer(a))
			if addr.addr != 0 {
				family := *(*uint16)(unsafe.Pointer(addr.addr))
				if family == syscall.AF_INET {
					// sockaddr_in: sin_family(2) sin_port(2) sin_addr(4)
					b := unsafe.Slice((*byte)(unsafe.Pointer(addr.addr+4)), 4)
					if want != nil && b[0] == want[0] && b[1] == want[1] && b[2] == want[2] && b[3] == want[3] {
						return cstr(d.name), nil
					}
				}
			}
			a = addr.next
		}
		dev = d.next
	}
	return "", fmt.Errorf("couldn't find a capture device for %s", ip)
}

// egressIPv4 returns this host's IPv4 toward the default route.
func egressIPv4() (net.IP, error) {
	c, err := net.Dial("udp", "8.8.8.8:9")
	if err != nil {
		return nil, fmt.Errorf("couldn't determine the egress interface: %w", err)
	}
	defer c.Close()
	la, ok := c.LocalAddr().(*net.UDPAddr)
	if !ok || la.IP.To4() == nil {
		return nil, fmt.Errorf("couldn't determine the local IPv4 address")
	}
	return la.IP.To4(), nil
}

// cstr reads a NUL-terminated C string at p.
func cstr(p uintptr) string {
	if p == 0 {
		return ""
	}
	var b []byte
	for i := 0; ; i++ {
		ch := *(*byte)(unsafe.Pointer(p + uintptr(i)))
		if ch == 0 {
			break
		}
		b = append(b, ch)
	}
	return string(b)
}
