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

const qmUnicast = 0x8000 // mDNS QU bit: ask responders to unicast their reply

// mdnsTypes is a broad spread of DNS-SD service types. The query goes to the
// multicast group, but on macOS the local mDNS responder answers from its cache
// (the aggregated view of the whole LAN) by unicast to our ephemeral port — so
// the wider this list, the more of the network's services we learn about.
var mdnsTypes = []string{
	"_http._tcp.local.", "_https._tcp.local.",
	"_airplay._tcp.local.", "_raop._tcp.local.", "_airport._tcp.local.",
	"_googlecast._tcp.local.", "_spotify-connect._tcp.local.", "_amzn-wplay._tcp.local.",
	"_sonos._tcp.local.", "_soundtouch._tcp.local.",
	"_ipp._tcp.local.", "_ipps._tcp.local.", "_printer._tcp.local.", "_pdl-datastream._tcp.local.", "_scanner._tcp.local.",
	"_ssh._tcp.local.", "_sftp-ssh._tcp.local.", "_rfb._tcp.local.",
	"_smb._tcp.local.", "_afpovertcp._tcp.local.", "_nfs._tcp.local.", "_webdav._tcp.local.",
	"_homekit._tcp.local.", "_hap._tcp.local.", "_matter._tcp.local.", "_matterc._udp.local.",
	"_hue._tcp.local.", "_philips-hue._tcp.local.", "_companion-link._tcp.local.",
	"_device-info._tcp.local.", "_workstation._tcp.local.", "_daap._tcp.local.", "_dpap._tcp.local.",
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
		s, err := systemMDNS(ctx, window)
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
// real LAN) and collects unicast replies until the window closes. It sends
// several bursts, spaced out, because a single mDNS/SSDP query is often not
// answered — responders reply probabilistically and apply duplicate
// suppression, so one shot is unreliable.
func multicastProbe(ctx context.Context, window time.Duration, dst *net.UDPAddr, payload []byte, bufSize int) ([]rawReply, error) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	p := ipv4.NewPacketConn(conn)
	_ = p.SetMulticastTTL(2)
	ifaces := candidateInterfaces()

	burst := func() {
		if len(ifaces) == 0 {
			_, _ = conn.WriteToUDP(payload, dst) // default route
			return
		}
		for i := range ifaces {
			if err := p.SetMulticastInterface(&ifaces[i]); err == nil {
				_, _ = conn.WriteToUDP(payload, dst)
			}
		}
	}

	// One burst immediately, then repeated bursts spread across the window. The
	// spread matters: on macOS the first query only *primes* the local
	// responder (it then goes and discovers), and later queries reap the cached
	// answers — bursts packed too closely never give it time. A net.UDPConn is
	// safe for one concurrent reader and writer.
	burst()
	go func() {
		t := time.NewTicker(800 * time.Millisecond)
		defer t.Stop()
		n := int(window / (800 * time.Millisecond))
		if n < 4 {
			n = 4
		}
		if n > 20 {
			n = 20
		}
		for i := 0; i < n; i++ {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				burst()
			}
		}
	}()

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

// queryMDNS sends a DNS-SD query to the mDNS group from an ephemeral port (with
// the QU bit, asking for a unicast reply) out every real interface, then reads
// the unicast answers. We deliberately do NOT bind port 5353: on macOS the
// system mDNS responder owns 5353 and no other socket receives its multicast,
// but it happily unicasts cached answers back to our ephemeral port — which is
// the aggregated view of the whole LAN.
func queryMDNS(ctx context.Context, window time.Duration) ([]Service, error) {
	m := new(dns.Msg)
	m.Id = 0
	for _, t := range mdnsTypes {
		m.Question = append(m.Question, dns.Question{Name: t, Qtype: dns.TypePTR, Qclass: dns.ClassINET | qmUnicast})
	}
	buf, err := m.Pack()
	if err != nil {
		return nil, fmt.Errorf("pack mDNS query: %w", err)
	}
	dst := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251), Port: 5353}

	// One long-lived socket for the whole window. The local responder answers our
	// ephemeral-port query by unicast, but a cold answer can take ~4-5s to arrive
	// — so the socket must stay open (same source port) long enough to catch it.
	replies, err := multicastProbe(ctx, window, dst, buf, 9000)
	if err != nil {
		return nil, fmt.Errorf("mDNS probe: %w", err)
	}
	payloads := make([][]byte, len(replies))
	for i, r := range replies {
		payloads[i] = r.data
	}
	return parseMDNS(payloads), nil
}

func querySSDP(ctx context.Context, window time.Duration) ([]Device, error) {
	// SSDP devices answer within an MX-bounded window (a few seconds); no need to
	// hold the socket as long as the slow mDNS cache warm-up needs.
	if window > 5*time.Second {
		window = 5 * time.Second
	}
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
