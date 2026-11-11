package fwconfig

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// analyzePfsense reads a pfSense or OPNsense config.xml and fills one Analysis.
// pf rules live under <filter> as <rule> elements and are bound to an interface
// (direction in by default) with address-based source/destination, so — like
// ASA — flows show the bound interface with "Internet (any)" standing in for the
// destinations, and each policy keeps the real source/destination addresses.
// Interfaces (<interfaces><wan>/<lan>/<opt1>…) become the segments/zones, named
// by their <descr>. Secrets (VPN keys, user passwords) live under other config
// sections the parser does not read.
func analyzePfsense(configText string) ([]Analysis, error) {
	s := strings.TrimSpace(configText)
	if s == "" {
		return nil, fmt.Errorf("pfsense: no config provided")
	}
	if !strings.HasPrefix(s, "<") {
		return nil, fmt.Errorf("pfsense: not XML — paste the firewall's config.xml (Diagnostics ▸ Backup/Restore)")
	}
	var cfg pfConfig
	if err := xml.Unmarshal([]byte(s), &cfg); err != nil {
		return nil, fmt.Errorf("pfsense: couldn't parse config.xml: %w", err)
	}

	a := newAnalysis("")

	// Interfaces → friendly name + segment. Keep document order.
	friendly := map[string]string{}
	for _, ifc := range cfg.Interfaces.Ifaces {
		id := ifc.XMLName.Local
		name := strings.TrimSpace(ifc.Descr)
		if name == "" {
			name = strings.ToUpper(id)
		}
		friendly[id] = name
		subnet := ""
		if ip := strings.TrimSpace(ifc.IPAddr); ip != "" && ip != "dhcp" && ip != "dhcp6" && !strings.Contains(ip, " ") {
			if ifc.Subnet != "" {
				subnet = ip + "/" + ifc.Subnet
			}
		}
		a.VLANs = append(a.VLANs, VLAN{Name: name, Interface: ifc.If, Subnet: subnet, Zone: name})
		a.Zones = append(a.Zones, Zone{Name: name, Interfaces: []string{name}})
	}
	nameOf := func(id string) string {
		id = strings.TrimSpace(id)
		if f, ok := friendly[id]; ok {
			return f
		}
		if id == "" {
			return ""
		}
		return strings.ToUpper(id)
	}

	nextID := 0
	for _, r := range cfg.Filter.Rules {
		action := "deny"
		if strings.EqualFold(strings.TrimSpace(r.Type), "pass") {
			action = "accept"
		}
		var srcIntf []string
		for _, part := range strings.Split(r.Interface, ",") {
			if n := nameOf(part); n != "" {
				srcIntf = append(srcIntf, n)
			}
		}
		var services []string
		if svc := pfSvc(r.Protocol, r.Dest.Port); svc != "" {
			services = []string{svc}
		}
		nextID++
		p := Policy{
			ID:       nextID,
			Name:     strings.TrimSpace(r.Descr),
			SrcIntf:  srcIntf,
			Services: services,
			Action:   action,
			Disabled: r.Disabled != nil,
		}
		if v := pfEndpoint(r.Source); v != "" {
			p.SrcAddr = []string{v}
		}
		if v := pfEndpoint(r.Dest); v != "" {
			p.DstAddr = []string{v}
		}
		if p.Name == "" {
			p.Name = fmt.Sprintf("rule %d", nextID)
		}
		a.Policies = append(a.Policies, p)
	}

	if len(a.Policies) > 0 {
		a.Notes = append(a.Notes, "pf rules are bound to an interface and filtered by address; flows show the bound interface, with \"Internet (any)\" standing in for the destinations the rule permits — see each policy's source/destination addresses for specifics.")
	}

	derive(&a)
	return []Analysis{a}, nil
}

type pfConfig struct {
	Interfaces pfInterfaces `xml:"interfaces"`
	Filter     pfFilter     `xml:"filter"`
}

type pfInterfaces struct {
	Ifaces []pfIface `xml:",any"`
}

type pfIface struct {
	XMLName xml.Name
	If      string `xml:"if"`
	Descr   string `xml:"descr"`
	IPAddr  string `xml:"ipaddr"`
	Subnet  string `xml:"subnet"`
}

type pfFilter struct {
	Rules []pfRule `xml:"rule"`
}

type pfRule struct {
	Type      string  `xml:"type"`
	Interface string  `xml:"interface"`
	Protocol  string  `xml:"protocol"`
	Descr     string  `xml:"descr"`
	Disabled  *string `xml:"disabled"` // present (even empty) ⇒ disabled
	Source    pfEnd   `xml:"source"`
	Dest      pfEnd   `xml:"destination"`
}

type pfEnd struct {
	Any     *string `xml:"any"`
	Network string  `xml:"network"`
	Address string  `xml:"address"`
	Port    string  `xml:"port"`
}

// pfEndpoint renders a source/destination as a readable address string.
func pfEndpoint(e pfEnd) string {
	switch {
	case e.Any != nil:
		return "any"
	case strings.TrimSpace(e.Network) != "":
		return strings.TrimSpace(e.Network) // interface net or alias
	case strings.TrimSpace(e.Address) != "":
		return strings.TrimSpace(e.Address)
	default:
		return ""
	}
}

func pfSvc(proto, port string) string {
	proto = strings.ToLower(strings.TrimSpace(proto))
	port = strings.ReplaceAll(strings.TrimSpace(port), ":", "-") // pf range a:b → a-b
	switch {
	case port != "" && proto != "" && proto != "ip" && proto != "any":
		return proto + "/" + port
	case port != "":
		return port
	case proto != "" && proto != "ip" && proto != "any":
		return proto
	default:
		return ""
	}
}
