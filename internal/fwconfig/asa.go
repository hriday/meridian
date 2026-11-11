package fwconfig

import (
	"fmt"
	"net"
	"strings"
)

// analyzeAsa reads a Cisco ASA running-config and fills one Analysis. ASA is
// indentation-structured: interface / object / object-group headers own the
// indented lines beneath them, while access-list / access-group / name are
// flat. ASA ACLs are bound to a single interface and have no destination zone,
// so flows show the bound interface with "Internet (any)" standing in for the
// permitted destinations — a Note records this, and each policy carries the
// real source/destination addresses. Secrets (tunnel-group PSKs, passwords) sit
// in sections the parser does not read and are never surfaced.
func analyzeAsa(configText string) ([]Analysis, error) {
	if strings.TrimSpace(configText) == "" {
		return nil, fmt.Errorf("asa: no config provided")
	}
	a := newAnalysis("")

	type ifaceAcc struct {
		phys, nameif, subnet string
		vlan                 int
	}
	var cur *ifaceAcc
	segByNameif := map[string]*VLAN{}
	var segOrder []string
	flushIface := func() {
		if cur != nil && cur.nameif != "" {
			if _, ok := segByNameif[cur.nameif]; !ok {
				segOrder = append(segOrder, cur.nameif)
			}
			segByNameif[cur.nameif] = &VLAN{Name: cur.nameif, Interface: cur.phys, Subnet: cur.subnet, Zone: cur.nameif, ID: cur.vlan}
		}
		cur = nil
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

	aces := map[string][]*Policy{}
	var aclOrder []string
	nextID := 0
	addAce := func(acl string, p *Policy) {
		if _, ok := aces[acl]; !ok {
			aclOrder = append(aclOrder, acl)
		}
		aces[acl] = append(aces[acl], p)
	}
	type bind struct{ intf, dir string }
	bindings := map[string]bind{}

	var ctx string // "", "objnet", "objsvc", "oggrpnet", "oggrpsvc"
	var curAddr *Address
	var curSvc *Service

	for _, raw := range strings.Split(configText, "\n") {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || trimmed == "!" {
			flushIface()
			ctx, curAddr, curSvc = "", nil, nil
			continue
		}
		indented := raw[0] == ' ' || raw[0] == '\t'
		toks := tokenize(trimmed)
		if len(toks) == 0 {
			continue
		}

		if !indented {
			flushIface()
			ctx, curAddr, curSvc = "", nil, nil
			switch toks[0] {
			case "interface":
				if len(toks) >= 2 {
					cur = &ifaceAcc{phys: toks[1]}
				}
			case "object":
				if len(toks) >= 3 && toks[1] == "network" {
					ctx, curAddr = "objnet", getAddr(toks[2])
				} else if len(toks) >= 3 && toks[1] == "service" {
					ctx, curSvc = "objsvc", getSvc(toks[2])
				}
			case "object-group":
				if len(toks) >= 3 && toks[1] == "network" {
					ctx, curAddr = "oggrpnet", getAddr(toks[2])
					curAddr.Kind = "group"
				} else if len(toks) >= 3 && toks[1] == "service" {
					ctx, curSvc = "oggrpsvc", getSvc(toks[2])
					curSvc.Proto = "group"
				}
			case "access-list":
				if p, acl := parseAsaAce(toks, &nextID); p != nil {
					addAce(acl, p)
				}
			case "access-group":
				// access-group <ACL> <in|out> interface <nameif>
				if len(toks) >= 5 && (toks[2] == "in" || toks[2] == "out") && toks[3] == "interface" {
					bindings[toks[1]] = bind{intf: toks[4], dir: toks[2]}
				}
			}
			continue
		}

		// indented line: belongs to the current interface or object block
		switch {
		case cur != nil:
			switch toks[0] {
			case "nameif":
				if len(toks) >= 2 {
					cur.nameif = toks[1]
				}
			case "ip":
				if len(toks) >= 4 && toks[1] == "address" {
					cur.subnet = ipMask2(toks[2], toks[3])
				}
			case "vlan":
				if len(toks) >= 2 {
					cur.vlan = atoiSafe(toks[1])
				}
			}
		case ctx == "objnet" && curAddr != nil:
			switch toks[0] {
			case "host":
				if len(toks) >= 2 {
					curAddr.Kind, curAddr.Value = "subnet", toks[1]+"/32"
				}
			case "subnet":
				if len(toks) >= 3 {
					curAddr.Kind, curAddr.Value = "subnet", ipMask2(toks[1], toks[2])
				}
			case "range":
				if len(toks) >= 3 {
					curAddr.Kind, curAddr.Value = "iprange", toks[1]+"-"+toks[2]
				}
			case "fqdn":
				// `fqdn v4 www.example.com` or `fqdn www.example.com`
				curAddr.Kind, curAddr.Value = "fqdn", toks[len(toks)-1]
			}
		case ctx == "objsvc" && curSvc != nil:
			// service tcp|udp [source ...] destination eq|range|lt|gt <port>
			if len(toks) >= 2 && toks[0] == "service" {
				curSvc.Proto = strings.ToUpper(toks[1])
				curSvc.Ports = asaPortFromTokens(toks)
			}
		case ctx == "oggrpnet" && curAddr != nil:
			if toks[0] == "network-object" {
				curAddr.Value = appendCSV(curAddr.Value, asaNetworkObject(toks[1:]))
			} else if toks[0] == "group-object" && len(toks) >= 2 {
				curAddr.Value = appendCSV(curAddr.Value, toks[1]) // nested object-group
			}
		case ctx == "oggrpsvc" && curSvc != nil:
			if toks[0] == "service-object" || toks[0] == "port-object" {
				curSvc.Ports = appendCSV(curSvc.Ports, strings.Join(toks[1:], " "))
			} else if toks[0] == "group-object" && len(toks) >= 2 {
				curSvc.Ports = appendCSV(curSvc.Ports, toks[1]) // nested object-group
			}
		}
	}
	flushIface()

	// apply access-group bindings to every ACE of the named ACL
	for acl, b := range bindings {
		for _, p := range aces[acl] {
			if b.dir == "in" {
				p.SrcIntf = []string{b.intf}
			} else {
				p.DstIntf = []string{b.intf}
			}
		}
	}

	for _, n := range segOrder {
		v := segByNameif[n]
		a.VLANs = append(a.VLANs, *v)
		a.Zones = append(a.Zones, Zone{Name: n, Interfaces: []string{n}})
	}
	for _, n := range addrOrder {
		a.Addresses = append(a.Addresses, *addrs[n])
	}
	for _, n := range svcOrder {
		a.Services = append(a.Services, *svcs[n])
	}
	for _, acl := range aclOrder {
		for _, p := range aces[acl] {
			a.Policies = append(a.Policies, *p)
		}
	}
	if len(a.Policies) > 0 {
		a.Notes = append(a.Notes, "ASA access lists are bound to a single interface and have no destination zone; flows show the bound interface, with \"Internet (any)\" standing in for the destinations the rule permits — see each policy's destination addresses for specifics.")
	}

	derive(&a)
	return []Analysis{a}, nil
}

// parseAsaAce parses one `access-list` ACE into a Policy, returning it with the
// ACL name; a remark or unparseable line yields (nil, "").
func parseAsaAce(toks []string, nextID *int) (*Policy, string) {
	if len(toks) < 4 {
		return nil, ""
	}
	acl := toks[1]
	i := 2
	if i < len(toks) && toks[i] == "remark" {
		return nil, ""
	}
	if i+1 < len(toks) && toks[i] == "line" { // access-list X line N ...
		i += 2
	}
	if i < len(toks) && toks[i] == "standard" {
		return nil, "" // standard ACLs aren't interface traffic policies
	}
	if i < len(toks) && toks[i] == "extended" {
		i++
	}
	if i+1 >= len(toks) {
		return nil, ""
	}
	action := normalizeAction(toks[i])
	i++

	// The protocol is a bare token (tcp/udp/ip/…) or a service object reference.
	var proto, svcFromObj string
	if toks[i] == "object" || toks[i] == "object-group" {
		if i+1 < len(toks) {
			svcFromObj = toks[i+1]
			i += 2
		}
	} else {
		proto = strings.ToLower(toks[i])
		i++
	}
	if i >= len(toks) {
		return nil, ""
	}

	src := readAsaSpec(toks, &i)
	skipAsaSrcPort(toks, &i) // optional source-port operator before the dst spec
	dst := readAsaSpec(toks, &i)

	var services []string
	switch {
	case svcFromObj != "":
		services = []string{svcFromObj}
	case proto != "" && proto != "ip":
		if port := readAsaDstPort(toks, &i); port != "" {
			services = []string{proto + "/" + port}
		}
	}

	*nextID++
	return &Policy{
		ID:       *nextID,
		Name:     acl,
		SrcAddr:  []string{src},
		DstAddr:  []string{dst},
		Services: services,
		Action:   action,
	}, acl
}

// skipAsaSrcPort advances i past an optional source-port operator (eq/neq/lt/
// gt <p>, or range <a> <b>) that can sit between the source and destination
// address specs. It never consumes an address keyword like object/host.
func skipAsaSrcPort(toks []string, i *int) {
	if *i >= len(toks) {
		return
	}
	switch toks[*i] {
	case "eq", "neq", "lt", "gt":
		if *i+1 < len(toks) {
			*i += 2
		}
	case "range":
		if *i+2 < len(toks) {
			*i += 3
		}
	}
}

// readAsaDstPort reads an optional destination-port spec after the dst address,
// returning a readable port string, advancing i past it.
func readAsaDstPort(toks []string, i *int) string {
	if *i >= len(toks) {
		return ""
	}
	switch toks[*i] {
	case "eq", "neq":
		if *i+1 < len(toks) {
			v := toks[*i+1]
			*i += 2
			return v
		}
	case "lt":
		if *i+1 < len(toks) {
			v := "<" + toks[*i+1]
			*i += 2
			return v
		}
	case "gt":
		if *i+1 < len(toks) {
			v := ">" + toks[*i+1]
			*i += 2
			return v
		}
	case "range":
		if *i+2 < len(toks) {
			v := toks[*i+1] + "-" + toks[*i+2]
			*i += 3
			return v
		}
	case "object", "object-group":
		if *i+1 < len(toks) {
			v := toks[*i+1]
			*i += 2
			return v
		}
	}
	return ""
}

// asaPortFromTokens extracts the port of an `object service` body, preferring
// the destination port spec when a `destination` keyword is present.
func asaPortFromTokens(toks []string) string {
	start := 0
	for i, t := range toks {
		if t == "destination" {
			start = i
			break
		}
	}
	for i := start; i < len(toks); i++ {
		switch toks[i] {
		case "eq", "neq":
			if i+1 < len(toks) {
				return toks[i+1]
			}
		case "lt":
			if i+1 < len(toks) {
				return "<" + toks[i+1]
			}
		case "gt":
			if i+1 < len(toks) {
				return ">" + toks[i+1]
			}
		case "range":
			if i+2 < len(toks) {
				return toks[i+1] + "-" + toks[i+2]
			}
		}
	}
	return ""
}

// readAsaSpec reads one ASA address spec (any | host <ip> | <ip> <mask> |
// object <n> | object-group <n> | interface <n>) advancing i past it.
func readAsaSpec(toks []string, i *int) string {
	if *i >= len(toks) {
		return ""
	}
	switch toks[*i] {
	case "any", "any4", "any6":
		*i++
		return "any"
	case "host":
		if *i+1 < len(toks) {
			v := toks[*i+1] + "/32"
			*i += 2
			return v
		}
	case "object", "object-group", "interface":
		if *i+1 < len(toks) {
			v := toks[*i+1]
			*i += 2
			return v
		}
	default:
		if *i+1 < len(toks) && net.ParseIP(toks[*i+1]) != nil { // <ip> <mask>
			v := ipMask2(toks[*i], toks[*i+1])
			*i += 2
			return v
		}
	}
	v := toks[*i]
	*i++
	return v
}

// asaNetworkObject renders a `network-object` member as a readable string.
func asaNetworkObject(toks []string) string {
	switch {
	case len(toks) >= 2 && toks[0] == "host":
		return toks[1] + "/32"
	case len(toks) >= 2 && toks[0] == "object":
		return toks[1]
	case len(toks) >= 2 && net.ParseIP(toks[1]) != nil:
		return ipMask2(toks[0], toks[1])
	case len(toks) >= 1:
		return toks[0]
	}
	return ""
}

func appendCSV(existing, add string) string {
	if add == "" {
		return existing
	}
	if existing == "" {
		return add
	}
	return existing + ", " + add
}

// ipMask2 formats an ip + separate mask as the shared ipMaskToCIDR expects.
func ipMask2(ipStr, maskStr string) string { return ipMaskToCIDR(ipStr + " " + maskStr) }
