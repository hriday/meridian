package discover

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"golang.org/x/net/ipv4"
)

// mdnsTypes is a spread of common DNS-SD service types, plus the meta query
// that asks a host to list the types it offers.
var mdnsTypes = []string{
	"_services._dns-sd._udp.local.",
	"_http._tcp.local.", "_https._tcp.local.",
	"_airplay._tcp.local.", "_raop._tcp.local.", "_googlecast._tcp.local.",
	"_spotify-connect._tcp.local.",
	"_ipp._tcp.local.", "_ipps._tcp.local.", "_printer._tcp.local.", "_pdl-datastream._tcp.local.", "_scanner._tcp.local.",
	"_ssh._tcp.local.", "_sftp-ssh._tcp.local.",
	"_smb._tcp.local.", "_afpovertcp._tcp.local.", "_nfs._tcp.local.",
	"_homekit._tcp.local.", "_hap._tcp.local.",
	"_device-info._tcp.local.", "_workstation._tcp.local.",
}

// Run performs an mDNS and an SSDP sweep concurrently for the given window and
// returns everything that answered.
func Run(ctx context.Context, window time.Duration) (Result, error) {
	var res Result
	var mu sync.Mutex
	var wg sync.WaitGroup
	var mdnsErr, ssdpErr error

	wg.Add(2)
	go func() {
		defer wg.Done()
		s, err := queryMDNS(ctx, window)
		mu.Lock()
		res.Services, mdnsErr = s, err
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		d, err := querySSDP(ctx, window)
		mu.Lock()
		res.Devices, ssdpErr = d, err
		mu.Unlock()
	}()
	wg.Wait()

	// Only fail if neither probe could even start (e.g. no usable network).
	if mdnsErr != nil && ssdpErr != nil {
		return res, fmt.Errorf("discovery failed: %v; %v", mdnsErr, ssdpErr)
	}
	return res, nil
}

func readDeadline(ctx context.Context, window time.Duration) time.Time {
	dl := time.Now().Add(window)
	if d, ok := ctx.Deadline(); ok && d.Before(dl) {
		dl = d
	}
	return dl
}

type rawReply struct {
	data []byte
	addr string
}

// candidateInterfaces returns up, multicast-capable, non-loopback interfaces
// that have an IPv4 address — excluding VPN tunnels (utun/ppp/ipsec), whose
// default-route multicast would never reach the physical LAN.
func candidateInterfaces() []net.Interface {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var out []net.Interface
	for _, ifi := range ifs {
		if ifi.Flags&net.FlagUp == 0 || ifi.Flags&net.FlagMulticast == 0 || ifi.Flags&net.FlagLoopback != 0 {
			continue
		}
		name := strings.ToLower(ifi.Name)
		if strings.HasPrefix(name, "utun") || strings.HasPrefix(name, "ppp") ||
			strings.HasPrefix(name, "ipsec") || strings.HasPrefix(name, "tun") ||
			strings.HasPrefix(name, "tap") {
			continue
		}
		addrs, _ := ifi.Addrs()
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok && ipn.IP.To4() != nil && !ipn.IP.IsLoopback() {
				out = append(out, ifi)
				break
			}
		}
	}
	return out
}

// multicastProbe sends payload to a multicast destination out of every
// candidate interface (so a multi-homed host with VPN tunnels still reaches the
// real LAN), then collects unicast replies until the window closes.
func multicastProbe(ctx context.Context, window time.Duration, dst *net.UDPAddr, payload []byte, bufSize int) ([]rawReply, error) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	p := ipv4.NewPacketConn(conn)
	_ = p.SetMulticastTTL(2)
	sent := 0
	for _, ifi := range candidateInterfaces() {
		if err := p.SetMulticastInterface(&ifi); err != nil {
			continue
		}
		if _, err := conn.WriteToUDP(payload, dst); err == nil {
			sent++
		}
	}
	if sent == 0 {
		// No usable interface — fall back to the default route.
		if _, err := conn.WriteToUDP(payload, dst); err != nil {
			return nil, err
		}
	}

	_ = conn.SetReadDeadline(readDeadline(ctx, window))
	var replies []rawReply
	b := make([]byte, bufSize)
	for {
		n, addr, err := conn.ReadFromUDP(b)
		if err != nil {
			break
		}
		cp := make([]byte, n)
		copy(cp, b[:n])
		replies = append(replies, rawReply{data: cp, addr: addr.IP.String()})
	}
	return replies, nil
}

