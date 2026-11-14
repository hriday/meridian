//go:build windows

package mtu

import (
	"context"
	"encoding/binary"
	"fmt"
	"net/netip"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const supported = true

// Windows honors the Don't-Fragment bit natively through the IP Helper API's
// IcmpSendEcho, which sends a DF ICMP echo and reports IP_PACKET_TOO_BIG when a
// link can't carry it — exactly what path-MTU discovery needs, unprivileged.

const (
	ipFlagDF         = 0x02
	ipSuccess        = 0
	ipReqTimedOut    = 11010
	ipPacketTooBig   = 11009
	ipFragReassembly = 11013 // treat as "didn't fit"
)

var (
	iphlpapi          = windows.NewLazySystemDLL("iphlpapi.dll")
	procIcmpCreate    = iphlpapi.NewProc("IcmpCreateFile")
	procIcmpClose     = iphlpapi.NewProc("IcmpCloseHandle")
	procIcmpSendEcho  = iphlpapi.NewProc("IcmpSendEcho")
)

// ipOptionInformation mirrors the Win32 IP_OPTION_INFORMATION struct.
type ipOptionInformation struct {
	TTL         byte
	Tos         byte
	Flags       byte
	OptionsSize byte
	OptionsData *byte
}

// icmpEchoReply mirrors the Win32 ICMP_ECHO_REPLY struct (64-bit layout).
type icmpEchoReply struct {
	Address       uint32
	Status        uint32
	RoundTripTime uint32
	DataSize      uint16
	Reserved      uint16
	Data          uintptr
	Options       ipOptionInformation
}

type winProber struct {
	handle    uintptr
	dest      uint32
	timeoutMs uint32
}

func newProber(ip netip.Addr, timeout time.Duration) (Prober, error) {
	if !ip.Is4() {
		return nil, fmt.Errorf("path MTU discovery currently supports IPv4 only")
	}
	h, _, err := procIcmpCreate.Call()
	if h == 0 || h == uintptr(^uintptr(0)) {
		return nil, fmt.Errorf("couldn't open an ICMP handle: %v", err)
	}
	a := ip.As4()
	dest := binary.LittleEndian.Uint32(a[:]) // IPAddr: the four octets in order
	ms := uint32(timeout / time.Millisecond)
	if ms == 0 {
		ms = 1000
	}
	return &winProber{handle: h, dest: dest, timeoutMs: ms}, nil
}

func (p *winProber) Close() error {
	procIcmpClose.Call(p.handle)
	return nil
}

func (p *winProber) Probe(ctx context.Context, payload int) (ProbeResult, error) {
	if err := ctx.Err(); err != nil {
		return ProbeResult{}, err
	}
	req := make([]byte, payload)
	reqPtr := uintptr(0)
	if payload > 0 {
		reqPtr = uintptr(unsafe.Pointer(&req[0]))
	} else {
		dummy := [1]byte{}
		reqPtr = uintptr(unsafe.Pointer(&dummy[0]))
	}
	replyLen := int(unsafe.Sizeof(icmpEchoReply{})) + payload + 8 + 16
	reply := make([]byte, replyLen)

	opts := ipOptionInformation{TTL: 128, Flags: ipFlagDF}

	ret, _, callErr := procIcmpSendEcho.Call(
		p.handle,
		uintptr(p.dest),
		reqPtr,
		uintptr(uint16(payload)),
		uintptr(unsafe.Pointer(&opts)),
		uintptr(unsafe.Pointer(&reply[0])),
		uintptr(uint32(replyLen)),
		uintptr(p.timeoutMs),
	)

	if ret == 0 {
		// Failure: the reason is in GetLastError (surfaced as callErr).
		if errno, ok := callErr.(syscall.Errno); ok {
			switch uintptr(errno) {
			case ipPacketTooBig, ipFragReassembly:
				return ProbeResult{Outcome: TooBig}, nil
			case ipReqTimedOut:
				return ProbeResult{Outcome: Unknown}, nil
			}
		}
		return ProbeResult{Outcome: Unknown}, nil
	}

	r := (*icmpEchoReply)(unsafe.Pointer(&reply[0]))
	switch r.Status {
	case ipSuccess:
		return ProbeResult{Outcome: Fits}, nil
	case ipPacketTooBig, ipFragReassembly:
		return ProbeResult{Outcome: TooBig}, nil
	default: // timeout or other: treat as no reply
		return ProbeResult{Outcome: Unknown}, nil
	}
}
