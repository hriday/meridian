package sshaudit

import (
	"bufio"
	"encoding/binary"
	"net"
	"strings"
	"testing"
)

// buildPacket wraps a payload in an unencrypted SSH binary packet with valid
// padding (RFC 4253 §6): total length a multiple of 8, padding ≥ 4.
func buildPacket(payload []byte) []byte {
	base := 4 + 1 + len(payload)
	padLen := 8 - (base % 8)
	if padLen < 4 {
		padLen += 8
	}
	pktLen := 1 + len(payload) + padLen
	out := make([]byte, 4)
	binary.BigEndian.PutUint32(out, uint32(pktLen))
	out = append(out, byte(padLen))
	out = append(out, payload...)
	out = append(out, make([]byte, padLen)...)
	return out
}

func TestReadKexInitConnOverPipe(t *testing.T) {
	client, server := net.Pipe()
	payload := buildKexInit(
		"sntrup761x25519-sha512@openssh.com,curve25519-sha256",
		"ssh-ed25519",
		"chacha20-poly1305@openssh.com",
		"hmac-sha2-256-etm@openssh.com",
	)
	go func() {
		defer server.Close()
		_, _ = server.Write([]byte("Welcome notice\r\nSSH-2.0-TestServer\r\n"))
		_, _ = bufio.NewReader(server).ReadString('\n') // consume client identification
		_, _ = server.Write(buildPacket(payload))
	}()

	serverID, k, err := readKexInitConn(client)
	_ = client.Close()
	if err != nil {
		t.Fatalf("readKexInitConn: %v", err)
	}
	if serverID != "SSH-2.0-TestServer" {
		t.Errorf("serverID = %q", serverID)
	}
	if len(k.Kex) != 2 || k.Kex[0] != "sntrup761x25519-sha512@openssh.com" {
		t.Errorf("kex = %v", k.Kex)
	}
	if !assessPQ(k.Kex).Preferred {
		t.Error("expected PQ preferred over the pipe")
	}
}

// nameList encodes an SSH name-list: uint32 length + comma-joined string.
func nameList(s string) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(len(s)))
	return append(b, s...)
}

// buildKexInit assembles a valid SSH_MSG_KEXINIT payload for tests.
func buildKexInit(kex, hostkey, cipher, mac string) []byte {
	var p []byte
	p = append(p, msgKexInit)
	p = append(p, make([]byte, 16)...) // cookie
	p = append(p, nameList(kex)...)
	p = append(p, nameList(hostkey)...)
	p = append(p, nameList(cipher)...) // enc c2s
	p = append(p, nameList(cipher)...) // enc s2c
	p = append(p, nameList(mac)...)    // mac c2s
	p = append(p, nameList(mac)...)    // mac s2c
	p = append(p, nameList("none,zlib@openssh.com")...)
	p = append(p, nameList("none,zlib@openssh.com")...)
	p = append(p, nameList("")...) // lang c2s
	p = append(p, nameList("")...) // lang s2c
	p = append(p, 0)               // first_kex_packet_follows
	p = append(p, 0, 0, 0, 0)      // reserved
	return p
}

func TestParseKexInit(t *testing.T) {
	payload := buildKexInit(
		"mlkem768x25519-sha256,curve25519-sha256",
		"ssh-ed25519,rsa-sha2-512",
		"chacha20-poly1305@openssh.com,aes256-gcm@openssh.com",
		"hmac-sha2-256-etm@openssh.com",
	)
	k, err := ParseKexInit(payload)
	if err != nil {
		t.Fatalf("ParseKexInit: %v", err)
	}
	if len(k.Kex) != 2 || k.Kex[0] != "mlkem768x25519-sha256" {
		t.Errorf("kex = %v", k.Kex)
	}
	if len(k.HostKeyAlgos) != 2 || k.HostKeyAlgos[0] != "ssh-ed25519" {
		t.Errorf("hostkey = %v", k.HostKeyAlgos)
	}
	if len(k.CiphersS2C) != 2 || k.CiphersS2C[1] != "aes256-gcm@openssh.com" {
		t.Errorf("ciphers = %v", k.CiphersS2C)
	}
	if len(k.MACsS2C) != 1 || k.MACsS2C[0] != "hmac-sha2-256-etm@openssh.com" {
		t.Errorf("macs = %v", k.MACsS2C)
	}
}

func TestParseKexInitRejectsNonKexInit(t *testing.T) {
	if _, err := ParseKexInit([]byte{21, 0, 0}); err == nil {
		t.Error("expected error for non-KEXINIT payload")
	}
	if _, err := ParseKexInit(nil); err == nil {
		t.Error("expected error for empty payload")
	}
}