// queryMDNS binds UDP/5353 (with SO_REUSEPORT, to coexist with the system mDNS
// responder), joins the mDNS group on each real interface, sends a DNS-SD
// query, and collects the multicast responses and ambient announcements for the
// window. Responders reply by multicast, so we must be group members on the
// right interface to hear them.
func queryMDNS(ctx context.Context, window time.Duration) ([]Service, error) {
	ifaces := candidateInterfaces()
	if len(ifaces) == 0 {
		return nil, fmt.Errorf("no multicast-capable interface")
	}

	m := new(dns.Msg)
	m.Id = 0
	for _, t := range mdnsTypes {
		m.Question = append(m.Question, dns.Question{Name: t, Qtype: dns.TypePTR, Qclass: dns.ClassINET})
	}
	buf, err := m.Pack()
	if err != nil {
		return nil, fmt.Errorf("pack mDNS query: %w", err)
	}

	lc := net.ListenConfig{Control: reusePortControl}
	pc, err := lc.ListenPacket(ctx, "udp4", "0.0.0.0:5353")
	if err != nil {
		return nil, fmt.Errorf("bind mDNS port: %w", err)
	}
	conn := pc.(*net.UDPConn)
	defer conn.Close()

	p := ipv4.NewPacketConn(conn)
	group := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251)}
	joined := 0
	for i := range ifaces {
		if err := p.JoinGroup(&ifaces[i], group); err == nil {
			joined++
		}
	}
	if joined == 0 {
		return nil, fmt.Errorf("couldn't join the mDNS group on any interface")
	}

	// Solicit responses: send the query out every interface, twice, since mDNS
	// queries are unreliable UDP.
	dst := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251), Port: 5353}
	sendOnAllInterfaces(ifaces, dst, buf)
	time.Sleep(150 * time.Millisecond)
	sendOnAllInterfaces(ifaces, dst, buf)

	_ = conn.SetReadDeadline(readDeadline(ctx, window))
	var payloads [][]byte
	b := make([]byte, 9000)
	for {
		n, _, err := conn.ReadFromUDP(b)
		if err != nil {
			break
		}
		cp := make([]byte, n)
		copy(cp, b[:n])
		payloads = append(payloads, cp)
	}
	return parseMDNS(payloads), nil
}

// sendOnAllInterfaces sends payload to a multicast destination out of each
// interface in turn.
func sendOnAllInterfaces(ifaces []net.Interface, dst *net.UDPAddr, payload []byte) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return
	}
	defer conn.Close()
	p := ipv4.NewPacketConn(conn)
	_ = p.SetMulticastTTL(2)
	for i := range ifaces {
		if err := p.SetMulticastInterface(&ifaces[i]); err != nil {
			continue
		}
		_, _ = conn.WriteToUDP(payload, dst)
	}
}

func querySSDP(ctx context.Context, window time.Duration) ([]Device, error) {
	msearch := "M-SEARCH * HTTP/1.1\r\n" +
		"HOST: 239.255.255.250:1900\r\n" +
		"MAN: \"ssdp:discover\"\r\n" +
		"MX: 2\r\n" +
		"ST: ssdp:all\r\n\r\n"
	dst := &net.UDPAddr{IP: net.IPv4(239, 255, 255, 250), Port: 1900}
	replies, err := multicastProbe(ctx, window, dst, []byte(msearch), 4096)
	if err != nil {
		return nil, fmt.Errorf("SSDP probe: %w", err)
	}
	var devices []Device
	for _, r := range replies {
		if d, ok := parseSSDP(string(r.data), r.addr); ok {
			devices = append(devices, d)
		}
	}
	return dedupeDevices(devices), nil
}
