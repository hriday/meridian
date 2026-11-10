package fwconfig

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// interpret turns a parsed config tree into one Analysis per VDOM (or a single
// Analysis named "" for a non-VDOM config). Real FortiGate multi-VDOM exports
// keep interfaces under `config global` (tagged `set vdom`) and split a VDOM's
// objects across more than one `config vdom` block, so scopes are built by
// merging across all of them.
func interpret(root *Block) ([]Analysis, error) {
	vdomBlocks := root.configs("vdom")
	if len(vdomBlocks) == 0 {
		// single, non-VDOM config: everything lives at the top level.
		scope := []*Block{root}
		a := analyzeScope("", collectEdits(scope, "system interface"), scope)
		return []Analysis{a}, nil
	}

	// VDOM names (ordered, unique) and their object edits, across all vdom blocks.
	var names []string
	seen := map[string]bool{}
	editsByVdom := map[string][]*Block{}
	for _, vb := range vdomBlocks {
		for _, e := range vb.edits() {
			if !seen[e.Name] {
				seen[e.Name] = true
				names = append(names, e.Name)
			}
			editsByVdom[e.Name] = append(editsByVdom[e.Name], e)
		}
	}

	// Interfaces come from `config global` (and any top-level section), tagged
	// with `set vdom`.
	var globalIfaces []*Block
	for _, g := range root.configs("global") {
		globalIfaces = append(globalIfaces, collectEdits([]*Block{g}, "system interface")...)
	}
	globalIfaces = append(globalIfaces, collectEdits([]*Block{root}, "system interface")...)

	out := make([]Analysis, 0, len(names))
	for _, name := range names {
		scope := editsByVdom[name]
		ifaceEdits := filterByVdom(globalIfaces, name)
		// also accept interfaces declared inside the VDOM scope (defensive).
		ifaceEdits = append(ifaceEdits, collectEdits(scope, "system interface")...)
		out = append(out, analyzeScope(name, ifaceEdits, scope))
	}
	return out, nil
}

// analyzeScope builds one VDOM's Analysis from its interface edits and the
// config blocks that hold its firewall/zone sections.
func analyzeScope(name string, ifaceEdits, scope []*Block) Analysis {
	a := newAnalysis(name)
	readInterfaces(ifaceEdits, &a)
	readZones(collectEdits(scope, "system zone"), &a)
	applyZones(&a)
	readAddresses(collectEdits(scope, "firewall address"), collectEdits(scope, "firewall addrgrp"), &a)
	readServices(collectEdits(scope, "firewall service custom"), collectEdits(scope, "firewall service group"), &a)
	readPolicies(collectEdits(scope, "firewall policy"), &a)
	derive(&a)
	return a
}

// collectEdits returns every edit object across all `config <path>` sections
// found directly in the given blocks.
func collectEdits(blocks []*Block, path string) []*Block {
	var out []*Block
	for _, b := range blocks {
		for _, c := range b.configs(path) {
			out = append(out, c.edits()...)
		}
	}
	return out
}

// filterByVdom keeps interface edits tagged for the named VDOM; if none carry a
// `set vdom`, it attributes all of them to the VDOM (single-interface-table
// exports).
func filterByVdom(ifaces []*Block, name string) []*Block {
	var matched []*Block
	for _, e := range ifaces {
		if v, ok := e.get("vdom"); ok && v == name {
			matched = append(matched, e)
		}
	}
	if len(matched) == 0 {
		return ifaces
	}
	return matched
}

func readInterfaces(edits []*Block, a *Analysis) {
	for _, e := range edits {
		if valOr(e, "role", "") == "wan" {
			a.wan[e.Name] = true
		}
		vid, ok := e.get("vlanid")
		if !ok {
			continue // only interfaces with a vlanid are VLANs
		}
		id, _ := strconv.Atoi(vid)
		v := VLAN{ID: id, Name: e.Name}
		if parent, ok := e.get("interface"); ok {
			v.Interface = parent
		}
		if ip, ok := e.get("ip"); ok {
			v.Subnet = ipMaskToCIDR(ip)
		}
		a.VLANs = append(a.VLANs, v)
	}
}

