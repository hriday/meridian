// Package sshaudit inspects an SSH server's offered algorithms and host keys
// without authenticating, and grades them — with a specific focus on
// post-quantum readiness.
//
// The threat it speaks to is "harvest now, decrypt later": an adversary records
// an encrypted session today and decrypts it once a cryptographically-relevant
// quantum computer exists. The defense is a post-quantum *key exchange* (e.g.
// sntrup761x25519-sha512 or mlkem768x25519-sha256), which protects the session
// key even against a future quantum attacker. Authentication (host-key
// signatures) is still classical everywhere — PQ signature algorithms are
// standardized but not yet fielded in SSH — so this tool grades the key
// exchange and reports signatures as informational, not a failure.
//
// How it works: an SSH server sends its SSH_MSG_KEXINIT — the full list of
// algorithms it will accept — in cleartext, before any key exchange or
// authentication. Reading that one packet yields the complete algorithm
// inventory. Host-key fingerprints come from a separate handshake that is
// aborted the moment the key is received (see probe.go), so no credentials are
// ever sent.
package sshaudit

import (
	"encoding/binary"
	"fmt"
	"strings"
)

const msgKexInit = 20 // SSH_MSG_KEXINIT (RFC 4253 §7.1)

// KexInit holds the algorithm name-lists a server advertises in its KEXINIT.
type KexInit struct {
	Kex          []string `json:"kex"`
	HostKeyAlgos []string `json:"hostKeyAlgos"`
	CiphersC2S   []string `json:"ciphersC2S"`
	CiphersS2C   []string `json:"ciphersS2C"`
	MACsC2S      []string `json:"macsC2S"`
	MACsS2C      []string `json:"macsS2C"`
	CompC2S      []string `json:"compC2S"`
	CompS2C      []string `json:"compS2C"`
}

// HostKey is one of the server's public host keys.
type HostKey struct {
	Type        string `json:"type"`        // ssh-ed25519, rsa-sha2-512, ecdsa-sha2-nistp256, …
	Bits        int    `json:"bits"`        // key size where meaningful (0 if unknown)
	Fingerprint string `json:"fingerprint"` // SHA256:… (OpenSSH format)
}

// PQStatus describes post-quantum key-exchange support.
type PQStatus struct {
	Supported  bool     `json:"supported"`  // a PQ hybrid kex is offered
	Preferred  bool     `json:"preferred"`  // a PQ hybrid kex is the server's first choice
	Algorithms []string `json:"algorithms"` // the PQ kex algorithms offered
	Verdict    string   `json:"verdict"`
}

// Report is the full audit result.
type Report struct {
	Host         string    `json:"host"`
	ServerID     string    `json:"serverId"` // raw banner, e.g. SSH-2.0-OpenSSH_9.6
	Software     string    `json:"software"` // parsed, e.g. OpenSSH_9.6
	Kex          []string  `json:"kex"`
	HostKeyAlgos []string  `json:"hostKeyAlgos"`
	Ciphers      []string  `json:"ciphers"`
	MACs         []string  `json:"macs"`
	HostKeys     []HostKey `json:"hostKeys"`
	PQ           PQStatus  `json:"pq"`
	Grade        string    `json:"grade"`
	Findings     []string  `json:"findings"`
	Remediation  []string  `json:"remediation,omitempty"` // concrete sshd_config fixes to raise the grade
	Summary      string    `json:"summary"`
}

// ParseKexInit parses the payload of an SSH_MSG_KEXINIT packet (the message
// byte, 16-byte cookie, then ten name-lists).
func ParseKexInit(payload []byte) (KexInit, error) {
	if len(payload) < 17 || payload[0] != msgKexInit {
		return KexInit{}, fmt.Errorf("not an SSH KEXINIT message")
	}
	p := payload[17:] // skip message type + cookie
	lists := make([][]string, 0, 10)
	for i := 0; i < 10; i++ {
		if len(p) < 4 {
			return KexInit{}, fmt.Errorf("truncated KEXINIT name-list %d", i)
		}
		n := binary.BigEndian.Uint32(p[:4])
		p = p[4:]
		if uint32(len(p)) < n {
			return KexInit{}, fmt.Errorf("truncated KEXINIT name-list %d (want %d bytes)", i, n)
		}
		lists = append(lists, splitNameList(string(p[:n])))
		p = p[n:]
	}
	return KexInit{
		Kex:          lists[0],
		HostKeyAlgos: lists[1],
		CiphersC2S:   lists[2],
		CiphersS2C:   lists[3],
		MACsC2S:      lists[4],
		MACsS2C:      lists[5],
		CompC2S:      lists[6],
		CompS2C:      lists[7],
	}, nil
}

