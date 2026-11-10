package app

import (
	"context"
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

func TestAppADAuditRun(t *testing.T) {
	a := New()
	a.adCracker = fakeCracker{cracked: map[string]string{"8846f7eaee8fb117ad06bdd830b7586c": "Password1"}}

	accts, err := a.ADAuditParseText("8846f7eaee8fb117ad06bdd830b7586c")
	require.NoError(t, err)
	require.Len(t, accts, 1)

	rep, err := a.ADAuditRun(accts, "wl.txt", nil)
	require.NoError(t, err)
	require.Equal(t, 1, rep.Cracked)
	require.Equal(t, "Password1", rep.Results[0].Plaintext)
	require.Equal(t, "word+digit", rep.Results[0].Pattern)
}

func TestAppADAuditCatalog(t *testing.T) {
	a := New()
	require.NotEmpty(t, a.WordlistCatalog())
}
