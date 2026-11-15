// Package wgprobe checks for a WireGuard responder (UDP, default 51820) by
// sending a Noise handshake-initiation message and watching for a reply.
//
// WireGuard is deliberately silent: it drops any initiation whose mac1 doesn't
// authenticate against the responder's own static public key, and it drops a
// valid-mac1 initiation it can't decrypt — unless it's rate-limiting, when it
// answers with a cookie reply. So:
//   - With the peer's public key we compute a valid mac1; a cookie reply (or a
//     handshake response) then positively confirms a WireGuard responder.
//   - Without it, mac1 is invalid, the packet is always silently dropped, and
//     silence proves nothing. The probe says so rather than guessing.
package wgprobe

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/blake2s"
	"golang.org/x/crypto/curve25519"
)

const (
	msgInitiation      = 1
	msgResponse        = 2
	msgCookieReply     = 3
	initiationLen      = 148
	mac1Offset         = 116 // bytes covered by mac1 = message[:116]
	labelMAC1          = "mac1----"
	defaultWGPort      = "51820"
)

// Result is the outcome of probing one UDP endpoint for WireGuard.
type Result struct {
	Target    string `json:"target"`
	ValidMAC1 bool   `json:"validMac1"` // true when a peer key let us authenticate the initiation
	Responds  bool   `json:"responds"`
	ReplyType string `json:"replyType,omitempty"` // cookie-reply | handshake-response | type N
	RTTms     int64  `json:"rttMs,omitempty"`
	Verdict   string `json:"verdict"`
}

// Probe sends a WireGuard handshake initiation to host (default port 51820). If
// peerPubKeyB64 is a 32-byte base64 Curve25519 public key, the initiation
// carries a valid mac1; otherwise mac1 is zero (and a non-response is expected).
func Probe(ctx context.Context, host, peerPubKeyB64 string) (Result, error) {
	addr := host
	if _, _, err := net.SplitHostPort(addr); err != nil {
		addr = net.JoinHostPort(host, defaultWGPort)
	}
	res := Result{Target: addr}

	var peerPub [32]byte
	hasKey := false
	if peerPubKeyB64 != "" {
		b, err := base64.StdEncoding.DecodeString(peerPubKeyB64)
		if err != nil || len(b) != 32 {
			return res, fmt.Errorf("peer public key must be 32-byte base64 (a WireGuard public key)")
		}
		copy(peerPub[:], b)
		hasKey = true
	}
	res.ValidMAC1 = hasKey

	pkt, err := buildInitiation(peerPub, hasKey)
	if err != nil {
		return res, err
	}

	timeout := 5 * time.Second
	if dl, ok := ctx.Deadline(); ok {
		timeout = time.Until(dl)
	}
	raddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return res, fmt.Errorf("resolving %s: %w", addr, err)
	}
	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return res, err
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(timeout))
	t0 := time.Now()
	if _, err := conn.Write(pkt); err != nil {
		return res, err
	}
	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err == nil && n > 0 {
		res.Responds = true
		res.RTTms = time.Since(t0).Milliseconds()
		res.ReplyType = replyType(buf[:n])
	}
	res.Verdict = verdict(hasKey, res.Responds, res.ReplyType)
	return res, nil
}

// buildInitiation assembles a 148-byte handshake initiation. The encrypted
// fields are random (we have no valid identity), which is fine: mac1 covers the
// message bytes, not their correctness, and the responder fails AEAD later.
func buildInitiation(peerPub [32]byte, computeMAC1 bool) ([]byte, error) {
	msg := make([]byte, initiationLen)
	msg[0] = msgInitiation
	if _, err := rand.Read(msg[4:8]); err != nil { // sender index
		return nil, err
	}
	// Ephemeral public from a random private, so it's a real curve point.
	var ePriv [32]byte
	if _, err := rand.Read(ePriv[:]); err != nil {
		return nil, err
	}
	ePub, err := curve25519.X25519(ePriv[:], curve25519.Basepoint)
	if err != nil {
		return nil, err
	}
	copy(msg[8:40], ePub)
	if _, err := rand.Read(msg[40:116]); err != nil { // encrypted static + timestamp
		return nil, err
	}
	if computeMAC1 {
		copy(msg[mac1Offset:mac1Offset+16], mac1(peerPub, msg[:mac1Offset]))
	}
	// mac2 stays zero (only set when answering a cookie).
	return msg, nil
}

// mac1 = BLAKE2s-128( key = BLAKE2s-256("mac1----" || peerPub), data ).
func mac1(peerPub [32]byte, data []byte) []byte {
	key := blake2s.Sum256(append([]byte(labelMAC1), peerPub[:]...))
	h, _ := blake2s.New128(key[:])
	h.Write(data)
	return h.Sum(nil)
}

func replyType(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	switch b[0] {
	case msgResponse:
		return "handshake-response"
	case msgCookieReply:
		return "cookie-reply"
	default:
		return fmt.Sprintf("type %d", b[0])
	}
}

func verdict(hasKey, responds bool, reply string) string {
	switch {
	case responds && reply == "cookie-reply":
		return "WireGuard responder confirmed — it answered with a cookie reply (it's rate-limiting / under load)."
	case responds && reply == "handshake-response":
		return "WireGuard responder confirmed — it completed a handshake response."
	case responds:
		return "Got a reply (" + reply + ") — something is on this port, but not a recognised WireGuard message."
	case hasKey:
		return "No reply. A valid handshake was sent; WireGuard silently drops initiations it can't authenticate, so this is inconclusive — it may be a WireGuard endpoint that dropped it, or nothing is listening."
	default:
		return "No reply (expected). Without the peer's public key the initiation's mac1 is invalid and WireGuard silently drops it — supply the peer public key for a meaningful probe."
	}
}
