//go:build linux

package discover

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

// Avahi protocol/interface sentinels (avahi-common/address.h, defs.h).
const (
	avahiIfaceUnspec = int32(-1) // AVAHI_IF_UNSPEC
	avahiProtoUnspec = int32(-1) // AVAHI_PROTO_UNSPEC
	avahiBus         = "org.freedesktop.Avahi"
)

type avahiItem struct {
	iface, proto      int32
	name, typ, domain string
}

// systemMDNS browses the LAN through the Avahi daemon over D-Bus — Linux's
// system mDNS resolver, which holds the aggregated multicast view. If Avahi or
// the system bus isn't available, it falls back to the raw multicast probe.
func systemMDNS(ctx context.Context, window time.Duration) ([]Service, error) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return queryMDNS(ctx, window)
	}
	defer conn.Close()

	// Catch ItemNew from every service browser we're about to create. Adding the
	// match before creating the browsers avoids missing cached items that Avahi
	// emits immediately.
	if err := conn.AddMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.Avahi.ServiceBrowser"),
		dbus.WithMatchMember("ItemNew"),
	); err != nil {
		return queryMDNS(ctx, window)
	}
	sigc := make(chan *dbus.Signal, 512)
	conn.Signal(sigc)

	server := conn.Object(avahiBus, dbus.ObjectPath("/"))

	created := 0
	for _, t := range mdnsTypes {
		regtype := strings.TrimSuffix(strings.TrimSuffix(t, "."), ".local")
		var path dbus.ObjectPath
		call := server.Call("org.freedesktop.Avahi.Server.ServiceBrowserNew", 0,
			avahiIfaceUnspec, avahiProtoUnspec, regtype, "", uint32(0))
		if call.Err != nil {
			continue
		}
		if call.Store(&path) == nil {
			created++
		}
	}
	if created == 0 {
		// Avahi isn't answering — fall back rather than return nothing.
		return queryMDNS(ctx, window)
	}

	// Collect browsed instances for the window.
	items := map[string]avahiItem{}
	browseDeadline := time.After(window)
collect:
	for {
		select {
		case <-ctx.Done():
			break collect
		case <-browseDeadline:
			break collect
		case s := <-sigc:
			if s == nil || s.Name != "org.freedesktop.Avahi.ServiceBrowser.ItemNew" || len(s.Body) < 5 {
				continue
			}
			it := avahiItem{}
			it.iface, _ = s.Body[0].(int32)
			it.proto, _ = s.Body[1].(int32)
			it.name, _ = s.Body[2].(string)
			it.typ, _ = s.Body[3].(string)
			it.domain, _ = s.Body[4].(string)
			key := it.name + "|" + it.typ + "|" + it.domain
			if _, ok := items[key]; !ok {
				items[key] = it
			}
		}
	}

	// Resolve each instance (host/port/TXT/address). Avahi's ResolveService is
	// synchronous and can stall, so bound each call and run them concurrently.
	var (
		mu  sync.Mutex
		out []Service
		wg  sync.WaitGroup
	)
	sem := make(chan struct{}, 8)
	for _, it := range items {
		it := it
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if svc, ok := resolveAvahi(ctx, server, it); ok {
				mu.Lock()
				out = append(out, svc)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return dedupeServices(out), nil
}

func resolveAvahi(ctx context.Context, server dbus.BusObject, it avahiItem) (Service, bool) {
	rctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var (
		riface, rproto               int32
		rname, rtype, rdomain, rhost string
		raproto                      int32
		raddr                        string
		rport                        uint16
		rtxt                         [][]byte
		rflags                       uint32
	)
	call := server.CallWithContext(rctx, "org.freedesktop.Avahi.Server.ResolveService", 0,
		it.iface, it.proto, it.name, it.typ, it.domain, avahiProtoUnspec, uint32(0))
	if call.Err != nil {
		return Service{}, false
	}
	if call.Store(&riface, &rproto, &rname, &rtype, &rdomain, &rhost, &raproto, &raddr, &rport, &rtxt, &rflags) != nil {
		return Service{}, false
	}

	svc := Service{
		Instance: rname,
		Type:     strings.TrimSuffix(rtype, "."),
		Host:     strings.TrimSuffix(rhost, "."),
		Port:     int(rport),
		Info:     cleanTXT(avahiTXT(rtxt)),
		Source:   "mDNS",
	}
	if ip := net.ParseIP(raddr); ip != nil && !ip.IsLoopback() && !ip.IsUnspecified() && !ip.IsLinkLocalUnicast() {
		svc.Addrs = []string{raddr}
	}
	return svc, true
}

// avahiTXT turns Avahi's raw TXT byte-arrays (each one a "key=value" entry) into
// strings.
func avahiTXT(raw [][]byte) []string {
	out := make([]string, 0, len(raw))
	for _, b := range raw {
		if len(b) > 0 {
			out = append(out, string(b))
		}
	}
	return out
}
