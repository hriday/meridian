package wgprobe

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"golang.org/x/crypto/blake2s"
)

func TestBuildInitiationShapeAndMAC1(t *testing.T) {
	var peer [32]byte
	_, _ = rand.Read(peer[:])
	msg, err := buildInitiation(peer, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(msg) != initiationLen {
		t.Fatalf("len = %d, want %d", len(msg), initiationLen)
	}
	if msg[0] != msgInitiation {
		t.Errorf("type = %d, want %d", msg[0], msgInitiation)
	}
	// mac1 must authenticate over message[:116] with the peer key.
	key := blake2s.Sum256(append([]byte(labelMAC1), peer[:]...))
	h, _ := blake2s.New128(key[:])
	h.Write(msg[:mac1Offset])
	want := h.Sum(nil)
	got := msg[mac1Offset : mac1Offset+16]
	for i := range want {
		if want[i] != got[i] {
			t.Fatalf("mac1 mismatch at %d", i)
		}
	}
}

func TestBuildInitiationNoKeyZeroMAC1(t *testing.T) {
	var peer [32]byte
	msg, err := buildInitiation(peer, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range msg[mac1Offset : mac1Offset+16] {
		if b != 0 {
			t.Fatal("mac1 should be zero when no key is given")
		}
	}
}

func TestReplyType(t *testing.T) {
	if replyType([]byte{3}) != "cookie-reply" {
		t.Error("type 3 should be cookie-reply")
	}
	if replyType([]byte{2}) != "handshake-response" {
		t.Error("type 2 should be handshake-response")
	}
	if replyType([]byte{9}) != "type 9" {
		t.Error("unknown type")
	}
}

func TestProbeNoResponder(t *testing.T) {
	// Nothing listens on this loopback port → no response, verdict mentions the
	// missing peer key (graceful, not an error).
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	res, err := Probe(ctx, "127.0.0.1:51999", "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Responds {
		t.Error("should not respond")
	}
	if res.Verdict == "" {
		t.Error("expected a verdict")
	}
}

func TestProbeRejectsBadKey(t *testing.T) {
	if _, err := Probe(context.Background(), "127.0.0.1", "not-base64!!"); err == nil {
		t.Fatal("bad peer key should error")
	}
}
