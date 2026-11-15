package ikeprobe

import (
	"encoding/binary"
	"testing"
)

func TestBuildSAInitHeader(t *testing.T) {
	var spi [8]byte
	spi[0] = 0xab
	pkt := buildSAInit(spi)
	if len(pkt) < 28 {
		t.Fatalf("packet too short: %d", len(pkt))
	}
	if pkt[17] != 0x20 {
		t.Errorf("version = %#x, want 0x20 (IKEv2)", pkt[17])
	}
	if pkt[18] != exchangeSAInit {
		t.Errorf("exchange type = %d, want %d", pkt[18], exchangeSAInit)
	}
	if pkt[19] != flagInitiator {
		t.Errorf("flags = %#x, want initiator %#x", pkt[19], flagInitiator)
	}
	if pkt[16] != payloadSA {
		t.Errorf("first payload = %d, want SA %d", pkt[16], payloadSA)
	}
	if got := int(binary.BigEndian.Uint32(pkt[24:28])); got != len(pkt) {
		t.Errorf("length field = %d, want %d", got, len(pkt))
	}
}

func TestParseResponseExtractsProposalAndNAT(t *testing.T) {
	// Chosen proposal: AES-CBC-256 / HMAC-SHA2-256 / HMAC-SHA2-256-128 / group 14.
	transforms := [][]byte{
		transform(false, tEncryption, 12, keyLen(256)),
		transform(false, tPRF, 5, nil),
		transform(false, tIntegrity, 12, nil),
		transform(true, tDH, 14, nil),
	}
	sa := payload(payloadNotify, proposal(1, transforms))
	// NAT_DETECTION_SOURCE_IP notify: protocol 0, spisize 0, type 16388.
	nat := payload(0, []byte{0, 0, byte(notifyNATSource >> 8), byte(notifyNATSource & 0xff)})
	body := append(sa, nat...)

	hdr := make([]byte, 28)
	hdr[16] = payloadSA
	hdr[17] = 0x20
	hdr[18] = exchangeSAInit
	hdr[19] = flagResponse
	binary.BigEndian.PutUint32(hdr[24:28], uint32(28+len(body)))
	pkt := append(hdr, body...)

	var res Result
	res.Responds = true
	parseResponse(pkt, &res)

	if res.Version != "IKEv2" {
		t.Fatalf("version = %q", res.Version)
	}
	if !res.NATTraversal {
		t.Error("expected NAT traversal detected")
	}
	got := map[string]string{}
	for _, tr := range res.Accepted {
		got[tr.Kind] = tr.Name
	}
	want := map[string]string{
		"Encryption": "AES-CBC-256",
		"PRF":        "HMAC-SHA2-256",
		"Integrity":  "HMAC-SHA2-256-128",
		"DH":         "MODP-2048 (group 14)",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
}

func TestParseResponseIKEv1(t *testing.T) {
	hdr := make([]byte, 28)
	hdr[17] = 0x10 // IKEv1
	var res Result
	parseResponse(hdr, &res)
	if res.Version != "IKEv1" {
		t.Errorf("version = %q, want IKEv1", res.Version)
	}
}

func TestParseNotifyNoProposal(t *testing.T) {
	nat := payload(0, []byte{0, 0, byte(notifyNoProposalChosen >> 8), byte(notifyNoProposalChosen)})
	hdr := make([]byte, 28)
	hdr[16] = payloadNotify
	hdr[17] = 0x20
	binary.BigEndian.PutUint32(hdr[24:28], uint32(28+len(nat)))
	var res Result
	parseResponse(append(hdr, nat...), &res)
	if len(res.Notify) != 1 || res.Notify[0] != "NO_PROPOSAL_CHOSEN" {
		t.Errorf("notify = %v, want [NO_PROPOSAL_CHOSEN]", res.Notify)
	}
}
