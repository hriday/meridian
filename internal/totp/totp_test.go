package totp

import "testing"

// RFC 6238 test secret "12345678901234567890" in base32.
const rfcSecret = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

func TestGenerateRFCVector(t *testing.T) {
	// RFC 6238 Appendix B: SHA1, T=59 → 8-digit 94287082 → 6-digit 287082.
	c, err := Generate(Config{Secret: rfcSecret, Digits: 6, Period: 30, Algorithm: "SHA1"}, 59)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if c.Code != "287082" {
		t.Errorf("code = %q, want 287082", c.Code)
	}
	// T=59, period 30: 59 % 30 = 29 used, 1 second left.
	if c.SecondsLeft != 1 {
		t.Errorf("secondsLeft = %d, want 1", c.SecondsLeft)
	}
}

func TestGenerate8Digit(t *testing.T) {
	c, err := Generate(Config{Secret: rfcSecret, Digits: 8, Period: 30, Algorithm: "SHA1"}, 59)
	if err != nil {
		t.Fatal(err)
	}
	if c.Code != "94287082" {
		t.Errorf("8-digit code = %q, want 94287082", c.Code)
	}
}

func TestParseBareSecret(t *testing.T) {
	cfg, err := Parse("gezd gnbv gy3t qojq")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Digits != 6 || cfg.Period != 30 || cfg.Algorithm != "SHA1" {
		t.Errorf("defaults wrong: %+v", cfg)
	}
	// spaces are stripped and it decodes fine
	if _, err := Generate(cfg, 59); err != nil {
		t.Errorf("spaced secret should decode: %v", err)
	}
}

func TestParseOtpauthURI(t *testing.T) {
	uri := "otpauth://totp/GitHub:alice@example.com?secret=" + rfcSecret + "&issuer=GitHub&digits=8&period=60&algorithm=SHA256"
	cfg, err := Parse(uri)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Secret != rfcSecret {
		t.Errorf("secret = %q", cfg.Secret)
	}
	if cfg.Digits != 8 || cfg.Period != 60 || cfg.Algorithm != "SHA256" {
		t.Errorf("params not parsed: %+v", cfg)
	}
	if cfg.Issuer != "GitHub" || cfg.Account != "alice@example.com" {
		t.Errorf("label not parsed: issuer=%q account=%q", cfg.Issuer, cfg.Account)
	}
}

func TestBadSecret(t *testing.T) {
	if _, err := Generate(Config{Secret: "!!!notbase32!!!"}, 0); err == nil {
		t.Error("invalid base32 should error")
	}
}
