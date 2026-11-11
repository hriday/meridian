// Package ciscopw decodes and classifies the password/secret strings in a Cisco
// (IOS/ASA) configuration. Type 7 is a reversible Vigenère cipher and is decoded
// to plaintext; type 0 is cleartext; the rest are one-way hashes (or key-wrapped)
// and are classified by strength rather than decoded. Pure and offline — a
// pasted config never leaves the machine.
package ciscopw

import (
	"fmt"
	"strconv"
	"strings"
)

// xlat is Cisco's fixed type-7 translation key.
var xlat = []byte("dsfd;kfoA,.iyewrkldJKDHSUBsgvca69834ncxv9873254k;fg87")

// DecodeType7 decodes a Cisco type-7 password to plaintext. The first two digits
// are the key offset; the rest are hex bytes XORed against the rotating key.
func DecodeType7(enc string) (string, error) {
	enc = strings.TrimSpace(enc)
	if len(enc) < 2 || len(enc)%2 != 0 {
		return "", fmt.Errorf("not a valid type-7 string")
	}
	seed, err := strconv.Atoi(enc[:2])
	if err != nil || seed < 0 || seed >= len(xlat) {
		return "", fmt.Errorf("not a valid type-7 string")
	}
	out := make([]byte, 0, (len(enc)-2)/2)
	for i := 2; i < len(enc); i += 2 {
		b, err := strconv.ParseUint(enc[i:i+2], 16, 8)
		if err != nil {
			return "", fmt.Errorf("not a valid type-7 string")
		}
		out = append(out, byte(b)^xlat[(seed+(i-2)/2)%len(xlat)])
	}
	return string(out), nil
}

// Secret is one password/secret found in a config.
type Secret struct {
	Context   string `json:"context"`   // the command it sits under, e.g. "username bob"
	Keyword   string `json:"keyword"`   // "password" | "secret"
	Type      string `json:"type"`      // Cisco type: 0,4,5,6,7,8,9
	Encoded   string `json:"encoded"`   // the stored value (empty for cleartext)
	Plaintext string `json:"plaintext"` // recovered plaintext (type 7 / 0)
	Decoded   bool   `json:"decoded"`   // plaintext is known
	Verdict   string `json:"verdict"`   // cleartext|reversible|reversible-with-key|weak-hash|deprecated-hash|strong-hash
}

// Analyze scans a config for password/secret commands and classifies each.
func Analyze(configText string) []Secret {
	var out []Secret
	section := ""
	for _, raw := range strings.Split(configText, "\n") {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		if raw[0] != ' ' && raw[0] != '\t' {
			section = strings.TrimSpace(raw)
		}
		toks := strings.Fields(raw)
		for i := 0; i < len(toks); i++ {
			if toks[i] != "password" && toks[i] != "secret" {
				continue
			}
			if i+1 >= len(toks) {
				continue
			}
			var typ, val string
			consumed := i + 1
			if isTypeDigit(toks[i+1]) && i+2 < len(toks) {
				typ, val, consumed = toks[i+1], toks[i+2], i+2
			} else {
				typ, val = "0", toks[i+1]
			}
			ctx := strings.Join(toks[:i], " ")
			if ctx == "" {
				ctx = section
			}
			out = append(out, classify(ctx, toks[i], typ, val))
			i = consumed
		}
	}
	return out
}

func classify(ctx, kw, typ, val string) Secret {
	s := Secret{Context: ctx, Keyword: kw, Type: typ, Encoded: val}
	switch typ {
	case "7":
		s.Verdict = "reversible"
		if p, err := DecodeType7(val); err == nil {
			s.Plaintext, s.Decoded = p, true
		}
	case "0":
		s.Plaintext, s.Decoded, s.Verdict, s.Encoded = val, true, "cleartext", ""
	case "6":
		s.Verdict = "reversible-with-key"
	case "5":
		s.Verdict = "weak-hash"
	case "4":
		s.Verdict = "deprecated-hash"
	case "8", "9":
		s.Verdict = "strong-hash"
	default:
		s.Verdict = "unknown"
	}
	return s
}

func isTypeDigit(s string) bool {
	switch s {
	case "0", "4", "5", "6", "7", "8", "9":
		return true
	}
	return false
}
