package app

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/hriday/meridian/internal/atlas"
	"github.com/hriday/meridian/internal/settings"
	"github.com/stretchr/testify/require"
)

// useTempSettings points the settings store at a temp dir for the test.
func useTempSettings(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	restore := settings.SetConfigDirForTest(func() (string, error) { return dir, nil })
	t.Cleanup(restore)
}

func TestAtlasKeyStatusMasks(t *testing.T) {
	useTempSettings(t)
	require.NoError(t, settings.SetAtlasKey("abcdef1234567890"))

	st, err := New().AtlasKeyStatus()
	require.NoError(t, err)
	require.True(t, st.Set)
	require.Contains(t, st.Hint, "567890")
	require.NotContains(t, st.Hint, "abcdef1234567890")
}

func TestAtlasKeyStatusUnset(t *testing.T) {
	useTempSettings(t)
	st, err := New().AtlasKeyStatus()
	require.NoError(t, err)
	require.False(t, st.Set)
}

func TestAtlasPingNoKey(t *testing.T) {
	useTempSettings(t)
	_, err := New().AtlasPing("1.1.1.1", 5)
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "key")
}

func TestAtlasPingUsesKey(t *testing.T) {
	useTempSettings(t)
	require.NoError(t, settings.SetAtlasKey("mykey"))

	old := atlasPing
	t.Cleanup(func() { atlasPing = old })
	var gotKey, gotTarget string
	var gotProbes int
	atlasPing = func(ctx context.Context, d atlas.Doer, key, target string, probes int, poll, timeout time.Duration) (atlas.Report, error) {
		gotKey, gotTarget, gotProbes = key, target, probes
		return atlas.Report{Target: target, MeasurementID: 99, Responding: 1}, nil
	}

	rep, err := New().AtlasPing("8.8.8.8", 7)
	require.NoError(t, err)
	require.Equal(t, "mykey", gotKey)
	require.Equal(t, "8.8.8.8", gotTarget)
	require.Equal(t, 7, gotProbes)
	require.Equal(t, 99, rep.MeasurementID)
}

func TestAtlasPingClampsProbes(t *testing.T) {
	useTempSettings(t)
	require.NoError(t, settings.SetAtlasKey("mykey"))

	old := atlasPing
	t.Cleanup(func() { atlasPing = old })
	var gotProbes int
	atlasPing = func(ctx context.Context, d atlas.Doer, key, target string, probes int, poll, timeout time.Duration) (atlas.Report, error) {
		gotProbes = probes
		return atlas.Report{}, nil
	}

	_, err := New().AtlasPing("8.8.8.8", 0) // below floor → default
	require.NoError(t, err)
	require.GreaterOrEqual(t, gotProbes, 1)

	_, err = New().AtlasPing("8.8.8.8", 9999) // above cap → clamped
	require.NoError(t, err)
	require.LessOrEqual(t, gotProbes, 50)
}

func TestAtlasClearKey(t *testing.T) {
	useTempSettings(t)
	require.NoError(t, settings.SetAtlasKey("abc"))
	require.NoError(t, New().ClearAtlasKey())
	st, err := New().AtlasKeyStatus()
	require.NoError(t, err)
	require.False(t, st.Set)
}

func TestAtlasSetKey(t *testing.T) {
	useTempSettings(t)
	require.NoError(t, New().SetAtlasKey("newkey"))
	k, err := settings.AtlasKey()
	require.NoError(t, err)
	require.Equal(t, "newkey", k)
}
