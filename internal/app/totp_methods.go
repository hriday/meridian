package app

import (
	"time"

	"github.com/hriday/meridian/internal/totp"
)

// TOTPCode generates the current TOTP code from a base32 secret or an otpauth://
// URI — locally, so the 2FA seed never leaves the machine. The frontend calls it
// each second for a live code + countdown.
func (a *App) TOTPCode(input string) (totp.Code, error) {
	cfg, err := totp.Parse(input)
	if err != nil {
		return totp.Code{}, err
	}
	return totp.Generate(cfg, time.Now().Unix())
}