func readZones(edits []*Block, a *Analysis) {
	for _, e := range edits {
		a.Zones = append(a.Zones, Zone{Name: e.Name, Interfaces: e.getAll("interface")})
	}
}

// applyZones sets each VLAN's Zone to the zone whose interface list contains it.
func applyZones(a *Analysis) {
	for i := range a.VLANs {
		for _, z := range a.Zones {
			for _, iface := range z.Interfaces {
				if iface == a.VLANs[i].Name {
					a.VLANs[i].Zone = z.Name
				}
			}
		}
	}
}

func readAddresses(addrs, groups []*Block, a *Analysis) {
	for _, e := range addrs {
		a.Addresses = append(a.Addresses, address(e))
	}
	for _, e := range groups {
		a.Addresses = append(a.Addresses, Address{
			Name: e.Name, Kind: "group", Value: strings.Join(e.getAll("member"), ", "),
		})
	}
}

func address(e *Block) Address {
	a := Address{Name: e.Name, Kind: "subnet"}
	kind, _ := e.get("type")
	switch kind {
	case "iprange":
		a.Kind = "iprange"
		a.Value = fmt.Sprintf("%s-%s", valOr(e, "start-ip", ""), valOr(e, "end-ip", ""))
	case "fqdn":
		a.Kind = "fqdn"
		a.Value = valOr(e, "fqdn", "")
	case "geography":
		a.Kind = "geography"
		a.Value = valOr(e, "country", "")
	default:
		if sub, ok := e.get("subnet"); ok {
			a.Value = ipMaskToCIDR(sub)
		}
	}
	return a
}

func readServices(customs, groups []*Block, a *Analysis) {
	for _, e := range customs {
		s := Service{Name: e.Name, Proto: valOr(e, "protocol", "")}
		s.Ports = firstNonEmpty(valOr(e, "tcp-portrange", ""), valOr(e, "udp-portrange", ""))
		a.Services = append(a.Services, s)
	}
	for _, e := range groups {
		a.Services = append(a.Services, Service{
			Name: e.Name, Proto: "group", Ports: strings.Join(e.getAll("member"), ", "),
		})
	}
}

func readPolicies(edits []*Block, a *Analysis) {
	for _, e := range edits {
		id, _ := strconv.Atoi(e.Name)
		action := valOr(e, "action", "deny") // FortiGate default action is deny
		p := Policy{
			ID:       id,
			Name:     valOr(e, "name", ""),
			SrcIntf:  e.getAll("srcintf"),
			DstIntf:  e.getAll("dstintf"),
			SrcAddr:  e.getAll("srcaddr"),
			DstAddr:  e.getAll("dstaddr"),
			Services: e.getAll("service"),
			Action:   action,
			Disabled: valOr(e, "status", "") == "disable",
			Negated: valOr(e, "srcaddr-negate", "") == "enable" ||
				valOr(e, "dstaddr-negate", "") == "enable" ||
				valOr(e, "service-negate", "") == "enable" ||
				valOr(e, "internet-service", "") == "enable",
		}
		a.Policies = append(a.Policies, p)
	}
}

func valOr(b *Block, key, def string) string {
	if v, ok := b.get(key); ok {
		return v
	}
	return def
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// ipMaskToCIDR turns "10.0.10.1 255.255.255.0" into "10.0.10.0/24" (the network
// address). Falls back to the raw value if it can't parse.
func ipMaskToCIDR(ipMask string) string {
	parts := strings.Fields(ipMask)
	if len(parts) != 2 {
		return ipMask
	}
	ip := net.ParseIP(parts[0]).To4()
	maskIP := net.ParseIP(parts[1]).To4()
	if ip == nil || maskIP == nil {
		return ipMask
	}
	mask := net.IPv4Mask(maskIP[0], maskIP[1], maskIP[2], maskIP[3])
	ones, _ := mask.Size()
	network := ip.Mask(mask)
	return fmt.Sprintf("%s/%d", network.String(), ones)
}
