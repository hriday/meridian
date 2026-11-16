package ndp

import "strings"

// parseDarwinRouters parses `ndp -rn` (the learned IPv6 default-router list):
//
//	fe80::1%en0 if=en0, flags=IST, pref=medium, expire=29m50s
func parseDarwinRouters(out string) []Router {
	var rs []Router
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) == 0 || !strings.Contains(f[0], ":") {
			continue
		}
		addr, zone := splitZone(f[0])
		if addr == "fe80::" || addr == "::" {
			continue // a bare prefix (e.g. a VPN tunnel's default route), not a real router
		}
		r := Router{Address: addr, Interface: zone}
		for _, tok := range f[1:] {
			tok = strings.TrimSuffix(tok, ",")
			if v, ok := strings.CutPrefix(tok, "if="); ok && r.Interface == "" {
				r.Interface = v
			} else if v, ok := strings.CutPrefix(tok, "pref="); ok {
				r.Preference = v
			} else if v, ok := strings.CutPrefix(tok, "expire="); ok {
				r.Expire = v
			}
		}
		rs = append(rs, r)
	}
	return rs
}

// parseLinuxRouters parses `ip -6 route show default`:
//
//	default via fe80::1 dev eth0 proto ra metric 1024 expires 1512sec pref medium
func parseLinuxRouters(out string) []Router {
	var rs []Router
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) < 2 || f[0] != "default" {
			continue
		}
		var r Router
		for i := 1; i < len(f); i++ {
			switch {
			case f[i] == "via" && i+1 < len(f):
				r.Address = f[i+1]
				i++
			case f[i] == "dev" && i+1 < len(f):
				r.Interface = f[i+1]
				i++
			case f[i] == "pref" && i+1 < len(f):
				r.Preference = f[i+1]
				i++
			case f[i] == "expires" && i+1 < len(f):
				r.Expire = f[i+1]
				i++
			}
		}
		if r.Address != "" {
			rs = append(rs, r)
		}
	}
	return rs
}

// parseWindowsRouters parses `netsh interface ipv6 show route`, taking the
// gateways of the default route (::/0):
//
//	No    Manual  256  ::/0   12  fe80::1
func parseWindowsRouters(out string) []Router {
	var rs []Router
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(strings.TrimSpace(line))
		if len(f) < 2 {
			continue
		}
		dflt := false
		for _, tok := range f {
			if tok == "::/0" {
				dflt = true
				break
			}
		}
		if !dflt {
			continue
		}
		gw := f[len(f)-1]
		if !strings.Contains(gw, ":") {
			continue // last column is an interface name, not a gateway
		}
		addr, zone := splitZone(gw)
		rs = append(rs, Router{Address: addr, Interface: zone})
	}
	return rs
}
