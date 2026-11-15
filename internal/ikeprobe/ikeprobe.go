// Package ikeprobe checks an IKE/IPsec VPN responder (IKEv2, UDP/500) by sending
// a real IKE_SA_INIT and reading what comes back: whether it answers, that it's
// IKEv2, which cryptographic proposal it accepts (encryption, PRF, integrity, DH
// group), and whether NAT traversal is supported. It never completes the
// handshake — no authentication, no keys derived — so it's a safe read-only probe
// of a VPN gateway's posture, next to stun / cipherscan / sshaudit.
package ikeprobe

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

// IKEv2 constants (RFC 7296).
const (
	exchangeSAInit = 34
	flagInitiator  = 0x08
	flagResponse   = 0x20

	payloadSA    = 33
	payloadKE    = 34
	payloadNonce = 40
	payloadNotify = 41

	tEncryption = 1
	tPRF        = 2
	tIntegrity  = 3
	tDH         = 4

	dhGroup14 = 14 // 2048-bit MODP — the group we send a KE for

	notifyNoProposalChosen = 14
	notifyInvalidKE        = 17
	notifyNATSource        = 16388
	notifyNATDest          = 16389
)

// Transform is one accepted algorithm in the responder's chosen proposal.
type Transform struct {
	Kind string `json:"kind"` // Encryption / PRF / Integrity / DH
	Name string `json:"name"`
}

// Result is the outcome of probing one IKE responder.
type Result struct {
	Target       string      `json:"target"`
	Responds     bool        `json:"responds"`
	Version      string      `json:"version,omitempty"` // "IKEv2", "IKEv1"
	Accepted     []Transform `json:"accepted,omitempty"`
	NATTraversal bool        `json:"natTraversal"`
	Notify       []string    `json:"notify,omitempty"`
	RTTms        int64       `json:"rttMs,omitempty"`
	Note         string      `json:"note,omitempty"`
}

// Probe sends an IKE_SA_INIT to host (default port 500) and reports the result.
// A timeout with no reply is not an error — it's a Result with Responds=false.
func Probe(ctx context.Context, host string) (Result, error) {
	addr := host
	if _, _, err := net.SplitHostPort(addr); err != nil {
		addr = net.JoinHostPort(host, "500")
	}
	res := Result{Target: addr}

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

	var spi [8]byte
	_, _ = rand.Read(spi[:])
	packet := buildSAInit(spi)

	_ = conn.SetDeadline(time.Now().Add(timeout))
	t0 := time.Now()
	if _, err := conn.Write(packet); err != nil {
		return res, err
	}
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		// No reply: filtered, or nothing listening. Not a hard error.
		res.Note = "no response — UDP/500 may be filtered, or no IKE service here"
		return res, nil
	}
	res.RTTms = time.Since(t0).Milliseconds()
	res.Responds = true
	parseResponse(buf[:n], &res)
	return res, nil
}

