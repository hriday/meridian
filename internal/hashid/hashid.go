// Package hashid identifies the likely type of a password hash from its format
// (prefix, length, character set) and gives the matching hashcat mode. It's
// heuristic — many hashes share a shape — so it returns ranked candidates, not a
// single answer. Pure and offline.
package hashid

import "strings"

// Candidate is one possible hash type.
type Candidate struct {
	Name string `json:"name"`
	Mode int    `json:"mode"` // hashcat -m value; -1 when hashcat has no mode
	Note string `json:"note,omitempty"`
}

// Identify returns ranked candidate types for a hash string (best guess first).
func Identify(hash string) []Candidate {
	h := strings.TrimSpace(hash)
	if h == "" {
		return nil
	}
	if c := byPrefix(h); c != nil {
		return c
	}
	if looksLikeDump(h) {
		return []Candidate{
			{Name: "secretsdump/pwdump line (LM:NT)", Mode: 1000, Note: "crack the NT half with -m 1000; the NT hash is the 4th colon field"},
		}
	}
	if isHex(h) {
		if c := byHexLength(len(h)); c != nil {
			return c
		}
	}
	return []Candidate{{Name: "Unknown", Mode: -1, Note: "doesn't match a common format"}}
}

func byPrefix(h string) []Candidate {
	lower := strings.ToLower(h)
	switch {
	case strings.HasPrefix(h, "$2a$"), strings.HasPrefix(h, "$2b$"), strings.HasPrefix(h, "$2y$"), strings.HasPrefix(h, "$2x$"):
		return one("bcrypt (Blowfish)", 3200)
	case strings.HasPrefix(h, "$1$"):
		return one("md5crypt (Unix $1$)", 500)
	case strings.HasPrefix(h, "$5$"):
		return one("sha256crypt (Unix $5$)", 7400)
	case strings.HasPrefix(h, "$6$"):
		return one("sha512crypt (Unix $6$)", 1800)
	case strings.HasPrefix(h, "$apr1$"):
		return one("Apache apr1 md5", 1600)
	case strings.HasPrefix(h, "$P$"), strings.HasPrefix(h, "$H$"):
		return one("phpass (WordPress/Joomla)", 400)
	case strings.HasPrefix(h, "$S$"):
		return one("Drupal 7 (sha512)", 7900)
	case strings.HasPrefix(h, "{SSHA}"):
		return one("LDAP {SSHA} (salted SHA1)", 111)
	case strings.HasPrefix(h, "{SHA}"):
		return one("LDAP {SHA} (SHA1)", 101)
	case strings.HasPrefix(lower, "$argon2id$"), strings.HasPrefix(lower, "$argon2i$"), strings.HasPrefix(lower, "$argon2d$"):
		return []Candidate{{Name: "Argon2", Mode: -1, Note: "hashcat mode varies by variant"}}
	case strings.HasPrefix(h, "$8$"):
		return one("Cisco IOS type 8 (pbkdf2-sha256)", 9200)
	case strings.HasPrefix(h, "$9$"):
		return one("Cisco IOS type 9 (scrypt)", 9300)
	case strings.HasPrefix(lower, "pbkdf2_sha256$"):
		return one("Django PBKDF2-SHA256", 10000)
	case strings.HasPrefix(h, "$krb5tgs$"):
		return one("Kerberos 5 TGS-REP (Kerberoast)", 13100)
	case strings.HasPrefix(h, "$krb5asrep$"):
		return one("Kerberos 5 AS-REP (AS-REP roast)", 18200)
	case strings.HasPrefix(lower, "sha256$"):
		return one("Django SHA256", 124)
	case strings.HasPrefix(h, "$DCC2$"), strings.HasPrefix(lower, "$dcc2$"):
		return one("Domain Cached Credentials 2 (mscash2)", 2100)
	case strings.HasPrefix(h, "*") && len(h) == 41 && isHex(h[1:]):
		return one("MySQL 4.1+ (SHA1(SHA1))", 300)
	}
	return nil
}

func byHexLength(n int) []Candidate {
	switch n {
	case 16:
		return []Candidate{{Name: "MySQL 3.23 (old)", Mode: 200}, {Name: "half-LM / DES", Mode: -1}}
	case 32:
		return []Candidate{
			{Name: "NTLM", Mode: 1000, Note: "the Windows/AD hash — most likely from a secretsdump"},
			{Name: "MD5", Mode: 0},
			{Name: "MD4", Mode: 900},
			{Name: "LM", Mode: 3000, Note: "if it's an uppercase-only legacy hash"},
		}
	case 40:
		return []Candidate{{Name: "SHA-1", Mode: 100}, {Name: "RIPEMD-160", Mode: 6000}, {Name: "MySQL 4.1+ (no *)", Mode: 300}}
	case 56:
		return []Candidate{{Name: "SHA-224", Mode: 1300}}
	case 64:
		return []Candidate{{Name: "SHA-256", Mode: 1400}, {Name: "SHA3-256 / Keccak-256", Mode: 5000}, {Name: "SHA-512/256", Mode: -1}}
	case 96:
		return []Candidate{{Name: "SHA-384", Mode: 10800}}
	case 128:
		return []Candidate{{Name: "SHA-512", Mode: 1700}, {Name: "Whirlpool", Mode: 6100}, {Name: "SHA3-512", Mode: 17600}}
	}
	return nil
}

func one(name string, mode int) []Candidate { return []Candidate{{Name: name, Mode: mode}} }

func isHex(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// looksLikeDump spots a secretsdump/pwdump line: user:rid:LMhash:NThash:::
func looksLikeDump(s string) bool {
	parts := strings.Split(s, ":")
	if len(parts) < 4 {
		return false
	}
	return len(parts[2]) == 32 && isHex(parts[2]) && len(parts[3]) == 32 && isHex(parts[3])
}
