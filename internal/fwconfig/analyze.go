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
	case "srx":
		return analyzeSrx(configText)
	case "arista", "eos":
		return analyzeArista(configText)
	case "checkpoint", "cp":
		return analyzeCheckpoint(configText)
	case "vyos":
		return analyzeVyos(configText)
	default:
		return nil, fmt.Errorf("vendor %q is not supported — use fortigate, panos, asa, srx, arista, checkpoint, or vyos", vendor)
	}
}
