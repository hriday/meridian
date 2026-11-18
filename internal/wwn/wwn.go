// Package wwn decodes a Fibre Channel World Wide Name (WWN / WWPN / WWNN): it
// reports the NAA format, extracts the embedded IEEE OUI (the same registry a
// MAC address uses, so the vendor is resolved via the shared OUI lookup), and
// re-formats the name. Parse is pure and offline; the vendor name is filled in
// by the caller from the OUI.
package wwn

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// Result is a decoded WWN. Vendor is left empty by Parse and filled by the
// caller via an OUI lookup.
type Result struct {
	Input     string   `json:"input"`
	Bare      string   `json:"bare"`      // 20000025b5000001
	Canonical string   `json:"canonical"` // 20:00:00:25:b5:00:00:01
	Dotted    string   `json:"dotted"`    // 2000.0025.b500.0001 (Cisco style)
	Bits      int      `json:"bits"`      // 64, or 128 for NAA 6
	NAA       string   `json:"naa"`       // 1 | 2 | 5 | 6 | ...
	NAADesc   string   `json:"naaDesc"`
	OUI       string   `json:"oui"`     // 00:25:b5 (empty if not derivable)
	OUIBare   string   `json:"ouiBare"` // 0025b5, for the vendor lookup
	VSID      string   `json:"vsid"`    // vendor-specific id (hex)
	Vendor    string   `json:"vendor"`  // filled by the caller from OUIBare
	Notes     []string `json:"notes"`
}

// Parse decodes a WWN in any common spelling (colons, dots, dashes, spaces or
// bare hex). It is pure — no network, no vendor name.
func Parse(input string) (Result, error) {
	clean := strings.ToLower(strings.Map(func(r rune) rune {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f', r >= 'A' && r <= 'F':
			return r
		default:
			return -1 // drop separators
		}
	}, input))
	if len(clean) != 16 && len(clean) != 32 {
		return Result{}, fmt.Errorf("a WWN is 16 hex digits (8 bytes) — or 32 for a 128-bit NAA-6 name; got %d hex digit(s)", len(clean))
	}
	raw, err := hex.DecodeString(clean)
	if err != nil {
		return Result{}, fmt.Errorf("not valid hex")
	}

	r := Result{
		Input:     strings.TrimSpace(input),
		Bare:      clean,
		Canonical: group(clean, 2, ":"),
		Dotted:    group(clean, 4, "."),
		Bits:      len(raw) * 8,
	}
	naa := raw[0] >> 4
	r.NAA = fmt.Sprintf("%x", naa)
	switch naa {
	case 1:
		// NAA(4) + reserved(12) + OUI(24) + VSID(24). OUI = bytes 2..4.
		r.NAADesc = "IEEE 48-bit (NAA 1)"
		r.OUIBare, r.VSID = clean[4:10], clean[10:16]
	case 2:
		// NAA(4) + vendor-specific(12) + OUI(24) + VSID(24). OUI = bytes 2..4.
		r.NAADesc = "IEEE Extended (NAA 2)"
		r.OUIBare, r.VSID = clean[4:10], clean[10:16]
		r.Notes = append(r.Notes, "NAA 2 carries a 12-bit vendor-specific field before the OUI — often used for NPIV / virtual ports.")
	case 5:
		// NAA(4) + OUI(24) + VSID(36). OUI = the 6 nibbles after the NAA nibble.
		r.NAADesc = "IEEE Registered (NAA 5)"
		r.OUIBare, r.VSID = clean[1:7], clean[7:16]
	case 6:
		r.NAADesc = "IEEE Registered Extended (NAA 6, 128-bit)"
		r.OUIBare, r.VSID = clean[1:7], clean[7:]
	default:
		r.NAADesc = fmt.Sprintf("Unrecognized NAA (0x%x)", naa)
		r.Notes = append(r.Notes, "The leading nibble isn't a standard NAA type (1, 2, 5 or 6) — this may not be a Fibre Channel WWN.")
	}

	if r.OUIBare != "" {
		r.OUI = group(r.OUIBare, 2, ":")
		if r.OUIBare == "000000" {
			r.Notes = append(r.Notes, "The OUI is all-zero — a locally assigned or placeholder WWN, not vendor-registered.")
		}
	}
	// Node vs port isn't encoded in a standard, derivable way — it's assigned by
	// the vendor — so we don't guess it.
	r.Notes = append(r.Notes, "Whether this is a node (WWNN) or port (WWPN) name isn't encoded in a standard way; it's assigned by the vendor/HBA.")
	return r, nil
}

// group inserts sep every n characters (e.g. "2000...":2:":" → "20:00:...").
func group(s string, n int, sep string) string {
	var b strings.Builder
	for i := 0; i < len(s); i += n {
		if i > 0 {
			b.WriteString(sep)
		}
		end := i + n
		if end > len(s) {
			end = len(s)
		}
		b.WriteString(s[i:end])
	}
	return b.String()
}
