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

func TestAtlasTraceNoKey(t *testing.T) {
	useTempSettings(t)
	_, err := New().AtlasTrace("1.1.1.1", 5)
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "key")
}

func TestAtlasTraceUsesKey(t *testing.T) {
	useTempSettings(t)
	require.NoError(t, settings.SetAtlasKey("mykey"))

	old := atlasTrace
	t.Cleanup(func() { atlasTrace = old })
	var gotKey, gotTarget string
	atlasTrace = func(ctx context.Context, d atlas.Doer, key, target string, probes int, poll, timeout time.Duration) (atlas.TraceReport, error) {
		gotKey, gotTarget = key, target
		return atlas.TraceReport{Target: target, MeasurementID: 123, Reached: 1}, nil
	}

	rep, err := New().AtlasTrace("8.8.8.8", 5)
	require.NoError(t, err)
	require.Equal(t, "mykey", gotKey)
	require.Equal(t, "8.8.8.8", gotTarget)
	require.Equal(t, 123, rep.MeasurementID)
}

func TestAtlasTraceClampsProbes(t *testing.T) {
	useTempSettings(t)
	require.NoError(t, settings.SetAtlasKey("mykey"))

	old := atlasTrace
	t.Cleanup(func() { atlasTrace = old })
	var gotProbes int
	atlasTrace = func(ctx context.Context, d atlas.Doer, key, target string, probes int, poll, timeout time.Duration) (atlas.TraceReport, error) {
		gotProbes = probes
		return atlas.TraceReport{}, nil
	}

	_, err := New().AtlasTrace("8.8.8.8", 0) // below floor → default
	require.NoError(t, err)
	require.GreaterOrEqual(t, gotProbes, 1)

	_, err = New().AtlasTrace("8.8.8.8", 9999) // above cap → clamped
	require.NoError(t, err)
	require.LessOrEqual(t, gotProbes, 20)
}
