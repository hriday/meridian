package app

import "github.com/hriday/meridian/internal/jwtinspect"

// JWTDecode takes a JWT apart (header, payload, claims, expiry, warnings)
// entirely locally — the token never leaves the machine.
func (a *App) JWTDecode(token string) (jwtinspect.Decoded, error) {
	return jwtinspect.Decode(token)
}

// JWTVerify checks a JWT's signature against a secret (HS*) or a PEM public key
// (RS*/ES*/PS*/EdDSA), locally.
func (a *App) JWTVerify(token, key string) (jwtinspect.VerifyResult, error) {
	return jwtinspect.Verify(token, key)
}
