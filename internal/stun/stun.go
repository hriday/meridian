// Package stun runs a STUN binding request (RFC 5389) to discover your public
// (reflexive) IP:port and reason about the NAT in front of you — the key to
// diagnosing one-way-audio and NAT-traversal problems in SIP and WebRTC. By
// querying two servers it can also spot a symmetric NAT, which breaks media
// without a relay. Point-in-time; nothing is stored.
package stun

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"
)

const magicCookie = 0x2112A442

var magicBytes = [4]byte{0x21, 0x12, 0xA4, 0x42}

// Result is the NAT picture discovered via STUN.
type Result struct {
	Server    string `json:"server"`
	LocalIP   string `json:"localIP"`
	PublicIP  string `json:"publicIP"`
	PublicPort int   `json:"publicPort"`
	NAT       bool   `json:"nat"`       // public IP differs from local
	NATType   string `json:"natType"`   // human description
	RTTMillis int    `json:"rttMillis"`
}

type mapping struct {
	ip   string
	port int
}

// Discover queries each server from ONE socket and reports the reflexive
// address and NAT assessment. Using one socket to several destinations is what
// makes symmetric-NAT detection valid: a cone NAT keeps the same mapping to
// every destination, a symmetric NAT changes it per destination. Servers that
// don't answer (e.g. UDP-filtered) are skipped.
func Discover(ctx context.Context, servers []string) (Result, error) {
	if len(servers) == 0 {
		return Result{}, errors.New("no STUN server given")
	}
	pc, err := net.ListenUDP("udp4", &net.UDPAddr{})
	if err != nil {
		return Result{}, err
	}
	defer pc.Close()
	local := egressIPv4()

	start := time.Now()
	var maps []mapping
	var used string
	for _, srv := range servers {
		m, err := queryOn(ctx, pc, srv)
		if err != nil {
			continue
		}
		if used == "" {
			used = srv
		}
		maps = append(maps, m)
	}
	if len(maps) == 0 {
		return Result{}, errors.New("no STUN server responded (UDP may be blocked on this network)")
	}

	res := Result{
		Server: used, LocalIP: local, PublicIP: maps[0].ip, PublicPort: maps[0].port,
		RTTMillis: int(time.Since(start).Milliseconds()),
		NAT:       local != "" && maps[0].ip != local,
	}
	switch {
	case !res.NAT:
		res.NATType = "No NAT — this host has a public address"
	case len(maps) < 2:
		res.NATType = "NAT present — only one STUN server was reachable, so the mapping type couldn't be classified"
	default:
		symmetric := false
		for _, m := range maps[1:] {
			if m != maps[0] {
				symmetric = true
			}
		}
		if symmetric {
			res.NATType = "Symmetric NAT — the mapping changes per destination; media needs a TURN relay"
		} else {
			res.NATType = "NAT with endpoint-independent mapping — good for VoIP/WebRTC"
		}
	}
	return res, nil
}

// queryOn sends a binding request to server from the shared socket and returns
// the reflexive mapping it reports.
func queryOn(ctx context.Context, pc *net.UDPConn, server string) (mapping, error) {
	srvAddr, err := net.ResolveUDPAddr("udp4", server)
	if err != nil {
		return mapping{}, err
	}
	var txID [12]byte
	_, _ = rand.Read(txID[:])
	if _, err := pc.WriteToUDP(buildRequest(txID), srvAddr); err != nil {
		return mapping{}, err
	}
	deadline := time.Now().Add(3 * time.Second)
	if dl, ok := ctx.Deadline(); ok && dl.Before(deadline) {
		deadline = dl
	}
	buf := make([]byte, 1024)
	for {
		_ = pc.SetReadDeadline(deadline)
		n, _, err := pc.ReadFromUDP(buf)
		if err != nil {
			return mapping{}, fmt.Errorf("no response from %s", server)
		}
		if n < 20 || string(buf[8:20]) != string(txID[:]) { // not our transaction
			if time.Now().After(deadline) {
				return mapping{}, fmt.Errorf("no response from %s", server)
			}
			continue
		}
		ip, port, err := parseResponse(txID, buf[:n])
		if err != nil {
			return mapping{}, err
		}
		return mapping{ip: ip.String(), port: port}, nil
	}
}

func egressIPv4() string {
	c, err := net.Dial("udp", "8.8.8.8:9")
	if err != nil {
		return ""
	}
	defer c.Close()
	if la, ok := c.LocalAddr().(*net.UDPAddr); ok {
		return la.IP.String()
	}
	return ""
}

func buildRequest(txID [12]byte) []byte {
	msg := make([]byte, 20)
	binary.BigEndian.PutUint16(msg[0:], 0x0001) // Binding Request
	binary.BigEndian.PutUint16(msg[2:], 0)      // no attributes
	binary.BigEndian.PutUint32(msg[4:], magicCookie)
	copy(msg[8:20], txID[:])
	return msg
}

// parseResponse extracts the mapped address, preferring XOR-MAPPED-ADDRESS.
func parseResponse(txID [12]byte, resp []byte) (net.IP, int, error) {
	if len(resp) < 20 {
		return nil, 0, errors.New("STUN response too short")
	}
	if binary.BigEndian.Uint16(resp[0:])&0xC000 != 0 {
		return nil, 0, errors.New("not a STUN response")
	}
	attrs := resp[20:]
	var fbIP net.IP
	var fbPort int
	for i := 0; i+4 <= len(attrs); {
		atype := binary.BigEndian.Uint16(attrs[i:])
		alen := int(binary.BigEndian.Uint16(attrs[i+2:]))
		if i+4+alen > len(attrs) {
			break
		}
		val := attrs[i+4 : i+4+alen]
		switch atype {
		case 0x0020: // XOR-MAPPED-ADDRESS
			if ip, port, ok := parseXorAddr(val); ok {
				return ip, port, nil
			}
		case 0x0001: // MAPPED-ADDRESS (fallback)
			if ip, port, ok := parseMappedAddr(val); ok {
				fbIP, fbPort = ip, port
			}
		}
		i += 4 + alen
		if pad := alen % 4; pad != 0 {
			i += 4 - pad
		}
	}
	if fbIP != nil {
		return fbIP, fbPort, nil
	}
	return nil, 0, errors.New("no mapped address in STUN response")
}

func parseXorAddr(val []byte) (net.IP, int, bool) {
	if len(val) < 8 || val[1] != 0x01 { // IPv4 only
		return nil, 0, false
	}
	port := int(binary.BigEndian.Uint16(val[2:4]) ^ uint16(magicCookie>>16))
	ip := make(net.IP, 4)
	for i := 0; i < 4; i++ {
		ip[i] = val[4+i] ^ magicBytes[i]
	}
	return ip, port, true
}

func parseMappedAddr(val []byte) (net.IP, int, bool) {
	if len(val) < 8 || val[1] != 0x01 {
		return nil, 0, false
	}
	port := int(binary.BigEndian.Uint16(val[2:4]))
	ip := net.IPv4(val[4], val[5], val[6], val[7]).To4()
	return ip, port, true
}
