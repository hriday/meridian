package fwconfig

import (
	"fmt"
	"strings"
)

// Analyze parses a firewall config export and returns one Analysis per VDOM
// (or a single Analysis for a non-VDOM config). vendor defaults to "fortigate",
// the only vendor supported in v1. The config text is processed entirely
// offline and encrypted secrets are never surfaced.
func Analyze(configText, vendor string) ([]Analysis, error) {
	if strings.TrimSpace(configText) == "" {
		return nil, fmt.Errorf("no config provided — paste or load a firewall config export")
	}
	v := strings.TrimSpace(strings.ToLower(vendor))
	if v == "" {
		v = "fortigate"
	}
	switch v {
	case "fortigate":
		root, err := parse(configText)
		if err != nil {
			return nil, err
		}
		return interpret(root)
	case "panos":
		return analyzePanos(configText)
	case "asa":
		return analyzeAsa(configText)
	case "ios", "iosxe", "ios-xe", "nxos", "nx-os":
		return analyzeIOS(configText)
	case "srx":
		return analyzeSrx(configText)
	case "junos":
		return analyzeJunos(configText)
	case "arista", "eos":
		return analyzeArista(configText)
	case "checkpoint", "cp":
		return analyzeCheckpoint(configText)
	case "vyos":
		return analyzeVyos(configText)
	case "pfsense", "opnsense":
		return analyzePfsense(configText)
	default:
		return nil, fmt.Errorf("vendor %q is not supported — use fortigate, panos, asa, ios, nxos, srx, junos, arista, checkpoint, vyos, or pfsense", vendor)
	}
}

// DetectVendor guesses the firewall vendor from distinctive markers in the
// config text, returning a vendor id Analyze accepts, or "" if it can't tell.
// Checks run most-specific first so the shared `ip access-list` / set-format
// families don't collide.
func DetectVendor(configText string) string {
	s := strings.ToLower(configText)
	has := func(subs ...string) bool {
		for _, sub := range subs {
			if strings.Contains(s, sub) {
				return true
			}
		}
		return false
	}
	switch {
	case has("<pfsense>", "<opnsense>"):
		return "pfsense"
	case has("set security policies", "set security zones", "set security address-book"):
		return "srx" // Junos SRX zone policies — distinct from stateless filters
	case has("set firewall family inet") && has("filter"):
		return "junos"
	case has("set rulebase", "set deviceconfig"):
		return "panos"
	case has("set firewall name", "set firewall ipv6-name"):
		return "vyos"
	case has("config firewall policy", "config system global", "config vdom"):
		return "fortigate"
	case has("asa version") || has("\nnameif ", " nameif "):
		return "asa"
	case has("ip access-list", "ip access-group", "interface gigabitethernet", "interface ethernet1/"):
		return "ios" // Cisco IOS / IOS-XE / NX-OS
	}
	return ""
}