func TestAssessPQ(t *testing.T) {
	// ML-KEM first → supported and preferred.
	pq := assessPQ([]string{"mlkem768x25519-sha256", "curve25519-sha256"})
	if !pq.Supported || !pq.Preferred {
		t.Errorf("mlkem first: %+v", pq)
	}
	// sntrup present but not first → supported, not preferred.
	pq = assessPQ([]string{"curve25519-sha256", "sntrup761x25519-sha512@openssh.com"})
	if !pq.Supported || pq.Preferred {
		t.Errorf("sntrup second: %+v", pq)
	}
	// no PQ algorithm at all.
	pq = assessPQ([]string{"curve25519-sha256", "ecdh-sha2-nistp256"})
	if pq.Supported {
		t.Errorf("classical only should not be PQ: %+v", pq)
	}
}

func TestGradeModernWithPQ(t *testing.T) {
	k := KexInit{
		Kex:          []string{"mlkem768x25519-sha256", "curve25519-sha256"},
		HostKeyAlgos: []string{"ssh-ed25519"},
		CiphersS2C:   []string{"chacha20-poly1305@openssh.com"},
		CiphersC2S:   []string{"chacha20-poly1305@openssh.com"},
		MACsS2C:      []string{"hmac-sha2-256-etm@openssh.com"},
		MACsC2S:      []string{"hmac-sha2-256-etm@openssh.com"},
	}
	g, _, pq := grade(k, nil)
	if g != "A" {
		t.Errorf("modern+PQ should be A, got %s", g)
	}
	if !pq.Preferred {
		t.Errorf("expected PQ preferred")
	}
}

func TestGradeWeakCipherIsF(t *testing.T) {
	k := KexInit{
		Kex:          []string{"curve25519-sha256"},
		HostKeyAlgos: []string{"ssh-ed25519"},
		CiphersS2C:   []string{"3des-cbc", "aes128-ctr"},
		CiphersC2S:   []string{"3des-cbc", "aes128-ctr"},
		MACsS2C:      []string{"hmac-sha2-256"},
		MACsC2S:      []string{"hmac-sha2-256"},
	}
	g, findings, _ := grade(k, nil)
	if g != "F" {
		t.Errorf("weak cipher should be F, got %s", g)
	}
	if len(findings) == 0 {
		t.Error("expected a finding for the weak cipher")
	}
}

func TestGradeNoPQCapsAtB(t *testing.T) {
	k := KexInit{
		Kex:          []string{"curve25519-sha256", "ecdh-sha2-nistp256"},
		HostKeyAlgos: []string{"ssh-ed25519"},
		CiphersS2C:   []string{"chacha20-poly1305@openssh.com"},
		CiphersC2S:   []string{"chacha20-poly1305@openssh.com"},
		MACsS2C:      []string{"hmac-sha2-256-etm@openssh.com"},
		MACsC2S:      []string{"hmac-sha2-256-etm@openssh.com"},
	}
	g, findings, pq := grade(k, nil)
	if g != "B" {
		t.Errorf("clean-but-no-PQ should be B, got %s", g)
	}
	if pq.Supported {
		t.Error("should not report PQ supported")
	}
	var hasPQFinding bool
	for _, f := range findings {
		if strings.Contains(strings.ToLower(f), "harvest-now") {
			hasPQFinding = true
		}
	}
	if !hasPQFinding {
		t.Errorf("expected harvest-now-decrypt-later finding, got %v", findings)
	}
}

func TestSSHRSAWithSHA2StaysA(t *testing.T) {
	// ssh-rsa offered only for backward compatibility, alongside rsa-sha2-*,
	// must not cap the grade — but is still noted.
	k := KexInit{
		Kex:          []string{"mlkem768x25519-sha256"},
		HostKeyAlgos: []string{"ssh-ed25519", "rsa-sha2-512", "rsa-sha2-256", "ssh-rsa"},
		CiphersS2C:   []string{"chacha20-poly1305@openssh.com"},
		CiphersC2S:   []string{"chacha20-poly1305@openssh.com"},
		MACsS2C:      []string{"hmac-sha2-256-etm@openssh.com"},
		MACsC2S:      []string{"hmac-sha2-256-etm@openssh.com"},
	}
	g, findings, _ := grade(k, nil)
	if g != "A" {
		t.Errorf("ssh-rsa alongside rsa-sha2 should stay A, got %s", g)
	}
	var noted bool
	for _, f := range findings {
		if strings.Contains(f, "ssh-rsa") {
			noted = true
		}
	}
	if !noted {
		t.Error("expected an informational ssh-rsa note")
	}
}

