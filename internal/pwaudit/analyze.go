package pwaudit

import (
	"regexp"
	"strings"
	"unicode"
)

// Result is one audited account: whether its password was recovered, and if so
// the plaintext with a strength rating and a detected pattern.
type Result struct {
	Account   Account `json:"account"`
	Cracked   bool    `json:"cracked"`
	Plaintext string  `json:"plaintext"`
	Strength  string  `json:"strength"` // weak | medium | strong
	Pattern   string  `json:"pattern"`
}

// ReuseGroup is a set of accounts that share one NT hash (the same password),
// or a single account using the well-known empty-password hash.
type ReuseGroup struct {
	NTHash    string   `json:"ntHash"`
	Plaintext string   `json:"plaintext"` // "" when the shared password is uncracked
	Accounts  []string `json:"accounts"`
}

// EmptyNTHash is the NTLM hash of an empty string — accounts with it have no password.
const EmptyNTHash = "31d6cfe0d16ae931b73c59d7e0c089c0"

var (
	seasonYear = regexp.MustCompile(`(?i)(spring|summer|fall|autumn|winter|jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)[^0-9]*\d{2,4}`)
	allDigits  = regexp.MustCompile(`^\d+$`)
	wordDigit  = regexp.MustCompile(`^[A-Za-z]+\d+[!@#$%^&*]?$`)
	walks      = []string{"qwerty", "qwertz", "azerty", "asdf", "zxcv", "qwer", "1qaz", "qazwsx"}
)

// Classify returns a plaintext password's strength (weak/medium/strong) and a
// human-readable pattern label ("none" when nothing obvious matches).
func Classify(plaintext string) (strength, pattern string) {
	low := strings.ToLower(plaintext)
	switch {
	case seasonYear.MatchString(plaintext):
		pattern = "season+year"
	case containsAny(low, walks):
		pattern = "keyboard walk"
	case allDigits.MatchString(plaintext):
		pattern = "all digits"
	case wordDigit.MatchString(plaintext):
		pattern = "word+digit"
	default:
		pattern = "none"
	}
	return rate(plaintext, pattern), pattern
}

func rate(p, pattern string) string {
	if pattern != "none" {
		return "weak" // a recognised weak pattern is weak regardless of length
	}
	switch classes := charClasses(p); {
	case len(p) < 8:
		return "weak"
	case len(p) >= 12 && classes >= 3:
		return "strong"
	default:
		return "medium"
	}
}

func charClasses(p string) int {
	var lower, upper, digit, other bool
	for _, r := range p {
		switch {
		case unicode.IsLower(r):
			lower = true
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsDigit(r):
			digit = true
		default:
			other = true
		}
	}
	n := 0
	for _, b := range []bool{lower, upper, digit, other} {
		if b {
			n++
		}
	}
	return n
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// DetectReuse groups accounts that share an NT hash (same password, cracked or
// not) and always surfaces accounts using the empty-password hash. Order is the
// first-seen hash order, for determinism.
func DetectReuse(results []Result) []ReuseGroup {
	names := map[string][]string{}
	plain := map[string]string{}
	var order []string
	for _, r := range results {
		h := r.Account.NTHash
		if _, seen := names[h]; !seen {
			order = append(order, h)
		}
		names[h] = append(names[h], r.Account.Name)
		if r.Cracked {
			plain[h] = r.Plaintext
		}
	}
	var groups []ReuseGroup
	for _, h := range order {
		isEmpty := h == EmptyNTHash
		if len(names[h]) < 2 && !isEmpty {
			continue
		}
		pt := plain[h]
		if isEmpty {
			pt = "(empty password)"
		}
		groups = append(groups, ReuseGroup{NTHash: h, Plaintext: pt, Accounts: names[h]})
	}
	return groups
}
