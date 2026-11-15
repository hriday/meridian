// Package discover finds devices on the local network by what they advertise:
// mDNS / DNS-SD (Bonjour — printers, Chromecasts, AirPlay, NAS, _ssh/_smb, …)
// and SSDP / UPnP (routers, smart TVs, media servers, IoT). It's the "what is
// this device, and what does it offer" layer on top of a ping or ARP sweep,
// which only tell you an address is alive.
//
// Both are multicast UDP and need no privileges. Point-in-time: it listens for
// a short window and reports what answered; nothing is stored.
package discover

import (
	"sort"
	"strings"

	"github.com/miekg/dns"
)

// Service is one mDNS / DNS-SD advertised service instance.
type Service struct {
	Instance string   `json:"instance"` // friendly name, e.g. "Living Room"
	Type     string   `json:"type"`     // e.g. _airplay._tcp
	Host     string   `json:"host"`     // e.g. livingroom.local
	Port     int      `json:"port"`
	Addrs    []string `json:"addrs"`
	Info     []string `json:"info"`   // selected TXT key=value entries
	Source   string   `json:"source"` // "mDNS"
}

// Device is one SSDP / UPnP responder.
type Device struct {
	Address  string `json:"address"`  // responder IP
	Location string `json:"location"` // LOCATION (device description URL)
	Server   string `json:"server"`   // SERVER header
	Type     string `json:"type"`     // ST / NT
	USN      string `json:"usn"`
	Source   string `json:"source"` // "SSDP"
}

// Result is a discovery pass.
type Result struct {
	Services []Service `json:"services"`
	Devices  []Device  `json:"devices"`
}

const dnssdMeta = "_services._dns-sd._udp"

// parseMDNS turns raw mDNS response packets into advertised service instances.
func parseMDNS(payloads [][]byte) []Service {
	type inst struct {
		typ, host string
		port      int
		txt       []string
	}
	insts := map[string]*inst{}
	hostAddrs := map[string][]string{}
	entry := func(name string) *inst {
		if insts[name] == nil {
			insts[name] = &inst{}
		}
		return insts[name]
	}

	for _, b := range payloads {
		var m dns.Msg
		if err := m.Unpack(b); err != nil {
			continue
		}
		for _, rr := range append(append([]dns.RR{}, m.Answer...), m.Extra...) {
			switch x := rr.(type) {
			case *dns.PTR:
				// Skip the meta service-type enumeration — its target is a type,
				// not an instance.
				if strings.HasPrefix(strings.ToLower(x.Hdr.Name), dnssdMeta) {
					continue
				}
				e := entry(x.Ptr)
				e.typ = serviceType(x.Hdr.Name)
			case *dns.SRV:
				e := entry(x.Hdr.Name)
				e.host = strings.TrimSuffix(x.Target, ".")
				e.port = int(x.Port)
			case *dns.TXT:
				e := entry(x.Hdr.Name)
				e.txt = append(e.txt, x.Txt...)
			case *dns.A:
				h := strings.TrimSuffix(x.Hdr.Name, ".")
				hostAddrs[h] = appendUnique(hostAddrs[h], x.A.String())
			case *dns.AAAA:
				h := strings.TrimSuffix(x.Hdr.Name, ".")
				hostAddrs[h] = appendUnique(hostAddrs[h], x.AAAA.String())
			}
		}
	}

	var out []Service
	for name, e := range insts {
		if e.typ == "" {
			continue // incomplete entry (a stray SRV/TXT with no matching PTR)
		}
		s := Service{
			Instance: instanceLabel(name),
			Type:     e.typ,
			Host:     e.host,
			Port:     e.port,
			Info:     cleanTXT(e.txt),
			Source:   "mDNS",
		}
		if e.host != "" {
			s.Addrs = hostAddrs[e.host]
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Type != out[j].Type {
			return out[i].Type < out[j].Type
		}
		return out[i].Instance < out[j].Instance
	})
	return out
}

