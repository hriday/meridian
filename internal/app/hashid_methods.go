package app

import "github.com/hriday/meridian/internal/hashid"

// IdentifyHash returns the likely types (and hashcat modes) of a hash string,
// best guess first. Pure/offline — nothing leaves the machine.
func (a *App) IdentifyHash(hash string) []hashid.Candidate {
	return hashid.Identify(hash)
}
