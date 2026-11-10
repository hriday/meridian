package pwaudit

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func sampleReport() AuditReport {
	return AuditReport{
		Total: 3, Cracked: 2, PercentCracked: 66.6,
		Results: []Result{
			{Account: Account{Name: "alice"}, Cracked: true, Plaintext: "Summer2024!", Strength: "weak", Pattern: "season+year"},
			{Account: Account{Name: "carol"}},
		},
		CrackerUsed: "hashcat",
	}
}

func TestSaveLoadReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.enc")
	require.NoError(t, SaveReport(path, "pw", sampleReport()))

	got, err := LoadReport(path, "pw")
	require.NoError(t, err)
	require.Equal(t, 2, got.Cracked)
	require.Equal(t, "Summer2024!", got.Results[0].Plaintext)
}

func TestLoadReportWrongPassphrase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.enc")
	require.NoError(t, SaveReport(path, "right", sampleReport()))
	_, err := LoadReport(path, "wrong")
	require.Error(t, err)
}
