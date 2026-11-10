package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hriday/meridian/internal/pwaudit"
	"github.com/stretchr/testify/require"
)

type fakeCracker struct{ cracked map[string]string }

func (f fakeCracker) Detect(context.Context) pwaudit.CrackerInfo {
	return pwaudit.CrackerInfo{Name: "fake", Available: true}
}
func (f fakeCracker) Crack(_ context.Context, _ []string, _ string, _ []string, _ func(pwaudit.Progress)) (map[string]string, error) {
	return f.cracked, nil
}

func TestRunADAudit(t *testing.T) {
	old := newCracker
	t.Cleanup(func() { newCracker = old })
	newCracker = func() pwaudit.Cracker {
		return fakeCracker{cracked: map[string]string{"8846f7eaee8fb117ad06bdd830b7586c": "Password1"}}
	}

	dump := filepath.Join(t.TempDir(), "dump.txt")
	require.NoError(t, os.WriteFile(dump,
		[]byte("jdoe:1104:aad3b435b51404eeaad3b435b51404ee:8846f7eaee8fb117ad06bdd830b7586c:::\n"), 0o600))

	var out, errb bytes.Buffer
	code := Run([]string{"adaudit", "--dump", dump, "--wordlist", "wl.txt"}, &out, &errb)
	require.Equal(t, 0, code, errb.String())
	s := out.String()
	require.Contains(t, s, "Cracked")
	require.Contains(t, s, "1 (100%)")
	require.NotContains(t, s, "Password1") // plaintext hidden without --show
}

func TestRunADAuditShow(t *testing.T) {
	old := newCracker
	t.Cleanup(func() { newCracker = old })
	newCracker = func() pwaudit.Cracker {
		return fakeCracker{cracked: map[string]string{"8846f7eaee8fb117ad06bdd830b7586c": "Password1"}}
	}
	dump := filepath.Join(t.TempDir(), "dump.txt")
	require.NoError(t, os.WriteFile(dump,
		[]byte("jdoe:1104:aad3b435b51404eeaad3b435b51404ee:8846f7eaee8fb117ad06bdd830b7586c:::\n"), 0o600))

	var out, errb bytes.Buffer
	code := Run([]string{"adaudit", "--dump", dump, "--wordlist", "wl.txt", "--show"}, &out, &errb)
	require.Equal(t, 0, code, errb.String())
	require.Contains(t, out.String(), "Password1") // revealed with --show
}

func TestRunADAuditJSONRedacts(t *testing.T) {
	old := newCracker
	t.Cleanup(func() { newCracker = old })
	newCracker = func() pwaudit.Cracker {
		return fakeCracker{cracked: map[string]string{"8846f7eaee8fb117ad06bdd830b7586c": "Password1"}}
	}
	dump := filepath.Join(t.TempDir(), "dump.txt")
	require.NoError(t, os.WriteFile(dump,
		[]byte("jdoe:1104:aad3b435b51404eeaad3b435b51404ee:8846f7eaee8fb117ad06bdd830b7586c:::\n"), 0o600))

	var out, errb bytes.Buffer
	code := Run([]string{"adaudit", "--dump", dump, "--wordlist", "wl.txt", "--json"}, &out, &errb)
	require.Equal(t, 0, code, errb.String())
	require.NotContains(t, out.String(), "Password1") // JSON must not leak plaintext without --show
	require.Contains(t, out.String(), `"cracked":1`)
}

func TestRunADAuditNeedsDump(t *testing.T) {
	var out, errb bytes.Buffer
	require.Equal(t, 2, Run([]string{"adaudit"}, &out, &errb))
}
