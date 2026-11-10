package pwaudit

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/hriday/meridian/internal/securebox"
)

// SaveReport writes an audit report to path, passphrase-encrypted and
// atomically. The report contains recovered plaintext, so this is the only
// sanctioned way it reaches disk.
func SaveReport(path, passphrase string, r AuditReport) error {
	plain, err := json.Marshal(r)
	if err != nil {
		return err
	}
	blob, err := securebox.Seal(passphrase, plain)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".adaudit-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(blob); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}

// LoadReport decrypts and parses an audit report. A wrong passphrase errors.
func LoadReport(path, passphrase string) (AuditReport, error) {
	blob, err := os.ReadFile(path)
	if err != nil {
		return AuditReport{}, err
	}
	plain, err := securebox.Open(passphrase, blob)
	if err != nil {
		return AuditReport{}, err
	}
	var r AuditReport
	if err := json.Unmarshal(plain, &r); err != nil {
		return AuditReport{}, err
	}
	return r, nil
}
