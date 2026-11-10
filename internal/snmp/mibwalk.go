package snmp

import (
	"encoding/hex"
	"fmt"
	"unicode"
	"unicode/utf8"

	g "github.com/gosnmp/gosnmp"
)

// MIBRow is one OID in a MIB walk, rendered for display.
type MIBRow struct {
	OID   string `json:"oid"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

// typeName maps a gosnmp ASN.1 type to a readable name.
func typeName(t g.Asn1BER) string {
	switch t {
	case g.OctetString:
		return "OctetString"
	case g.Integer:
		return "Integer"
	case g.Counter32:
		return "Counter32"
	case g.Counter64:
		return "Counter64"
	case g.Gauge32:
		return "Gauge32"
	case g.TimeTicks:
		return "TimeTicks"
	case g.IPAddress:
		return "IpAddress"
	case g.ObjectIdentifier:
		return "OID"
	case g.Uinteger32:
		return "Unsigned32"
	case g.Null:
		return "Null"
	case g.NoSuchObject:
		return "noSuchObject"
	case g.NoSuchInstance:
		return "noSuchInstance"
	case g.EndOfMibView:
		return "endOfMibView"
	default:
		return "—"
	}
}

// MIBWalk walks the subtree at root and returns each OID with its type and a
// display value. A root with no subtree (a leaf instance) falls back to a direct
// Get so a single value still returns a row. The result is capped at max.
func MIBWalk(c Client, root string, max int) ([]MIBRow, error) {
	pdus, err := c.Walk(root)
	if err != nil {
		return nil, err
	}
	if len(pdus) == 0 { // leaf: try the exact OID
		got, gerr := c.Get([]string{root})
		if gerr != nil {
			return nil, gerr
		}
		for _, p := range got {
			if p.Value != nil {
				pdus = append(pdus, p)
			}
		}
	}
	if max > 0 && len(pdus) > max {
		pdus = pdus[:max]
	}
	rows := make([]MIBRow, len(pdus))
	for i, p := range pdus {
		rows[i] = MIBRow{OID: p.OID, Type: p.Type, Value: formatMIBValue(p.Value)}
	}
	return rows, nil
}

// formatMIBValue renders a decoded value for display: printable text as-is, a
// binary OctetString (e.g. a MAC) as hex, everything else via fmt.
func formatMIBValue(v any) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		if isPrintable(val) {
			return val
		}
		return hex.EncodeToString([]byte(val))
	default:
		return fmt.Sprint(v)
	}
}

// isPrintable reports whether s is text safe to show as-is (valid UTF-8 with no
// control runes other than tab/newline/CR).
func isPrintable(s string) bool {
	if !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if r == '\t' || r == '\n' || r == '\r' {
			continue
		}
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