// buildSAInit assembles an IKE_SA_INIT request offering a broad set of modern
// transforms with DH group 14 (whose public value we include as the KE payload).
func buildSAInit(initiatorSPI [8]byte) []byte {
	// One proposal, protocol IKE, with alternatives per transform type.
	transforms := [][]byte{
		transform(false, tEncryption, 20, keyLen(256)), // AES-GCM-16-256
		transform(false, tEncryption, 12, keyLen(256)), // AES-CBC-256
		transform(false, tEncryption, 12, keyLen(128)), // AES-CBC-128
		transform(false, tEncryption, 3, nil),          // 3DES
		transform(false, tPRF, 5, nil),                 // HMAC-SHA2-256
		transform(false, tPRF, 6, nil),                 // HMAC-SHA2-384
		transform(false, tPRF, 7, nil),                 // HMAC-SHA2-512
		transform(false, tPRF, 2, nil),                 // HMAC-SHA1
		transform(false, tIntegrity, 12, nil),          // HMAC-SHA2-256-128
		transform(false, tIntegrity, 13, nil),          // HMAC-SHA2-384-192
		transform(false, tIntegrity, 14, nil),          // HMAC-SHA2-512-256
		transform(false, tIntegrity, 2, nil),           // HMAC-SHA1-96
		transform(true, tDH, dhGroup14, nil),           // 2048-bit MODP (last)
	}
	prop := proposal(1, transforms)
	sa := payload(payloadKE, prop)

	// KE payload: group 14, a 256-byte public value. We never derive keys, so a
	// random in-range value is sufficient to elicit the responder's proposal.
	ke := make([]byte, 4+256)
	binary.BigEndian.PutUint16(ke[0:2], dhGroup14)
	_, _ = rand.Read(ke[4:])
	ke[4] = 0x7f // keep it below the MODP prime
	kePayload := payload(payloadNonce, ke)

	// Nonce payload: 32 random bytes.
	nonce := make([]byte, 32)
	_, _ = rand.Read(nonce)
	noncePayload := payload(0, nonce)

	body := append(append(sa, kePayload...), noncePayload...)

	hdr := make([]byte, 28)
	copy(hdr[0:8], initiatorSPI[:]) // initiator SPI
	// responder SPI stays zero
	hdr[16] = payloadSA  // first payload
	hdr[17] = 0x20       // version 2.0
	hdr[18] = exchangeSAInit
	hdr[19] = flagInitiator
	// message ID 0
	binary.BigEndian.PutUint32(hdr[24:28], uint32(28+len(body)))
	return append(hdr, body...)
}

// transform encodes one transform substructure (RFC 7296 §3.3.2).
func transform(last bool, ttype uint8, id uint16, attrs []byte) []byte {
	more := uint8(3) // more transforms follow
	if last {
		more = 0
	}
	length := 8 + len(attrs)
	b := []byte{more, 0, byte(length >> 8), byte(length), ttype, 0, byte(id >> 8), byte(id)}
	return append(b, attrs...)
}

// keyLen is the AES key-length attribute (TV format, type 14).
func keyLen(bits uint16) []byte {
	return []byte{0x80, 0x0e, byte(bits >> 8), byte(bits)}
}

// proposal encodes a single (last) proposal substructure for protocol IKE.
func proposal(num uint8, transforms [][]byte) []byte {
	var body []byte
	for _, t := range transforms {
		body = append(body, t...)
	}
	length := 8 + len(body)
	// 0 = last proposal; protocol 1 = IKE; SPI size 0 (IKE_SA_INIT).
	hdr := []byte{0, 0, byte(length >> 8), byte(length), num, 1, 0, byte(len(transforms))}
	return append(hdr, body...)
}

// payload wraps a body in a generic payload header pointing at the next payload.
func payload(next uint8, body []byte) []byte {
	length := 4 + len(body)
	return append([]byte{next, 0, byte(length >> 8), byte(length)}, body...)
}

// parseResponse fills res from a received IKE datagram.
func parseResponse(b []byte, res *Result) {
	if len(b) < 28 {
		res.Note = "reply too short to be IKE"
		return
	}
	major := b[17] >> 4
	switch major {
	case 2:
		res.Version = "IKEv2"
	case 1:
		res.Version = "IKEv1"
		res.Note = "IKEv1 responder — detailed proposal probing is IKEv2 only"
		return
	default:
		res.Note = fmt.Sprintf("unexpected IKE major version %d", major)
		return
	}

	next := b[16]
	off := 28
	for next != 0 && off+4 <= len(b) {
		plen := int(binary.BigEndian.Uint16(b[off+2 : off+4]))
		if plen < 4 || off+plen > len(b) {
			break
		}
		payloadBody := b[off+4 : off+plen]
		switch next {
		case payloadSA:
			res.Accepted = parseSA(payloadBody)
		case payloadNotify:
			parseNotify(payloadBody, res)
		}
		next = b[off]
		off += plen
	}
	if len(res.Accepted) > 0 && res.Note == "" {
		res.Note = "IKEv2 responder accepted a proposal"
	}
}

