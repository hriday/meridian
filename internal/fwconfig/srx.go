package fwconfig

import (
	"fmt"
	"strings"
)

// analyzeSrx reads a Juniper SRX config in `set`-format (the output of
// `show configuration | display set`) and fills one Analysis. Curly-brace
// exports are rejected with a clear message. Only the security sections that
// feed the model are read, so IKE pre-shared-keys and other secrets — which
// live in sections the parser ignores — are never surfaced.
func analyzeSrx(configText string) ([]Analysis, error) {
	if strings.TrimSpace(configText) == "" {
		return nil, fmt.Errorf("srx: no config provided")
	}
	if !startsWithSet(configText) {
		return nil, fmt.Errorf("srx: this looks like curly-brace format; paste 'show configuration | display set' output")
	}

	a := newAnalysis("")

	segs := map[string]*VLAN{}
	var segOrder []string
	getSeg := func(name, iface string) *VLAN {
		s, ok := segs[name]
		if !ok {
			s = &VLAN{Name: name, Interface: iface}
			segs[name] = s
			segOrder = append(segOrder, name)
		}
		return s
	}

	zoneMembers := map[string][]string{}
	var zoneOrder []string
	addZoneMember := func(z, m string) {
		if _, ok := zoneMembers[z]; !ok {
			zoneOrder = append(zoneOrder, z)
		}
		zoneMembers[z] = append(zoneMembers[z], m)
	}

	addrs := map[string]*Address{}
	var addrOrder []string
	getAddr := func(name string) *Address {
		x, ok := addrs[name]
		if !ok {
			x = &Address{Name: name}
			addrs[name] = x
			addrOrder = append(addrOrder, name)
		}
		return x
	}
	svcs := map[string]*Service{}
	var svcOrder []string
	getSvc := func(name string) *Service {
		s, ok := svcs[name]
		if !ok {
			s = &Service{Name: name}
			svcs[name] = s
			svcOrder = append(svcOrder, name)
		}
		return s
	}

	pols := map[string]*Policy{}
	var polOrder []string
	nextID := 0
	getPol := func(from, to, name string) *Policy {
		k := from + "\x00" + to + "\x00" + name
		p, ok := pols[k]
		if !ok {
			nextID++
			p = &Policy{ID: nextID, Name: name, SrcIntf: []string{from}, DstIntf: []string{to}}
			pols[k] = p
			polOrder = append(polOrder, k)
		}
		return p
	}

	for _, raw := range strings.Split(configText, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		toks := tokenize(line)
		if len(toks) < 2 || toks[0] != "set" {
			continue
		}
		toks = toks[1:]

		switch toks[0] {
		case "interfaces":
			// interfaces <if> unit <u> [vlan-id <n>] [family inet address <cidr>]
			if len(toks) >= 4 && toks[2] == "unit" {
				name := toks[1] + "." + toks[3]
				s := getSeg(name, toks[1])
				if v, ok := tokenAfter(toks, "vlan-id"); ok {
					s.ID = atoiSafe(v)
				}
				if addr, ok := tokenAfter(toks, "address"); ok {
					s.Subnet = cidrNetwork(addr)
				}
			}
		case "security":
			if len(toks) < 2 {
				continue
			}
			switch toks[1] {
			case "zones":
				// security zones security-zone <Z> interfaces <ifunit>
				if len(toks) >= 6 && toks[2] == "security-zone" && toks[4] == "interfaces" {
					addZoneMember(toks[3], toks[5])
				}
			case "address-book":
				// security address-book <book> address <N> <value...>
				// security address-book <book> address-set <N> address <m>
				if len(toks) >= 6 && toks[3] == "address" {
					applySrxAddress(getAddr(toks[4]), toks[5:])
				} else if len(toks) >= 7 && toks[3] == "address-set" && toks[5] == "address" {
					x := getAddr(toks[4])
					x.Kind = "group"
					x.Value = appendCSV(x.Value, toks[6])
				}
			case "policies":
				// security policies from-zone <X> to-zone <Y> policy <P> match|then ...
				// security policies global policy <P> match|then ...  (zone-less)
				switch {
				case len(toks) >= 9 && toks[2] == "from-zone" && toks[4] == "to-zone" && toks[6] == "policy":
					p := getPol(toks[3], toks[5], toks[7])
					applySrxPolicy(p, toks[8:])
				case len(toks) >= 5 && toks[2] == "global" && toks[3] == "policy":
					p := getPol("any", "any", toks[4])
					applySrxPolicy(p, toks[5:])
				}
			}
		case "applications":
			switch {
			case len(toks) >= 3 && toks[1] == "application":
				s := getSvc(toks[2])
				if proto, ok := tokenAfter(toks, "protocol"); ok {
					s.Proto = strings.ToUpper(proto)
				}
				if port, ok := tokenAfter(toks, "destination-port"); ok {
					s.Ports = port
				}
			case len(toks) >= 5 && toks[1] == "application-set" && toks[3] == "application":
				s := getSvc(toks[2])
				s.Proto = "group"
				s.Ports = appendCSV(s.Ports, toks[4])
			}
		}
	}

	for _, n := range segOrder {
		a.VLANs = append(a.VLANs, *segs[n])
	}
	for _, z := range zoneOrder {
		a.Zones = append(a.Zones, Zone{Name: z, Interfaces: zoneMembers[z]})
	}
	for _, z := range a.Zones {
		wan := isWanZone(z.Name)
		for _, m := range z.Interfaces {
			if s, ok := segs[m]; ok {
				s.Zone = z.Name
			}
			if wan {
				a.wan[m] = true
			}
		}
	}
	a.VLANs = a.VLANs[:0]
	for _, n := range segOrder {
		a.VLANs = append(a.VLANs, *segs[n])
	}
	for _, n := range addrOrder {
		a.Addresses = append(a.Addresses, *addrs[n])
	}
	for _, n := range svcOrder {
		a.Services = append(a.Services, *svcs[n])
	}
	for _, k := range polOrder {
		a.Policies = append(a.Policies, *pols[k])
	}

	derive(&a)
	return []Analysis{a}, nil
}

