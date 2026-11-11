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

func TestSealOpenBytesRoundTrip(t *testing.T) {
	plain := []byte(`{"endpoint":"https://r2.example.com","secret":"shh"}`)
	blob, err := Seal("team", plain)
	require.NoError(t, err)
	require.NotEmpty(t, blob)
	require.NotContains(t, string(blob), "shh", "the secret must not appear in the sealed blob")

	got, err := Open("team", blob)
	require.NoError(t, err)
	require.Equal(t, plain, got)
}

func TestOpenBytesWrongPassphrase(t *testing.T) {
	blob, err := Seal("right", []byte("hello"))
	require.NoError(t, err)
	_, err = Open("wrong", blob)
	require.Error(t, err)
}

func TestSealOpenRoundTrip(t *testing.T) {
	ev := sampleEvents()
	blob, err := SealEvents("correct horse", ev)
	require.NoError(t, err)
	require.NotEmpty(t, blob)

	got, err := OpenEvents("correct horse", blob)
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, "10.1.0.0/16", got[1].Record.Prefix)
	require.True(t, ev[0].At.Equal(got[0].At))
}

func TestOpenWrongPassphrase(t *testing.T) {
	blob, err := SealEvents("right", sampleEvents())
	require.NoError(t, err)
	_, err = OpenEvents("wrong", blob)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unlock")
}

func TestOpenTruncatedBlob(t *testing.T) {
	_, err := OpenEvents("pw", []byte{1, 2, 3})
	require.Error(t, err)
}

// Save/Load must stay interoperable with Seal/Open: a blob written by Save on
// disk opens with OpenEvents, and vice versa.
func TestSaveLoadSealOpenInterop(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.enc")
	require.NoError(t, Save(path, "pw", sampleEvents()))
	blob, err := os.ReadFile(path)
	require.NoError(t, err)
	got, err := OpenEvents("pw", blob)
	require.NoError(t, err)
	require.Len(t, got, 2)
}
