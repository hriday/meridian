package fwconfig

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// analyzePanos reads a Palo Alto PAN-OS config in `set`-format (the output of
// `set cli config-output-format set`) and fills one Analysis. Real set-format
// output puts each leaf on its own line, so rules and zones are accumulated
// across lines keyed by name. XML exports are rejected with a clear message.
// Only sections that feed the model are read; anything else (ike psk, logging,
// certificates) is ignored, so secrets are never surfaced.
func analyzePanos(configText string) ([]Analysis, error) {
	if strings.TrimSpace(configText) == "" {
		return nil, fmt.Errorf("panos: no config provided")
	}
	if firstMeaningful(configText) == '<' {
		return nil, fmt.Errorf("panos: this looks like XML; paste the 'set'-format config (set cli config-output-format set)")
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
		if iface != "" {
			s.Interface = iface
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

	rules := map[string]*Policy{}
	var ruleOrder []string
	nextID := 0
	getRule := func(name string) *Policy {
		p, ok := rules[name]
		if !ok {
			nextID++
			p = &Policy{ID: nextID, Name: name}
			rules[name] = p
			ruleOrder = append(ruleOrder, name)
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
		toks = toks[1:] // drop "set"

		switch toks[0] {
		case "network":
			if len(toks) >= 5 && toks[1] == "interface" && toks[4] == "layer3" {
				iface := toks[3]
				rest := toks[5:]
				switch {
				case len(rest) >= 2 && rest[0] == "ip":
					getSeg(iface, iface).Subnet = cidrNetwork(rest[1])
				case len(rest) >= 4 && rest[0] == "units" && rest[2] == "ip":
					getSeg(rest[1], iface).Subnet = cidrNetwork(rest[3])
				case len(rest) >= 4 && rest[0] == "units" && rest[2] == "tag":
					getSeg(rest[1], iface).ID = atoiSafe(rest[3])
				}
			}
		case "zone":
			if len(toks) >= 5 && toks[2] == "network" && toks[3] == "layer3" {
				for _, m := range parseList(toks[4:]) {
					addZoneMember(toks[1], m)
				}
			}
		case "address":
			if len(toks) >= 4 {
				x := getAddr(toks[1])
				switch toks[2] {
				case "ip-netmask":
					x.Kind, x.Value = "subnet", toks[3]
				case "ip-range":
					x.Kind, x.Value = "iprange", toks[3]
				case "fqdn":
					x.Kind, x.Value = "fqdn", toks[3]
				}
			}
		case "address-group":
			if len(toks) >= 4 && toks[2] == "static" {
				x := getAddr(toks[1])
				x.Kind = "group"
				x.Value = strings.Join(parseList(toks[3:]), ", ")
			}
		case "service":
			// set service <N> protocol tcp|udp port <p> [source-port <p>]
			if len(toks) >= 2 {
				s := getSvc(toks[1])
				if proto, ok := tokenAfter(toks, "protocol"); ok {
					s.Proto = strings.ToUpper(proto)
				}
				if port, ok := tokenAfter(toks, "port"); ok {
					s.Ports = port
				}
			}
		case "service-group":
			if len(toks) >= 4 && toks[2] == "members" {
				s := getSvc(toks[1])
				s.Proto = "group"
				s.Ports = strings.Join(parseList(toks[3:]), ", ")
			}
		case "rulebase":
			// set rulebase security rules <Name> <field> <value...>
			if len(toks) >= 6 && toks[1] == "security" && toks[2] == "rules" {
				p := getRule(toks[3])
				field, vals := toks[4], parseList(toks[5:])
				applyPanosRuleField(p, field, vals)
			}
		}
	}

	for _, n := range segOrder {
		a.VLANs = append(a.VLANs, *segs[n])
	}
	for _, z := range zoneOrder {
		a.Zones = append(a.Zones, Zone{Name: z, Interfaces: zoneMembers[z]})
	}
	// segment → zone label, and Internet-facing zones → wan members
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
	// re-emit VLANs with zone labels filled
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
	for _, n := range ruleOrder {
		a.Policies = append(a.Policies, *rules[n])
	}

	derive(&a)
	return []Analysis{a}, nil
}

// applyPanosRuleField fills one accumulated field of a PAN-OS security rule.
func applyPanosRuleField(p *Policy, field string, vals []string) {
	switch field {
	case "from":
		p.SrcIntf = append(p.SrcIntf, vals...)
	case "to":
		p.DstIntf = append(p.DstIntf, vals...)
	case "source":
		p.SrcAddr = append(p.SrcAddr, vals...)
	case "destination":
		p.DstAddr = append(p.DstAddr, vals...)
	case "application":
		for _, app := range vals {
			if app != "" && app != "any" {
				p.Services = append(p.Services, "app:"+app)
			}
		}
	case "service":
		for _, s := range vals {
			if s != "" && s != "any" && s != "application-default" {
				p.Services = append(p.Services, s)
			}
		}
	case "action":
		if len(vals) > 0 {
			p.Action = normalizeAction(vals[0])
		}
	case "disabled":
		if len(vals) > 0 && isYes(vals[0]) {
			p.Disabled = true
		}
	case "negate-source", "negate-destination":
		if len(vals) > 0 && isYes(vals[0]) {
			p.Negated = true
		}
	}
}

// --- shared helpers (also used by the SRX analyzer) ---

// parseList returns a value that is either a single token or a bracketed list
// `[ a b c ]`, with the brackets stripped.
func parseList(toks []string) []string {
	if len(toks) == 0 {
		return nil
	}
	if toks[0] == "[" {
		var out []string
		for _, t := range toks[1:] {
			if t == "]" {
				break
			}
			out = append(out, t)
		}
		return out
	}
	return []string{toks[0]}
}

// tokenAfter returns the token following key, if present.
func tokenAfter(toks []string, key string) (string, bool) {
	for i := 0; i+1 < len(toks); i++ {
		if toks[i] == key {
			return toks[i+1], true
		}
	}
	return "", false
}

// cidrNetwork turns an interface address like "203.0.113.1/24" into its network
// "203.0.113.0/24"; a bare host or unparseable value is returned unchanged.
func cidrNetwork(s string) string {
	if _, ipnet, err := net.ParseCIDR(s); err == nil {
		return ipnet.String()
	}
	return s
}

func atoiSafe(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func isYes(s string) bool { return strings.EqualFold(s, "yes") || strings.EqualFold(s, "enable") }

// normalizeAction maps a vendor action word to the model's accept|deny.
func normalizeAction(a string) string {
	switch strings.ToLower(a) {
	case "allow", "permit", "accept":
		return "accept"
	default: // deny, drop, reset-client/server/both, reject, block
		return "deny"
	}
}

// isWanZone reports whether a zone name conventionally faces the Internet.
func isWanZone(name string) bool {
	switch strings.ToLower(name) {
	case "untrust", "internet", "outside", "wan", "external":
		return true
	}
	return false
}

// firstMeaningful returns the first non-space, non-comment, non-blank byte of a
// config (used to sniff XML / curly-brace formats), or 0 if none.
func firstMeaningful(text string) byte {
	for _, raw := range strings.Split(text, "\n") {
		t := strings.TrimSpace(raw)
		if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "!") {
			continue
		}
		return t[0]
	}
	return 0
}
