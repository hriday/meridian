// Package pwaudit is a defensive Active Directory password-audit engine: it
// parses NTLM hashes the operator already administers, drives an external
// cracker (hashcat/John) through an injected seam, and reports weak passwords —
// their plaintext, pattern, strength, and reuse. Recovered plaintext stays in
// memory and, only on explicit request, an encrypted store; it is never logged
// or transmitted.
package pwaudit

import (
	"bufio"
	"io"
	"regexp"
	"strings"
)

// Account is one directory account and its NT hash.
type Account struct {
	Name    string `json:"name"`
	NTHash  string `json:"ntHash"`
	RID     string `json:"rid"`
	Machine bool   `json:"machine"`
}

var hex32 = regexp.MustCompile(`^[0-9a-fA-F]{32}$`)

// ParseDump reads a pwdump/secretsdump file (DOMAIN\user:rid:lmhash:nthash:::)
// and returns the accounts with a valid 32-hex NT hash (field 4). Machine
// accounts (name ending in "$") are kept and flagged. Blank and comment lines
// are skipped.
func ParseDump(r io.Reader) ([]Account, error) {
	var out []Account
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, ":")
		if len(f) < 4 || !hex32.MatchString(f[3]) {
			continue
		}
		name := f[0]
		if i := strings.LastIndex(name, `\`); i >= 0 {
			name = name[i+1:]
		}
		out = append(out, Account{
			Name:    name,
			NTHash:  strings.ToLower(f[3]),
			RID:     f[1],
			Machine: strings.HasSuffix(name, "$"),
		})
	}
	return out, sc.Err()
}

// ParseRaw reads pasted NT hashes: each line is a bare 32-hex hash or
// "name:hash". Lines without a valid 32-hex hash are skipped.
func ParseRaw(text string) ([]Account, error) {
	var out []Account
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, hash := "", line
		if i := strings.LastIndex(line, ":"); i >= 0 {
			name, hash = line[:i], line[i+1:]
		}
		hash = strings.TrimSpace(hash)
		if !hex32.MatchString(hash) {
			continue
		}
		out = append(out, Account{
			Name:    strings.TrimSpace(name),
			NTHash:  strings.ToLower(hash),
			Machine: strings.HasSuffix(strings.TrimSpace(name), "$"),
		})
	}
	return out, nil
}
