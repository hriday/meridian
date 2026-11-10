package fwconfig

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// interpret turns a parsed config tree into one Analysis per VDOM (or a single
// Analysis named "" for a non-VDOM config).
func interpret(root *Block) ([]Analysis, error) {
	var scopes []*Block
	var names []string
	if vdoms := root.configs("vdom"); len(vdoms) > 0 {
		for _, e := range vdoms[0].edits() {
			scopes = append(scopes, e)
			names = append(names, e.Name)
		}
	}
	if len(scopes) == 0 {
		scopes = []*Block{root}
		names = []string{""}
	}

	out := make([]Analysis, 0, len(scopes))
	for i, scope := range scopes {
		a := newAnalysis(names[i])
		readInterfaces(scope, &a)
		readZones(scope, &a)
		applyZones(&a)
		readAddresses(scope, &a)
		readServices(scope, &a)
		readPolicies(scope, &a)
		out = append(out, a)
	}
	return out, nil
}

func firstConfig(scope *Block, path string) *Block {
	if cs := scope.configs(path); len(cs) > 0 {
		return cs[0]
	}
	return nil
}

func readInterfaces(scope *Block, a *Analysis) {
	c := firstConfig(scope, "system interface")
	if c == nil {
		return
	}
	for _, e := range c.edits() {
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

func readZones(scope *Block, a *Analysis) {
	c := firstConfig(scope, "system zone")
	if c == nil {
		return
	}
	for _, e := range c.edits() {
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

func readAddresses(scope *Block, a *Analysis) {
	if c := firstConfig(scope, "firewall address"); c != nil {
		for _, e := range c.edits() {
			a.Addresses = append(a.Addresses, address(e))
		}
	}
	if c := firstConfig(scope, "firewall addrgrp"); c != nil {
		for _, e := range c.edits() {
			a.Addresses = append(a.Addresses, Address{
				Name: e.Name, Kind: "group", Value: strings.Join(e.getAll("member"), ", "),
			})
		}
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

func readServices(scope *Block, a *Analysis) {
	if c := firstConfig(scope, "firewall service custom"); c != nil {
		for _, e := range c.edits() {
			s := Service{Name: e.Name, Proto: valOr(e, "protocol", "")}
			ports := firstNonEmpty(valOr(e, "tcp-portrange", ""), valOr(e, "udp-portrange", ""))
			s.Ports = ports
			a.Services = append(a.Services, s)
		}
	}
	if c := firstConfig(scope, "firewall service group"); c != nil {
		for _, e := range c.edits() {
			a.Services = append(a.Services, Service{
				Name: e.Name, Proto: "group", Ports: strings.Join(e.getAll("member"), ", "),
			})
		}
	}
}

func readPolicies(scope *Block, a *Analysis) {
	c := firstConfig(scope, "firewall policy")
	if c == nil {
		return
	}
	for _, e := range c.edits() {
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
