//go:build windows

package arpscan

import (
	"context"
	"encoding/binary"
	"net"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const arpSupported = true

var (
	iphlpapi    = windows.NewLazySystemDLL("iphlpapi.dll")
	procSendARP = iphlpapi.NewProc("SendARP")
)

// sweepSubnet resolves each host with the SendARP IP-Helper API, which fires an
// ARP request and returns the MAC — unprivileged, no raw sockets or Npcap.
// Calls run concurrently because SendARP blocks until reply or timeout.
func sweepSubnet(ctx context.Context, iface string, srcMAC net.HardwareAddr, srcIP net.IP, ipnet *net.IPNet, window time.Duration, maxHosts int) (map[string]string, error) {
	found := map[string]string{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 64)

	for _, t := range hostsIn(ipnet, maxHosts) {
		if t.Equal(srcIP) {
			continue
		}
		if ctx.Err() != nil {
			break
		}
		sem <- struct{}{}
		wg.Add(1)
		go func(target net.IP) {
			defer wg.Done()
			defer func() { <-sem }()
			if mac, ok := sendARP(target); ok {
				mu.Lock()
				found[target.String()] = mac
				mu.Unlock()
			}
		}(t)
	}
	wg.Wait()
	return found, nil
}

// sendARP calls iphlpapi!SendARP for one target, returning its MAC on success.
func sendARP(target net.IP) (string, bool) {
	t4 := target.To4()
	if t4 == nil {
		return "", false
	}
	dest := binary.LittleEndian.Uint32(t4)
	var mac [8]byte // SendARP wants a ULONG-aligned buffer; 6 bytes are used
	macLen := uint32(len(mac))
	ret, _, _ := procSendARP.Call(
		uintptr(dest),
		0, // SrcIP 0: let Windows pick the source
		uintptr(unsafe.Pointer(&mac[0])),
		uintptr(unsafe.Pointer(&macLen)),
	)
	if ret != 0 || macLen < 6 {
		return "", false
	}
	return net.HardwareAddr(mac[:6]).String(), true
}