// applySrxAddress fills an address-book address object from its value tokens.
func applySrxAddress(x *Address, vals []string) {
	if len(vals) == 0 {
		return
	}
	switch vals[0] {
	case "dns-name":
		if len(vals) >= 2 {
			x.Kind, x.Value = "fqdn", vals[1]
		}
	case "range-address":
		// range-address <a> to <b>
		if len(vals) >= 3 {
			x.Kind, x.Value = "iprange", vals[1]+"-"+vals[len(vals)-1]
		}
	case "wildcard-address":
		if len(vals) >= 2 {
			x.Kind, x.Value = "wildcard", vals[1]
		}
	default:
		x.Kind, x.Value = "subnet", vals[0]
	}
}

// applySrxPolicy folds one `match`/`then` line into an accumulated policy.
func applySrxPolicy(p *Policy, toks []string) {
	if len(toks) < 2 {
		return
	}
	switch toks[0] {
	case "match":
		field, val := toks[1], toks[2:]
		switch field {
		case "source-address":
			p.SrcAddr = append(p.SrcAddr, val...)
		case "destination-address":
			p.DstAddr = append(p.DstAddr, val...)
		case "application":
			for _, app := range val {
				if app != "" && app != "any" {
					p.Services = append(p.Services, app)
				}
			}
		}
	case "then":
		// Only the action verb sets the action; `then log`/`then count`/
		// `then log session-init` etc. are not actions and must not clobber it.
		switch toks[1] {
		case "permit", "deny", "reject":
			p.Action = normalizeAction(toks[1])
		}
	}
}

// startsWithSet reports whether the first meaningful config line begins with
// "set" (distinguishing SRX set-format from curly-brace format).
func startsWithSet(text string) bool {
	for _, raw := range strings.Split(text, "\n") {
		t := strings.TrimSpace(raw)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		return strings.HasPrefix(t, "set ")
	}
	return false
}
