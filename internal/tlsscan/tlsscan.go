// Package tlsscan probes which TLS protocol versions and cipher suites a server
// accepts, and grades the result — flagging legacy protocols (TLS 1.0/1.1) and
// weak ciphers (RC4, 3DES, CBC) still negotiable. It's the depth companion to
// the TLS-certificate and cert-expiry tools, which look at the certificate; this
// looks at the handshake.
//
// Scope note: Go's crypto/tls can't test SSLv3 or export/DES ciphers (removed
// from the standard library), so those are reported as "not tested" rather than
// a false all-clear. What it does test — TLS 1.0–1.3 and every cipher Go still
// implements, including the ones in tls.InsecureCipherSuites — covers the
// findings that matter in practice.
package tlsscan

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strings"
)

// Handshaker performs one TLS handshake with the given config and returns the
// resulting connection state. Injected so tests run offline.
type Handshaker func(ctx context.Context, hostport string, cfg *tls.Config) (tls.ConnectionState, error)

type Protocol struct {
	Name      string `json:"name"`
	Version   uint16 `json:"version"`
	Supported bool   `json:"supported"`
	Legacy    bool   `json:"legacy"` // TLS 1.0 / 1.1
}

type Cipher struct {
	ID      uint16 `json:"id"`
	Name    string `json:"name"`
	Secure  bool   `json:"secure"`
	Version string `json:"version"` // "TLS 1.2" | "TLS 1.3"
}

// PQStatus describes whether the server negotiates a post-quantum hybrid key
// exchange — the defense against "harvest now, decrypt later".
type PQStatus struct {
	Supported bool   `json:"supported"`
	Group     string `json:"group"` // e.g. X25519MLKEM768
	Verdict   string `json:"verdict"`
}

type Report struct {
	Host          string     `json:"host"`
	Protocols     []Protocol `json:"protocols"`
	Ciphers       []Cipher   `json:"ciphers"`
	PQ            PQStatus   `json:"pq"`
	CertSignature string     `json:"certSignature"` // leaf cert signature algorithm (classical today)
	Grade         string     `json:"grade"`         // A–F
	Findings      []string   `json:"findings"`
	Summary       string     `json:"summary"`
}

var probeVersions = []Protocol{
	{Name: "TLS 1.0", Version: tls.VersionTLS10, Legacy: true},
	{Name: "TLS 1.1", Version: tls.VersionTLS11, Legacy: true},
	{Name: "TLS 1.2", Version: tls.VersionTLS12},
	{Name: "TLS 1.3", Version: tls.VersionTLS13},
}

// Scan probes host for supported protocols and (for TLS 1.2) cipher suites.
func Scan(ctx context.Context, hs Handshaker, host string) (Report, error) {
	hostport := normalizeHostPort(host)
	sni, _, _ := net.SplitHostPort(hostport)
	r := Report{Host: hostport}

	anySupported := false
	for _, p := range probeVersions {
		cfg := &tls.Config{
			ServerName:         sni,
			InsecureSkipVerify: true, //nolint:gosec // probing the handshake, not trusting the cert
			MinVersion:         p.Version,
			MaxVersion:         p.Version,
		}
		if p.Version <= tls.VersionTLS12 {
			cfg.CipherSuites = allTLS12SuiteIDs()
		}
		_, err := hs(ctx, hostport, cfg)
		p.Supported = err == nil
		if p.Supported {
			anySupported = true
		}
		r.Protocols = append(r.Protocols, p)
	}
	if !anySupported {
		return Report{}, fmt.Errorf("no TLS handshake succeeded — not a TLS service, or unreachable")
	}

	// Enumerate TLS 1.2 cipher suites one at a time.
	if supported(r.Protocols, tls.VersionTLS12) {
		for _, s := range allTLS12Suites() {
			cfg := &tls.Config{
				ServerName:         sni,
				InsecureSkipVerify: true, //nolint:gosec // probing the handshake
				MinVersion:         tls.VersionTLS12,
				MaxVersion:         tls.VersionTLS12,
				CipherSuites:       []uint16{s.id},
			}
			if _, err := hs(ctx, hostport, cfg); err == nil {
				r.Ciphers = append(r.Ciphers, Cipher{
					ID: s.id, Name: tls.CipherSuiteName(s.id), Secure: s.secure, Version: "TLS 1.2",
				})
			}
		}
	}
	// TLS 1.3 negotiates its own (always-strong) suite; record what it picks,
	// and the leaf certificate's signature algorithm (classical today).
	if supported(r.Protocols, tls.VersionTLS13) {
		cfg := &tls.Config{ServerName: sni, InsecureSkipVerify: true, MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13} //nolint:gosec
		if st, err := hs(ctx, hostport, cfg); err == nil {
			if st.CipherSuite != 0 {
				r.Ciphers = append(r.Ciphers, Cipher{
					ID: st.CipherSuite, Name: tls.CipherSuiteName(st.CipherSuite), Secure: true, Version: "TLS 1.3",
				})
			}
			if len(st.PeerCertificates) > 0 {
				r.CertSignature = st.PeerCertificates[0].SignatureAlgorithm.String()
			}
		}
		r.PQ = probePQ(ctx, hs, sni, hostport)
	}

	r.Grade, r.Findings = grade(r.Protocols, r.Ciphers)
	// PQ readiness is reported and (when absent) noted, but it does not fail the
	// letter grade: unlike SSH, post-quantum TLS is still early in deployment,
	// so a modern TLS 1.3 + strong-cipher server is legitimately an A today.
	if supported(r.Protocols, tls.VersionTLS13) && !r.PQ.Supported {
		r.Findings = append(r.Findings, "No post-quantum key exchange — recorded traffic is exposed to harvest-now-decrypt-later. TLS PQ (X25519MLKEM768) is still rolling out, so this is common today; enable it when your stack supports it.")
	}
	r.Summary = summarize(r)
	return r, nil
}