func TestSSHRSAOnlyCapsAtC(t *testing.T) {
	k := KexInit{
		Kex:          []string{"mlkem768x25519-sha256"},
		HostKeyAlgos: []string{"ssh-ed25519", "ssh-rsa"},
		CiphersS2C:   []string{"chacha20-poly1305@openssh.com"},
		CiphersC2S:   []string{"chacha20-poly1305@openssh.com"},
		MACsS2C:      []string{"hmac-sha2-256-etm@openssh.com"},
		MACsC2S:      []string{"hmac-sha2-256-etm@openssh.com"},
	}
	if g, _, _ := grade(k, nil); g != "C" {
		t.Errorf("ssh-rsa with no rsa-sha2 alternative should be C, got %s", g)
	}
}

func TestGradeDSSHostKeyIsF(t *testing.T) {
	k := KexInit{
		Kex:          []string{"mlkem768x25519-sha256"},
		HostKeyAlgos: []string{"ssh-dss", "ssh-ed25519"},
		CiphersS2C:   []string{"aes256-gcm@openssh.com"},
		CiphersC2S:   []string{"aes256-gcm@openssh.com"},
		MACsS2C:      []string{"hmac-sha2-256-etm@openssh.com"},
		MACsC2S:      []string{"hmac-sha2-256-etm@openssh.com"},
	}
	g, _, _ := grade(k, nil)
	if g != "F" {
		t.Errorf("ssh-dss host key should be F, got %s", g)
	}
}

func TestSoftwareFromBanner(t *testing.T) {
	if got := softwareFromBanner("SSH-2.0-OpenSSH_9.6p1 Ubuntu-3ubuntu13"); got != "OpenSSH_9.6p1 Ubuntu-3ubuntu13" {
		t.Errorf("software = %q", got)
	}
	if got := softwareFromBanner("SSH-2.0-OpenSSH_9.6"); got != "OpenSSH_9.6" {
		t.Errorf("software = %q", got)
	}
	if got := softwareFromBanner("garbage"); got != "garbage" {
		t.Errorf("fallback software = %q", got)
	}
}

func TestBuildReport(t *testing.T) {
	k := KexInit{
		Kex:          []string{"mlkem768x25519-sha256"},
		HostKeyAlgos: []string{"ssh-ed25519"},
		CiphersS2C:   []string{"chacha20-poly1305@openssh.com"},
		CiphersC2S:   []string{"chacha20-poly1305@openssh.com"},
		MACsS2C:      []string{"hmac-sha2-256-etm@openssh.com"},
		MACsC2S:      []string{"hmac-sha2-256-etm@openssh.com"},
	}
	r := Build("ssh.example.com:22", "SSH-2.0-OpenSSH_9.9", k, []HostKey{
		{Type: "ssh-ed25519", Bits: 256, Fingerprint: "SHA256:abc"},
	})
	if r.Grade != "A" || r.Software != "OpenSSH_9.9" {
		t.Errorf("report = %+v", r)
	}
	if !strings.Contains(r.Summary, "A") {
		t.Errorf("summary = %q", r.Summary)
	}
}

func TestRemediateTargetsTheActualWeakness(t *testing.T) {
	// A server with a PQ-default key exchange but a legacy MAC (the common
	// Ubuntu-OpenSSH case): the fix should be the MAC, not the key exchange.
	k := KexInit{
		Kex:          []string{"sntrup761x25519-sha512@openssh.com", "curve25519-sha256"},
		CiphersS2C:   []string{"chacha20-poly1305@openssh.com"},
		CiphersC2S:   []string{"chacha20-poly1305@openssh.com"},
		MACsS2C:      []string{"hmac-sha2-256-etm@openssh.com", "hmac-sha1"},
		MACsC2S:      []string{"hmac-sha2-256-etm@openssh.com", "hmac-sha1"},
		HostKeyAlgos: []string{"rsa-sha2-512", "ssh-ed25519", "ssh-rsa"},
	}
	rem := strings.Join(remediate(k, assessPQ(k.Kex)), "\n")
	if !strings.Contains(rem, "/etc/ssh/sshd_config.d/") {
		t.Errorf("expected the fix to point at a sshd_config.d drop-in: %s", rem)
	}
	if !strings.Contains(rem, "MACs ") {
		t.Errorf("expected a MACs fix, got: %s", rem)
	}
	if strings.Contains(rem, "KexAlgorithms ") {
		t.Errorf("PQ is already preferred — should not suggest changing KexAlgorithms: %s", rem)
	}
	if !strings.Contains(rem, "sshd -t") {
		t.Errorf("expected a validate/reload step: %s", rem)
	}
}
