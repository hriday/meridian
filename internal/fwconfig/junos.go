package fwconfig

import (
	"fmt"
	"strings"
)

// analyzeJunos reads a non-SRX Juniper Junos config in set-format
// (`show configuration | display set`) and fills one Analysis from its stateless
// firewall filters — the MX/EX/QFX ACL model, distinct from SRX zone-based
// security policies (handled separately). A filter is a named list of terms;
// each term matches on from-* conditions (source/destination address, port,
// protocol) and ends in a `then` action (accept/discard/reject). Filters are
// bound to an interface unit with `family inet filter input|output <name>`, so
// flows show the bound interface with "Internet (any)" standing in for the
// permitted destinations. A term with no explicit action defaults to accept
// (Junos implicitly accepts a matched term that only counts/logs); the implicit
// final discard is not represented. Secrets are never surfaced.
func analyzeJunos(configText string) ([]Analysis, error) {
	if strings.TrimSpace(configText) == "" {
		return nil, fmt.Errorf("junos: no config provided")
	}
	a := newAnalysis("")

	terms := map[string]*termAcc{}
	var termOrder []string
	getTerm := func(filter, term string) *termAcc {
		k := filter + "\x00" + term
		t, ok := terms[k]
		if !ok {
			t = &termAcc{filter: filter, term: term}
			terms[k] = t
			termOrder = append(termOrder, k)
		}
		return t
	}

	bindings := map[string][]bind{} // filter -> interfaces

	segByName := map[string]*VLAN{}
	var segOrder []string
	addSeg := func(name, subnet string) {
		if v, ok := segByName[name]; ok {
			if subnet != "" {
				v.Subnet = subnet
			}
			return
		}
		segOrder = append(segOrder, name)
		segByName[name] = &VLAN{Name: name, Interface: name, Subnet: subnet, Zone: name}
	}

	for _, raw := range strings.Split(configText, "\n") {
		toks := tokenize(strings.TrimSpace(raw))
		if len(toks) < 2 || toks[0] != "set" {
			continue
		}
		switch toks[1] {
		case "firewall":
			parseJunosFirewall(toks, getTerm)
		case "interfaces":
			parseJunosInterface(toks, addSeg, bindings)
		}
	}

	// Build policies in first-seen term order.
	aces := map[string][]*Policy{}
	var aclOrder []string
	nextID := 0
	for _, k := range termOrder {
		t := terms[k]
		var services []string
		proto := ""
		if len(t.proto) > 0 {
			proto = strings.ToLower(t.proto[0])
		}
		for _, p := range t.ports {
			if proto != "" {
				services = append(services, proto+"/"+p)
			} else {
				services = append(services, p)
			}
		}
		if len(services) == 0 && proto != "" && proto != "ip" {
			services = []string{proto}
		}
		action := t.action
		if action == "" {
			action = "accept"
		}
		nextID++
		p := &Policy{
			ID:       nextID,
			Name:     t.filter,
			SrcAddr:  orAny(t.src),
			DstAddr:  orAny(t.dst),
			Services: services,
			Action:   normalizeAction(action),
		}
		if _, ok := aces[t.filter]; !ok {
			aclOrder = append(aclOrder, t.filter)
		}
		aces[t.filter] = append(aces[t.filter], p)
	}

	for filter, bs := range bindings {
		var srcIntf, dstIntf []string
		for _, b := range bs {
			if b.dir == "input" {
				srcIntf = append(srcIntf, b.intf)
			} else {
				dstIntf = append(dstIntf, b.intf)
			}
		}
		for _, p := range aces[filter] {
			if len(srcIntf) > 0 {
				p.SrcIntf = srcIntf
			}
			if len(dstIntf) > 0 {
				p.DstIntf = dstIntf
			}
		}
	}

	for _, n := range segOrder {
		a.VLANs = append(a.VLANs, *segByName[n])
		a.Zones = append(a.Zones, Zone{Name: n, Interfaces: []string{n}})
	}
	for _, acl := range aclOrder {
		for _, p := range aces[acl] {
			a.Policies = append(a.Policies, *p)
		}
	}
	if len(a.Policies) > 0 {
		a.Notes = append(a.Notes, "Junos firewall filters are stateless and bound to an interface unit; flows show the bound interface, with \"Internet (any)\" standing in for the destinations the term permits — see each policy's destination addresses for specifics.")
	}

	derive(&a)
	return []Analysis{a}, nil
}

// parseJunosFirewall reads one `set firewall family <inet|inet6> filter NAME
// term TERM …` line into the term accumulator.
func parseJunosFirewall(toks []string, getTerm func(filter, term string) *termAcc) {
	filter := after(toks, "filter")
	term := after(toks, "term")
	if filter == "" || term == "" {
		return
	}
	t := getTerm(filter, term)
	switch {
	case has(toks, "from", "source-address"):
		t.src = append(t.src, after(toks, "source-address"))
	case has(toks, "from", "destination-address"):
		t.dst = append(t.dst, after(toks, "destination-address"))
	case has(toks, "from", "source-prefix-list"):
		t.src = append(t.src, after(toks, "source-prefix-list"))
	case has(toks, "from", "destination-prefix-list"):
		t.dst = append(t.dst, after(toks, "destination-prefix-list"))
	case has(toks, "from", "destination-port"):
		t.ports = append(t.ports, after(toks, "destination-port"))
	case has(toks, "from", "source-port"):
		t.ports = append(t.ports, "src:"+after(toks, "source-port"))
	case has(toks, "from", "port"):
		t.ports = append(t.ports, after(toks, "port"))
	case has(toks, "from", "protocol"):
		t.proto = append(t.proto, after(toks, "protocol"))
	case contains(toks, "then"):
		if a := after(toks, "then"); a == "accept" || a == "discard" || a == "reject" {
			t.action = a
		}
	}
}

// parseJunosInterface reads an interface's address and filter bindings.
func parseJunosInterface(toks []string, addSeg func(name, subnet string), bindings map[string][]bind) {
	// set interfaces <ifd> unit <n> family <inet|inet6> address <A/M>
	// set interfaces <ifd> unit <n> family <inet|inet6> filter input|output <name>
	ifd := after(toks, "interfaces")
	unit := after(toks, "unit")
	name := ifd
	if unit != "" {
		name = ifd + "." + unit
	}
	switch {
	case contains(toks, "address"):
		if a := after(toks, "address"); strings.Contains(a, "/") {
			addSeg(name, a)
		}
	case has(toks, "filter", "input"):
		addSeg(name, "")
		bindings[after(toks, "input")] = append(bindings[after(toks, "input")], bind{intf: name, dir: "input"})
	case has(toks, "filter", "output"):
		addSeg(name, "")
		bindings[after(toks, "output")] = append(bindings[after(toks, "output")], bind{intf: name, dir: "output"})
	}
}

type bind struct{ intf, dir string }

type termAcc struct {
	filter, term    string
	src, dst, ports []string
	proto           []string
	action          string
}

// after returns the token immediately following the first occurrence of key.
func after(toks []string, key string) string {
	for i, t := range toks {
		if t == key && i+1 < len(toks) {
			return toks[i+1]
		}
	}
	return ""
}

// has reports whether both a and b appear in toks (a before b not required, but
// both present — enough to disambiguate Junos set lines).
func has(toks []string, a, b string) bool { return contains(toks, a) && contains(toks, b) }

func contains(toks []string, s string) bool {
	for _, t := range toks {
		if t == s {
			return true
		}
	}
	return false
}

func orAny(v []string) []string {
	if len(v) == 0 {
		return []string{"any"}
	}
	return v
}
