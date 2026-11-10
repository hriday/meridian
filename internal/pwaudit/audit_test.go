package pwaudit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeCracker struct {
	avail   bool
	cracked map[string]string
}

func (f fakeCracker) Detect(context.Context) CrackerInfo {
	return CrackerInfo{Name: "fake", Available: f.avail}
}
func (f fakeCracker) Crack(_ context.Context, hashes []string, _ string, _ []string, onP func(Progress)) (map[string]string, error) {
	if onP != nil {
		onP(Progress{Recovered: len(f.cracked), Total: len(hashes), Percent: 100})
	}
	return f.cracked, nil
}

const hashA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const hashB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

func TestRunAudit(t *testing.T) {
	accounts := []Account{
		{Name: "alice", NTHash: hashA},
		{Name: "bob", NTHash: hashA}, // shares alice's password
		{Name: "carol", NTHash: hashB},
	}
	fc := fakeCracker{avail: true, cracked: map[string]string{hashA: "Summer2024!"}}

	progressed := 0
	rep, err := RunAudit(context.Background(), accounts, fc, "wl.txt", nil, func(Progress) { progressed++ })
	require.NoError(t, err)

	require.Equal(t, 3, rep.Total)
	require.Equal(t, 2, rep.Cracked) // alice + bob (shared hash)
	require.InDelta(t, 66.6, rep.PercentCracked, 1.0)
	require.Len(t, rep.Reuse, 1) // alice+bob
	require.Equal(t, "fake", rep.CrackerUsed)
	require.Greater(t, progressed, 0)

	// alice's result carries the classification
	var alice Result
	for _, r := range rep.Results {
		if r.Account.Name == "alice" {
			alice = r
		}
	}
	require.True(t, alice.Cracked)
	require.Equal(t, "Summer2024!", alice.Plaintext)
	require.Equal(t, "season+year", alice.Pattern)

	// pattern summary includes season+year
	found := false
	for _, ps := range rep.Patterns {
		if ps.Pattern == "season+year" {
			found = true
		}
	}
	require.True(t, found)
}

func TestRunAuditNoCracker(t *testing.T) {
	_, err := RunAudit(context.Background(), []Account{{NTHash: hashA}}, fakeCracker{avail: false}, "wl", nil, nil)
	require.Error(t, err)
}
