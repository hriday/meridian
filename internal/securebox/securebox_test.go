package securebox

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSealOpenRoundTrip(t *testing.T) {
	blob, err := Seal("correct horse", []byte("sensitive report data"))
	require.NoError(t, err)
	require.NotContains(t, string(blob), "sensitive") // actually encrypted

	got, err := Open("correct horse", blob)
	require.NoError(t, err)
	require.Equal(t, []byte("sensitive report data"), got)
}

func TestOpenWrongPassphrase(t *testing.T) {
	blob, err := Seal("right", []byte("x"))
	require.NoError(t, err)
	_, err = Open("wrong", blob)
	require.Error(t, err)
}

func TestOpenTruncated(t *testing.T) {
	_, err := Open("pw", []byte("too short"))
	require.Error(t, err)
}
