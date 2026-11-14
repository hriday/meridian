package app

import "github.com/hriday/meridian/internal/pwhash"

// PwHashGenerate makes a bcrypt hash (and optional htpasswd line) locally — the
// password never leaves the machine.
func (a *App) PwHashGenerate(password, user string, cost int) (pwhash.GenResult, error) {
	return pwhash.Generate(password, user, cost)
}

// PwHashCheck verifies a password against a bcrypt hash (or htpasswd line).
func (a *App) PwHashCheck(password, hash string) (pwhash.CheckResult, error) {
	return pwhash.Check(password, hash)
}