func splitNameList(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

// pqAlgo reports whether a kex algorithm name is a post-quantum hybrid.
func pqAlgo(name string) bool {
	n := strings.ToLower(name)
	return strings.Contains(n, "sntrup761") ||
		strings.Contains(n, "mlkem") ||
		strings.Contains(n, "kyber")
}

func assessPQ(kex []string) PQStatus {
	var pq PQStatus
	for i, k := range kex {
		if pqAlgo(k) {
			pq.Supported = true
			pq.Algorithms = append(pq.Algorithms, k)
			if i == 0 {
				pq.Preferred = true
			}
		}
	}
	switch {
	case pq.Preferred:
		pq.Verdict = "Post-quantum key exchange is the server's default (" + pq.Algorithms[0] + ") — recorded sessions are protected against harvest-now-decrypt-later."
	case pq.Supported:
		pq.Verdict = "Post-quantum key exchange is offered (" + strings.Join(pq.Algorithms, ", ") + ") but isn't the server's first preference; a modern client will still negotiate it."
	default:
		pq.Verdict = "No post-quantum key exchange — a recorded session could be decrypted by a future quantum computer."
	}
	return pq
}

// weak-algorithm predicates. "Critical" entries fail the grade outright;
// "moderate" entries cap it.
func badCipher(c string) bool {
	c = strings.ToLower(c)
	return c == "none" || strings.Contains(c, "3des") || strings.Contains(c, "arcfour") ||
		strings.Contains(c, "blowfish") || strings.Contains(c, "cast128") ||
		strings.HasSuffix(c, "-cbc")
}

func badMAC(m string) bool {
	m = strings.ToLower(m)
	return strings.Contains(m, "hmac-md5") || strings.Contains(m, "none")
}

func weakMAC(m string) bool {
	m = strings.ToLower(m)
	// SHA-1 and truncated MACs are weak but not catastrophic; ETM SHA-2 is fine.
	return strings.HasPrefix(m, "hmac-sha1") || strings.Contains(m, "umac-64") ||
		strings.Contains(m, "-96")
}

func badKex(k string) bool {
	k = strings.ToLower(k)
	return strings.Contains(k, "group1-sha1") || k == "diffie-hellman-group1-sha1" ||
		strings.Contains(k, "gss-group1")
}

func weakKex(k string) bool {
	k = strings.ToLower(k)
	return strings.HasSuffix(k, "-sha1") || strings.Contains(k, "group-exchange-sha1")
}

func dedupeLower(a, b []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range append(append([]string{}, a...), b...) {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// grade returns an A–F grade, the findings behind it, and the PQ status.
func grade(k KexInit, keys []HostKey) (string, []string, PQStatus) {
	pq := assessPQ(k.Kex)
	var findings []string
	critical, moderate := false, false

	for _, c := range dedupeLower(k.CiphersS2C, k.CiphersC2S) {
		if badCipher(c) {
			critical = true
			findings = append(findings, "Weak cipher offered: "+c+" — disable it.")
		}
	}
	for _, m := range dedupeLower(k.MACsS2C, k.MACsC2S) {
		switch {
		case badMAC(m):
			critical = true
			findings = append(findings, "Weak MAC offered: "+m+" — disable it.")
		case weakMAC(m):
			moderate = true
			findings = append(findings, "Legacy MAC offered: "+m+" — prefer an ETM SHA-2 MAC.")
		}
	}
	for _, x := range k.Kex {
		switch {
		case badKex(x):
			critical = true
			findings = append(findings, "Weak key exchange offered: "+x+" — disable it.")
		case weakKex(x):
			moderate = true
			findings = append(findings, "SHA-1 key exchange offered: "+x+" — prefer a SHA-2 or post-quantum method.")
		}
	}
	hasRSASHA2 := false
	for _, h := range k.HostKeyAlgos {
		if strings.HasPrefix(strings.ToLower(h), "rsa-sha2-") {
			hasRSASHA2 = true
		}
	}
	for _, h := range k.HostKeyAlgos {
		hl := strings.ToLower(h)
		switch {
		case strings.HasPrefix(hl, "ssh-dss"):
			critical = true
			findings = append(findings, "DSA host key (ssh-dss) offered — obsolete and weak; remove it.")
		case hl == "ssh-rsa":
			if hasRSASHA2 {
				// Legacy compat only; the server will negotiate rsa-sha2-* with
				// any modern client. Note it, but don't cap the grade.
				findings = append(findings, "Legacy ssh-rsa (SHA-1) host-key signature offered for backward compatibility — harmless since rsa-sha2-256/512 is also offered, but you can disable it.")
			} else {
				moderate = true
				findings = append(findings, "SHA-1 RSA host-key signature (ssh-rsa) offered with no rsa-sha2 alternative — prefer rsa-sha2-256/512 or ed25519.")
			}
		}
	}

	if !pq.Supported {
		findings = append(findings, "No post-quantum key exchange — recorded sessions are exposed to harvest-now-decrypt-later. Enable a post-quantum method (sntrup761x25519-sha512 or mlkem768x25519-sha256, OpenSSH 9+).")
	}

	var g string
	switch {
	case critical:
		g = "F"
	case moderate:
		g = "C"
	case !pq.Supported:
		g = "B"
	default:
		g = "A"
	}
	return g, findings, pq
}

// remediate returns concrete sshd_config directives that would raise the grade,
// derived from the weaknesses actually present (so it only suggests what this
// server needs). Empty when there's nothing to fix.
func remediate(k KexInit, pq PQStatus) []string {
	var out []string

	for _, c := range dedupeLower(k.CiphersS2C, k.CiphersC2S) {
		if badCipher(c) {
			out = append(out, "Restrict ciphers to AEAD only:\n    Ciphers chacha20-poly1305@openssh.com,aes256-gcm@openssh.com,aes128-gcm@openssh.com")
			break
		}
	}
	for _, m := range dedupeLower(k.MACsS2C, k.MACsC2S) {
		if badMAC(m) || weakMAC(m) {
			out = append(out, "Restrict MACs to ETM SHA-2 (drops hmac-sha1 and non-ETM MACs):\n    MACs hmac-sha2-256-etm@openssh.com,hmac-sha2-512-etm@openssh.com,umac-128-etm@openssh.com")
			break
		}
	}
	weakKexPresent := false
	for _, x := range k.Kex {
		if badKex(x) || weakKex(x) {
			weakKexPresent = true
			break
		}
	}
	if weakKexPresent || !pq.Preferred {
		out = append(out, "Prefer a post-quantum key exchange (OpenSSH 9+), dropping SHA-1 KEX:\n    KexAlgorithms sntrup761x25519-sha512@openssh.com,mlkem768x25519-sha256,curve25519-sha256,curve25519-sha256@libssh.org")
	}
	hasRSASHA2, dsa, sshRSAOnly := false, false, false
	for _, h := range k.HostKeyAlgos {
		hl := strings.ToLower(h)
		if strings.HasPrefix(hl, "rsa-sha2-") {
			hasRSASHA2 = true
		}
		if strings.HasPrefix(hl, "ssh-dss") {
			dsa = true
		}
	}
	for _, h := range k.HostKeyAlgos {
		if strings.EqualFold(h, "ssh-rsa") && !hasRSASHA2 {
			sshRSAOnly = true
		}
	}
	if dsa || sshRSAOnly {
		out = append(out, "Use modern host-key signatures (and remove any DSA host key):\n    HostKeyAlgorithms rsa-sha2-512,rsa-sha2-256,ssh-ed25519,ecdsa-sha2-nistp256")
	}
	if len(out) > 0 {
		out = append([]string{"Edit /etc/ssh/sshd_config (or a drop-in under /etc/ssh/sshd_config.d/) and set:"}, out...)
		out = append(out, "Then validate and reload: sudo sshd -t && sudo systemctl reload ssh")
	}
	return out
}

// softwareFromBanner extracts the software string from an SSH identification
// banner ("SSH-2.0-OpenSSH_9.6" → "OpenSSH_9.6").
func softwareFromBanner(banner string) string {
	b := strings.TrimSpace(banner)
	if strings.HasPrefix(b, "SSH-") {
		parts := strings.SplitN(b, "-", 3)
		if len(parts) == 3 {
			return parts[2]
		}
	}
	return b
}

// Build assembles a Report from a server banner, its KEXINIT and host keys.
func Build(host, serverID string, k KexInit, keys []HostKey) Report {
	g, findings, pq := grade(k, keys)
	r := Report{
		Host:         host,
		ServerID:     strings.TrimSpace(serverID),
		Software:     softwareFromBanner(serverID),
		Kex:          k.Kex,
		HostKeyAlgos: k.HostKeyAlgos,
		Ciphers:      k.CiphersS2C,
		MACs:         k.MACsS2C,
		HostKeys:     keys,
		PQ:           pq,
		Grade:        g,
		Findings:     findings,
		Remediation:  remediate(k, pq),
	}
	pqMark := "✗"
	if pq.Preferred {
		pqMark = "✓ (default)"
	} else if pq.Supported {
		pqMark = "✓ (offered)"
	}
	r.Summary = fmt.Sprintf("grade %s · %s · PQ key exchange %s · %d host key(s)",
		g, r.Software, pqMark, len(keys))
	return r
}
