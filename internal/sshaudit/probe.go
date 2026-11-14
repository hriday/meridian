package sshaudit

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

const clientBanner = "SSH-2.0-meridian_audit\r\n"

// normalizeHostPort defaults the port to 22.
func normalizeHostPort(s string) string {
	s = strings.TrimSpace(s)
	if _, _, err := net.SplitHostPort(s); err == nil {
		return s
	}
	return net.JoinHostPort(s, "22")
}

// Probe connects to host (default :22), reads its identification banner and
// KEXINIT, collects host-key fingerprints, and returns a graded report. It
// never authenticates: the algorithm inventory comes from the cleartext
// KEXINIT, and host keys from handshakes aborted the instant the key arrives.
func Probe(ctx context.Context, host string) (Report, error) {
	hostport := normalizeHostPort(host)
	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", hostport)
	if err != nil {
		return Report{}, fmt.Errorf("connect %s: %w", hostport, err)
	}
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	} else {
		_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	}
	serverID, kex, err := readKexInitConn(conn)
	_ = conn.Close()
	if err != nil {
		return Report{}, err
	}
	keys := fetchHostKeys(ctx, hostport, kex.HostKeyAlgos)
	return Build(hostport, serverID, kex, keys), nil
}

// readKexInitConn performs the SSH identification exchange on an open
// connection and reads the server's first binary packet (its KEXINIT).
func readKexInitConn(conn net.Conn) (string, KexInit, error) {
	br := bufio.NewReader(conn)

	// The server may send informational lines before its SSH- identification.
	var serverID string
	for i := 0; i < 50; i++ {
		line, err := br.ReadString('\n')
		if err != nil {
			return "", KexInit{}, fmt.Errorf("read SSH banner: %w", err)
		}
		if strings.HasPrefix(line, "SSH-") {
			serverID = strings.TrimRight(line, "\r\n")
			break
		}
	}
	if serverID == "" {
		return "", KexInit{}, errors.New("no SSH identification string — not an SSH service?")
	}
	if _, err := conn.Write([]byte(clientBanner)); err != nil {
		return "", KexInit{}, fmt.Errorf("send identification: %w", err)
	}

	payload, err := readPacketPayload(br)
	if err != nil {
		return serverID, KexInit{}, err
	}
	kex, err := ParseKexInit(payload)
	if err != nil {
		return serverID, KexInit{}, err
	}
	return serverID, kex, nil
}

// readPacketPayload reads one unencrypted SSH binary packet (RFC 4253 §6) and
// returns its payload.
func readPacketPayload(r io.Reader) ([]byte, error) {
	var hdr [5]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, fmt.Errorf("read packet header: %w", err)
	}
	pktLen := binary.BigEndian.Uint32(hdr[:4])
	padLen := uint32(hdr[4])
	if pktLen < 2 || pktLen > 256*1024 {
		return nil, fmt.Errorf("implausible SSH packet length %d", pktLen)
	}
	if padLen+1 > pktLen {
		return nil, errors.New("malformed SSH packet (padding exceeds length)")
	}
	body := make([]byte, pktLen-1) // packet_length counts padding_length byte + payload + padding
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, fmt.Errorf("read packet body: %w", err)
	}
	payloadLen := uint32(len(body)) - padLen
	return body[:payloadLen], nil
}

// representativeAlgos picks one host-key algorithm per underlying key type, so
// one handshake per key suffices.
func representativeAlgos(advertised []string) []string {
	byBase := map[string]string{}
	base := func(a string) string {
		switch {
		case strings.Contains(a, "ed25519"):
			return "ed25519"
		case strings.Contains(a, "ecdsa"):
			return "ecdsa"
		case strings.Contains(a, "dss"):
			return "dss"
		case strings.Contains(a, "rsa"):
			return "rsa"
		default:
			return a
		}
	}
	// Prefer the strongest signature variant for RSA.
	rank := func(a string) int {
		switch {
		case strings.HasPrefix(a, "rsa-sha2-512"):
			return 3
		case strings.HasPrefix(a, "rsa-sha2-256"):
			return 2
		default:
			return 1
		}
	}
	for _, a := range advertised {
		if strings.Contains(a, "cert-v01") {
			continue // skip certificate algorithms; we want the bare key
		}
		b := base(a)
		if cur, ok := byBase[b]; !ok || rank(a) > rank(cur) {
			byBase[b] = a
		}
	}
	var out []string
	for _, a := range byBase {
		out = append(out, a)
	}
	return out
}

// fetchHostKeys performs one aborted handshake per host-key type and returns
// the fingerprints. Failures per type are skipped silently (best-effort).
func fetchHostKeys(ctx context.Context, hostport string, advertised []string) []HostKey {
	var keys []HostKey
	seen := map[string]bool{}
	for _, algo := range representativeAlgos(advertised) {
		hk, ok := oneHostKey(ctx, hostport, algo)
		if ok && !seen[hk.Fingerprint] {
			seen[hk.Fingerprint] = true
			keys = append(keys, hk)
		}
	}
	return keys
}

var errGotHostKey = errors.New("host key captured")

func oneHostKey(ctx context.Context, hostport, algo string) (HostKey, bool) {
	var captured ssh.PublicKey
	cfg := &ssh.ClientConfig{
		User:              "meridian",
		HostKeyAlgorithms: []string{algo},
		HostKeyCallback: func(_ string, _ net.Addr, key ssh.PublicKey) error {
			captured = key
			return errGotHostKey // abort before authentication
		},
		Timeout: 8 * time.Second,
	}
	d := net.Dialer{Timeout: 8 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", hostport)
	if err != nil {
		return HostKey{}, false
	}
	defer conn.Close()
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	} else {
		_ = conn.SetDeadline(time.Now().Add(8 * time.Second))
	}
	// We expect this to fail with errGotHostKey (or an auth error); either way
	// captured is set if the key arrived.
	_, _, _, _ = ssh.NewClientConn(conn, hostport, cfg)
	if captured == nil {
		return HostKey{}, false
	}
	return HostKey{
		Type:        captured.Type(),
		Bits:        keyBits(captured),
		Fingerprint: ssh.FingerprintSHA256(captured),
	}, true
}

func keyBits(key ssh.PublicKey) int {
	ck, ok := key.(ssh.CryptoPublicKey)
	if !ok {
		return 0
	}
	switch pub := ck.CryptoPublicKey().(type) {
	case *rsa.PublicKey:
		return pub.N.BitLen()
	case *ecdsa.PublicKey:
		return pub.Curve.Params().BitSize
	case ed25519.PublicKey:
		return 256
	default:
		return 0
	}
}
