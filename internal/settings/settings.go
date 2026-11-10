// Package settings persists per-user Meridian configuration to a local file in
// the OS config directory. It currently holds the operator's RIPE Atlas API key,
// which is a secret: it lives only on this machine, in a 0600 file, and is never
// committed, logged, or sent anywhere but RIPE Atlas by the code that uses it.
package settings

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// configDir returns the root config directory; a package var so tests can point
// it at a temp dir.
var configDir = os.UserConfigDir

type file struct {
	AtlasAPIKey string `json:"atlasApiKey"`
}

func path() (string, error) {
	root, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "meridian", "settings.json"), nil
}

func load() (file, error) {
	p, err := path()
	if err != nil {
		return file{}, err
	}
	b, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		return file{}, nil
	}
	if err != nil {
		return file{}, err
	}
	var f file
	if err := json.Unmarshal(b, &f); err != nil {
		return file{}, err
	}
	return f, nil
}

func save(f file) error {
	p, err := path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o600) // owner-only: it holds a secret
}

// AtlasKey returns the stored RIPE Atlas API key, or "" if none is set.
func AtlasKey() (string, error) {
	f, err := load()
	if err != nil {
		return "", err
	}
	return f.AtlasAPIKey, nil
}

// SetAtlasKey stores the key (trimmed). An empty value clears it.
func SetAtlasKey(key string) error {
	f, err := load()
	if err != nil {
		return err
	}
	f.AtlasAPIKey = strings.TrimSpace(key)
	return save(f)
}

// ClearAtlasKey removes the stored key.
func ClearAtlasKey() error {
	return SetAtlasKey("")
}
