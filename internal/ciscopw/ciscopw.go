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

// Analyze scans a config for password/secret commands and classifies each. It
// is anchored on Cisco's command grammar rather than the bare word "password":
// a TYPED secret is any password-ish keyword (password/secret/passwd/key/
// key-string/md5) directly followed by a type digit (0,4,5,6,7,8,9) and a value
// — this catches reversible type-7 keys in OSPF/NTP/TACACS/key-chains too. A
// no-digit cleartext is recorded only inside a real password command (enable/
// username/password/passwd), with an ASA `… encrypted` trailer treated as a
// hash. Banner/description/remark text is skipped, so the word "password" in
// prose is never mistaken for a secret.
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
		if len(toks) == 0 {
			continue
		}
		switch toks[0] {
		case "banner", "description", "remark":
			continue // prose, not config
		}
		if strings.HasPrefix(toks[0], "!") {
			continue
		}

		// Typed secrets: <keyword> <type-digit> <value>, anywhere on the line.
		matched := false
		for i := 0; i+2 < len(toks); i++ {
			if isSecretKeyword(toks[i]) && isTypeDigit(toks[i+1]) {
				out = append(out, classify(ctxOf(toks, i, section), toks[i], toks[i+1], toks[i+2]))
				matched = true
				i += 2
			}
		}
		if matched {
			continue
		}

		// No-digit cleartext (or an ASA `… encrypted` hash) only inside a real
		// password command.
		if !isPasswordCmd(toks[0]) {
			continue
		}
		asaEncrypted := len(toks) >= 2 && toks[len(toks)-1] == "encrypted"
		for i := 0; i+1 < len(toks); i++ {
			kw := toks[i]
			if kw != "password" && kw != "secret" && kw != "passwd" {
				continue
			}
			val := toks[i+1]
			if isTypeDigit(val) {
				break // a truncated `password 7` — not a cleartext value
			}
			ctx := ctxOf(toks, i, section)
			if asaEncrypted {
				out = append(out, Secret{Context: ctx, Keyword: kw, Type: "encrypted", Encoded: val, Verdict: "weak-hash"})
			} else {
				out = append(out, classify(ctx, kw, "0", val))
			}
			break // one implicit secret per line
		}
	}
	return out
}

func ctxOf(toks []string, i int, section string) string {
	if ctx := strings.Join(toks[:i], " "); ctx != "" {
		return ctx
	}
	return section
}

func isSecretKeyword(s string) bool {
	switch s {
	case "password", "secret", "passwd", "key", "key-string", "md5":
		return true
	}
	return false
}

func isPasswordCmd(first string) bool {
	switch first {
	case "enable", "username", "password", "passwd":
		return true
	}
	return false
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
