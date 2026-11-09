package ipspace

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func sampleEvents() []Event {
	return []Event{
		{ID: "e1", Type: EventAdd, Actor: "alice", At: time.Unix(1000, 0).UTC(), Record: Record{ID: "A", Prefix: "10.0.0.0/8", Name: "Corp"}},
		{ID: "e2", Type: EventAdd, Actor: "alice", At: time.Unix(2000, 0).UTC(), Record: Record{ID: "B", Prefix: "10.1.0.0/16", Name: "HQ", ParentID: "A"}},
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.enc")
	ev := sampleEvents()
	require.NoError(t, Save(path, "correct horse", ev))

	got, err := Load(path, "correct horse")
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, "10.1.0.0/16", got[1].Record.Prefix)
	require.True(t, ev[0].At.Equal(got[0].At))
}

func TestLoadWrongPassphrase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.enc")
	require.NoError(t, Save(path, "right", sampleEvents()))
	_, err := Load(path, "wrong")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unlock")
}

func TestLoadTamperedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.enc")
	require.NoError(t, Save(path, "pw", sampleEvents()))
	blob, err := os.ReadFile(path)
	require.NoError(t, err)
	blob[len(blob)-1] ^= 0xff
	require.NoError(t, os.WriteFile(path, blob, 0o600))
	_, err = Load(path, "pw")
	require.Error(t, err)
}

func TestLoadMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nope.enc")
	require.False(t, Exists(path))
	_, err := Load(path, "pw")
	require.Error(t, err)
}
