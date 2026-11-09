package ipspace

import (
	"os"
	"os/user"
	"path/filepath"
)

// Actor identifies who is making a change: the MERIDIAN_ACTOR override, else the
// OS username, else "unknown". Stamped on every event (team/conflict metadata).
func Actor() string {
	if a := os.Getenv("MERIDIAN_ACTOR"); a != "" {
		return a
	}
	if u, err := user.Current(); err == nil && u.Username != "" {
		return u.Username
	}
	return "unknown"
}

// DefaultPath is the standard store location in the OS app-data directory.
func DefaultPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "Meridian", "ipspace.enc")
}
