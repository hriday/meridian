package ciscopw

import (
	"fmt"
	"testing"
)

// encodeType7 is the inverse used only for round-trip testing.
func encodeType7(plain string, seed int) string {
	out := fmt.Sprintf("%02d", seed)
	for i := 0; i < len(plain); i++ {
		out += fmt.Sprintf("%02X", plain[i]^xlat[(seed+i)%len(xlat)])
	}
	return out
}

func TestDecodeType7KnownVector(t *testing.T) {
	// A widely-published Cisco type-7 vector.
	got, err := DecodeType7("0822455D0A16")
	if err != nil {
		t.Fatalf("DecodeType7: %v", err)
	}
	if got != "cisco" {
		t.Errorf("DecodeType7(0822455D0A16) = %q, want cisco", got)
	}
}

func TestDecodeType7RoundTrip(t *testing.T) {
	for _, p := range []string{"cisco", "S3cr3t!", "a", "longer-passphrase-123"} {
		for _, seed := range []int{0, 7, 15, 42} {
			enc := encodeType7(p, seed)
			got, err := DecodeType7(enc)
			if err != nil {
				t.Fatalf("DecodeType7(%s): %v", enc, err)
			}
			if got != p {
				t.Errorf("round-trip seed %d: decode(encode(%q)) = %q", seed, p, got)
			}
		}
	}
}

func TestDecodeType7Invalid(t *testing.T) {
	for _, bad := range []string{"", "0", "zz11", "08ZZ"} {
		if _, err := DecodeType7(bad); err == nil {
			t.Errorf("DecodeType7(%q) should error", bad)
		}
	}
}

func TestAnalyzeConfig(t *testing.T) {
	cfg := `enable secret 5 $1$mERr$abcdefghijklmnopqrstuv.
enable password 7 0822455D0A16
username admin privilege 15 secret 9 $9$abcd
username bob password 7 104B0718
line vty 0 4
 password 0 PlainText123
`
	secs := Analyze(cfg)
	byCtx := map[string]Secret{}
	for _, s := range secs {
		byCtx[s.Context+"|"+s.Type] = s
	}

	// type 7 under enable → decoded "cisco", reversible
	if s, ok := byCtx["enable|7"]; !ok || !s.Decoded || s.Plaintext != "cisco" {
		t.Errorf("enable password 7 not decoded to cisco: %+v", byCtx["enable|7"])
	}
	// type 7 under username bob → decoded
	if s, ok := byCtx["username bob|7"]; !ok || s.Plaintext == "" {
		t.Errorf("username bob password 7 not decoded: %+v", s)
	}
	// type 5 → weak hash, not decoded
	if s, ok := byCtx["enable|5"]; !ok || s.Decoded || s.Verdict != "weak-hash" {
		t.Errorf("enable secret 5 should be weak-hash/not-decoded: %+v", s)
	}
	// type 9 → strong hash
	if s := byCtx["username admin privilege 15|9"]; s.Verdict != "strong-hash" {
		t.Errorf("secret 9 should be strong-hash: %+v", s)
	}
	// type 0 → cleartext
	var found0 bool
	for _, s := range secs {
		if s.Type == "0" && s.Plaintext == "PlainText123" && s.Verdict == "cleartext" {
			found0 = true
		}
	}
	if !found0 {
		t.Errorf("type-0 cleartext not flagged; secrets=%+v", secs)
	}
}
