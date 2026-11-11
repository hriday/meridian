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

// The word "password"/"secret" in banners, descriptions, or community names must
// not be mistaken for a secret.
func TestAnalyzeIgnoresNonSecretText(t *testing.T) {
	cfg := `banner motd ^C Please change your password now ^C
interface Gi0/1
 description reset password portal link
snmp-server community s3cr3t RO
no service password-encryption`
	secs := Analyze(cfg)
	if len(secs) != 0 {
		t.Errorf("no secrets should be found in non-secret text; got %+v", secs)
	}
}

// ASA stores hashed passwords with a trailing `encrypted`; that is a hash, not
// cleartext, and must not be shown as a recovered password.
func TestAnalyzeASAEncryptedIsHash(t *testing.T) {
	cfg := `enable password 2KFQnbNIdI.2KYOU encrypted
passwd aBcDeFgH12345678 encrypted`
	secs := Analyze(cfg)
	if len(secs) != 2 {
		t.Fatalf("want 2 secrets, got %d: %+v", len(secs), secs)
	}
	for _, s := range secs {
		if s.Decoded || s.Plaintext != "" {
			t.Errorf("ASA encrypted must not decode: %+v", s)
		}
		if s.Verdict == "cleartext" || s.Verdict == "reversible" {
			t.Errorf("ASA encrypted should be classified as a hash: %+v", s)
		}
	}
}

// Reversible type-7 keys appear outside password/secret (OSPF md5, key chains,
// TACACS) and must still be decoded.
func TestAnalyzeType7KeysEverywhere(t *testing.T) {
	cfg := `interface Gi0/1
 ip ospf message-digest-key 1 md5 7 0822455D0A16
key chain KC
 key 1
  key-string 7 0822455D0A16
tacacs-server key 7 0822455D0A16`
	secs := Analyze(cfg)
	n := 0
	for _, s := range secs {
		if s.Type == "7" && s.Plaintext == "cisco" {
			n++
		}
	}
	if n != 3 {
		t.Errorf("want 3 decoded type-7 keys, got %d: %+v", n, secs)
	}
}