// probePQ actively tests whether the server will negotiate a post-quantum
// hybrid key-exchange group, by offering only that group: a successful
// handshake proves support, a failure proves its absence.
func probePQ(ctx context.Context, hs Handshaker, sni, hostport string) PQStatus {
	cfg := &tls.Config{
		ServerName:         sni,
		InsecureSkipVerify: true, //nolint:gosec // probing the handshake, not trusting the cert
		MinVersion:         tls.VersionTLS13,
		MaxVersion:         tls.VersionTLS13,
		CurvePreferences:   []tls.CurveID{tls.X25519MLKEM768},
	}
	if _, err := hs(ctx, hostport, cfg); err == nil {
		return PQStatus{
			Supported: true,
			Group:     "X25519MLKEM768",
			Verdict:   "The server negotiates a post-quantum hybrid key exchange (X25519MLKEM768) — recorded traffic is protected against harvest-now-decrypt-later.",
		}
	}
	return PQStatus{
		Supported: false,
		Verdict:   "No post-quantum key exchange — a recorded session could be decrypted by a future quantum computer. (Still uncommon in TLS today.)",
	}
}

// brokenCipher reports whether a suite is cryptographically broken (as opposed
// to merely legacy): RC4, 3DES/DES, NULL, EXPORT, anonymous, or MD5-based.
// These fail the grade outright; CBC-mode and RSA-key-exchange suites are
// "legacy" (discouraged, no forward secrecy) but not broken.
func brokenCipher(name string) bool {
	n := strings.ToUpper(name)
	for _, bad := range []string{"RC4", "3DES", "_DES_", "NULL", "EXPORT", "_ANON", "MD5"} {
		if strings.Contains(n, bad) {
			return true
		}
	}
	return false
}

// grade returns an A–F grade and the findings behind it. F is reserved for
// genuinely broken configurations (a broken cipher negotiable, or no modern
// protocol at all). A deprecated protocol (TLS 1.0/1.1) caps at C; a legacy
// but unbroken cipher (CBC / no forward secrecy) caps at B — matching the way
// SSL Labs penalizes rather than fails these.
func grade(protos []Protocol, ciphers []Cipher) (string, []string) {
	var findings []string
	legacyProto := false
	for _, p := range protos {
		if p.Supported && p.Legacy {
			legacyProto = true
			findings = append(findings, p.Name+" is enabled — a deprecated protocol; disable it (modern clients use TLS 1.2+).")
		}
	}
	broken, legacyCipher := false, false
	for _, c := range ciphers {
		if c.Secure {
			continue
		}
		if brokenCipher(c.Name) {
			broken = true
			findings = append(findings, "Broken cipher negotiable: "+c.Name+" — remove it immediately.")
		} else {
			legacyCipher = true
			findings = append(findings, "Legacy cipher negotiable: "+c.Name+" (CBC mode or no forward secrecy) — prefer AEAD suites with ECDHE.")
		}
	}
	tls13 := supported(protos, tls.VersionTLS13)
	tls12 := supported(protos, tls.VersionTLS12)

	switch {
	case broken:
		return "F", findings
	case !tls12 && !tls13:
		findings = append(findings, "No TLS 1.2 or 1.3 — modern clients may refuse to connect.")
		return "F", findings
	case legacyProto:
		return "C", findings
	case legacyCipher:
		return "B", findings
	case tls13 && tls12:
		return "A", findings
	case tls12 && !tls13:
		findings = append(findings, "TLS 1.3 not offered — enable it for the strongest handshake.")
		return "B", findings
	default:
		return "A", findings
	}
}

func summarize(r Report) string {
	var vers []string
	for _, p := range r.Protocols {
		if p.Supported {
			vers = append(vers, p.Name)
		}
	}
	pqMark := "PQ ✗"
	if r.PQ.Supported {
		pqMark = "PQ ✓"
	}
	return fmt.Sprintf("grade %s · %s · %s · %d cipher suite(s)", r.Grade, strings.Join(vers, ", "), pqMark, len(r.Ciphers))
}

func supported(protos []Protocol, v uint16) bool {
	for _, p := range protos {
		if p.Version == v {
			return p.Supported
		}
	}
	return false
}

type suite struct {
	id     uint16
	secure bool
}

// allTLS12Suites returns every cipher suite Go implements that applies to TLS
// 1.2, each tagged secure or insecure per the standard library's own split.
func allTLS12Suites() []suite {
	var out []suite
	add := func(cs []*tls.CipherSuite, secure bool) {
		for _, c := range cs {
			for _, v := range c.SupportedVersions {
				if v == tls.VersionTLS12 {
					out = append(out, suite{id: c.ID, secure: secure})
					break
				}
			}
		}
	}
	add(tls.CipherSuites(), true)
	add(tls.InsecureCipherSuites(), false)
	sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	return out
}

func allTLS12SuiteIDs() []uint16 {
	s := allTLS12Suites()
	ids := make([]uint16, len(s))
	for i, x := range s {
		ids[i] = x.id
	}
	return ids
}

// normalizeHostPort strips a scheme/path and defaults the port to 443.
func normalizeHostPort(s string) string {
	s = strings.TrimSpace(s)
	if strings.Contains(s, "://") {
		if u, err := url.Parse(s); err == nil && u.Host != "" {
			s = u.Host
		}
	} else if i := strings.IndexByte(s, '/'); i >= 0 {
		s = s[:i]
	}
	if _, _, err := net.SplitHostPort(s); err == nil {
		return s
	}
	return net.JoinHostPort(s, "443")
}