// serviceType reduces a record owner name like "_airplay._tcp.local." to
// "_airplay._tcp".
func serviceType(name string) string {
	n := strings.TrimSuffix(strings.TrimSuffix(name, "."), ".local")
	return n
}

// instanceLabel returns the first (instance) label of an mDNS instance name,
// with DNS escaping removed.
func instanceLabel(fqdn string) string {
	labels := dns.SplitDomainName(fqdn)
	if len(labels) == 0 {
		return strings.TrimSuffix(fqdn, ".")
	}
	return unescapeLabel(labels[0])
}

func unescapeLabel(s string) string {
	// miekg/dns escapes spaces and dots as \ + char; undo the common cases.
	s = strings.ReplaceAll(s, `\ `, " ")
	s = strings.ReplaceAll(s, `\.`, ".")
	return s
}

// cleanTXT drops empty entries and caps how many TXT records are kept.
func cleanTXT(txt []string) []string {
	var out []string
	for _, t := range txt {
		t = strings.TrimSpace(t)
		if t != "" {
			out = append(out, t)
		}
		if len(out) >= 8 {
			break
		}
	}
	return out
}

func appendUnique(s []string, v string) []string {
	for _, x := range s {
		if x == v {
			return s
		}
	}
	return append(s, v)
}

// parseSSDP parses one SSDP response/announcement into a Device. ok is false if
// the datagram isn't a usable SSDP message.
func parseSSDP(raw, addr string) (Device, bool) {
	lines := strings.Split(raw, "\n")
	if len(lines) == 0 {
		return Device{}, false
	}
	first := strings.ToUpper(strings.TrimSpace(lines[0]))
	if !strings.HasPrefix(first, "HTTP/1.1 200") && !strings.HasPrefix(first, "NOTIFY") {
		return Device{}, false
	}
	d := Device{Address: addr, Source: "SSDP"}
	for _, ln := range lines[1:] {
		i := strings.IndexByte(ln, ':')
		if i < 0 {
			continue
		}
		key := strings.ToUpper(strings.TrimSpace(ln[:i]))
		val := strings.TrimSpace(ln[i+1:])
		switch key {
		case "LOCATION":
			d.Location = val
		case "SERVER":
			d.Server = val
		case "ST", "NT":
			if d.Type == "" {
				d.Type = val
			}
		case "USN":
			d.USN = val
		}
	}
	if d.Location == "" && d.Type == "" && d.USN == "" {
		return Device{}, false
	}
	return d, true
}

// dedupeServices collapses service instances that appear more than once (e.g.
// the same instance seen on several interfaces), unioning their addresses.
func dedupeServices(in []Service) []Service {
	type key struct{ typ, inst string }
	idx := map[key]int{}
	var out []Service
	for _, s := range in {
		k := key{s.Type, s.Instance}
		if i, ok := idx[k]; ok {
			for _, a := range s.Addrs {
				out[i].Addrs = appendUnique(out[i].Addrs, a)
			}
			if out[i].Host == "" {
				out[i].Host = s.Host
			}
			if out[i].Port == 0 {
				out[i].Port = s.Port
			}
			if len(out[i].Info) == 0 {
				out[i].Info = s.Info
			}
			continue
		}
		idx[k] = len(out)
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Type != out[j].Type {
			return out[i].Type < out[j].Type
		}
		return out[i].Instance < out[j].Instance
	})
	return out
}

// dedupeDevices collapses SSDP responses from the same USN (or address+type).
func dedupeDevices(in []Device) []Device {
	seen := map[string]bool{}
	var out []Device
	for _, d := range in {
		key := d.USN
		if key == "" {
			key = d.Address + "|" + d.Type
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Address != out[j].Address {
			return out[i].Address < out[j].Address
		}
		return out[i].Type < out[j].Type
	})
	return out
}
