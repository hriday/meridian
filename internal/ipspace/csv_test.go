package ipspace

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportCSVSkipsBadRows(t *testing.T) {
	s := NewStore("a")
	csv := "prefix,name,status,tags,notes\n" +
		"10.0.0.0/8,Corp,active,,\n" +
		"10.1.0.0/16,HQ,active,prod|servers,the note\n" +
		"notacidr,Bad,active,,\n" +
		"10.0.0.0/8,Dup,active,,\n"
	rep, err := s.ImportCSV(strings.NewReader(csv))
	require.NoError(t, err)
	require.Equal(t, 2, rep.Added)
	require.Equal(t, 2, rep.Skipped)
	require.Len(t, rep.Errors, 2)
	require.Len(t, s.Records(), 2)
}

func TestExportCSVRoundTrips(t *testing.T) {
	s := NewStore("a")
	_, _ = s.Add("10.0.0.0/8", "Corp", Active, []string{"a", "b"}, "n")
	_, _ = s.Add("10.1.0.0/16", "HQ", Active, nil, "")

	var buf bytes.Buffer
	require.NoError(t, s.ExportCSV(&buf))

	s2 := NewStore("a")
	rep, err := s2.ImportCSV(&buf)
	require.NoError(t, err)
	require.Equal(t, 2, rep.Added)

	got := s2.Records()
	require.Len(t, got, 2)
	require.Equal(t, "10.0.0.0/8", got[0].Prefix)
	require.Equal(t, "Corp", got[0].Name)
	require.Equal(t, []string{"a", "b"}, got[0].Tags)
	require.Equal(t, "10.1.0.0/16", got[1].Prefix)
}
