package cli

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/hriday/meridian/internal/ipspace"
	"github.com/stretchr/testify/require"
)

func TestRunIPSpaceExport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.enc")
	st := ipspace.NewStore("tester")
	_, _ = st.Add("10.0.0.0/8", "Corp", ipspace.Active, nil, "")
	require.NoError(t, ipspace.Save(path, "pw", st.Events()))

	var out, errb bytes.Buffer
	code := Run([]string{"ipspace", "export", "--file", path, "--passphrase", "pw"}, &out, &errb)
	require.Equal(t, 0, code, errb.String())
	require.Contains(t, out.String(), "10.0.0.0/8")
	require.Contains(t, out.String(), "Corp")
}

func TestRunIPSpaceWrongPassphrase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.enc")
	require.NoError(t, ipspace.Create(path, "pw"))
	var out, errb bytes.Buffer
	code := Run([]string{"ipspace", "list", "--file", path, "--passphrase", "bad"}, &out, &errb)
	require.NotEqual(t, 0, code)
	require.NotEmpty(t, errb.String())
}