// parseSA extracts the accepted transforms from the responder's single proposal.
func parseSA(b []byte) []Transform {
	if len(b) < 8 {
		return nil
	}
	// proposal header: last(1) reserved(1) length(2) num(1) proto(1) spisize(1) ntrans(1)
	spiSize := int(b[6])
	ntrans := int(b[7])
	off := 8 + spiSize
	var out []Transform
	for i := 0; i < ntrans && off+8 <= len(b); i++ {
		tlen := int(binary.BigEndian.Uint16(b[off+2 : off+4]))
		if tlen < 8 || off+tlen > len(b) {
			break
		}
		ttype := b[off+4]
		tid := binary.BigEndian.Uint16(b[off+6 : off+8])
		keyBits := 0
		if tlen > 8 { // a key-length attribute may follow (AES)
			attr := b[off+8 : off+tlen]
			if len(attr) >= 4 && attr[0] == 0x80 && attr[1] == 0x0e {
				keyBits = int(binary.BigEndian.Uint16(attr[2:4]))
			}
		}
		if name := transformName(ttype, tid, keyBits); name != "" {
			out = append(out, Transform{Kind: kindName(ttype), Name: name})
		}
		off += tlen
	}
	return out
}

// parseNotify records notify messages and flags NAT traversal.
func parseNotify(b []byte, res *Result) {
	if len(b) < 4 {
		return
	}
	// protocol(1) spisize(1) notifyType(2) ...
	spiSize := int(b[1])
	if len(b) < 4+spiSize {
		return
	}
	ntype := binary.BigEndian.Uint16(b[2:4])
	switch ntype {
	case notifyNATSource, notifyNATDest:
		res.NATTraversal = true
	case notifyNoProposalChosen:
		res.Notify = append(res.Notify, "NO_PROPOSAL_CHOSEN")
	case notifyInvalidKE:
		msg := "INVALID_KE_PAYLOAD"
		data := b[4+spiSize:]
		if len(data) >= 2 {
			msg = fmt.Sprintf("INVALID_KE_PAYLOAD (wants DH group %d)", binary.BigEndian.Uint16(data[:2]))
		}
		res.Notify = append(res.Notify, msg)
	}
}

func kindName(t uint8) string {
	switch t {
	case tEncryption:
		return "Encryption"
	case tPRF:
		return "PRF"
	case tIntegrity:
		return "Integrity"
	case tDH:
		return "DH"
	}
	return fmt.Sprintf("type%d", t)
}

func transformName(ttype uint8, id uint16, keyBits int) string {
	switch ttype {
	case tEncryption:
		switch id {
		case 3:
			return "3DES"
		case 12:
			if keyBits > 0 {
				return fmt.Sprintf("AES-CBC-%d", keyBits)
			}
			return "AES-CBC"
		case 20:
			if keyBits > 0 {
				return fmt.Sprintf("AES-GCM-16-%d", keyBits)
			}
			return "AES-GCM-16"
		case 28:
			return "ChaCha20-Poly1305"
		}
	case tPRF:
		switch id {
		case 2:
			return "HMAC-SHA1"
		case 5:
			return "HMAC-SHA2-256"
		case 6:
			return "HMAC-SHA2-384"
		case 7:
			return "HMAC-SHA2-512"
		}
	case tIntegrity:
		switch id {
		case 2:
			return "HMAC-SHA1-96"
		case 12:
			return "HMAC-SHA2-256-128"
		case 13:
			return "HMAC-SHA2-384-192"
		case 14:
			return "HMAC-SHA2-512-256"
		}
	case tDH:
		switch id {
		case 2:
			return "MODP-1024 (group 2)"
		case 14:
			return "MODP-2048 (group 14)"
		case 19:
			return "ECP-256 (group 19)"
		case 20:
			return "ECP-384 (group 20)"
		case 21:
			return "ECP-521 (group 21)"
		}
	}
	return fmt.Sprintf("id%d", id)
}
