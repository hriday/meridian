package secheaders

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHeaderFixGivesConcreteValues(t *testing.T) {
	require.Contains(t, headerFix(Header{Name: "Content-Security-Policy", Present: false}, true), "Content-Security-Policy: default-src")
	require.Contains(t, headerFix(Header{Name: "Content-Security-Policy", Present: true}, true), "unsafe-inline")
	require.Equal(t, "X-Content-Type-Options: nosniff", headerFix(Header{Name: "X-Content-Type-Options"}, true))
	require.Contains(t, headerFix(Header{Name: "Strict-Transport-Security"}, false), "HTTPS first")
	require.Contains(t, headerFix(Header{Name: "Strict-Transport-Security"}, true), "max-age=63072000")
	require.Empty(t, headerFix(Header{Name: "Unknown-Header"}, true))
}

func TestRemediateSkipsOKHeaders(t *testing.T) {
	r := &Report{HTTPS: true, Headers: []Header{
		{Name: "X-Content-Type-Options", Grade: GradeOK},
		{Name: "Content-Security-Policy", Grade: GradeMissing},
	}}
	out := r.remediate()
	require.Len(t, out, 2) // a "where to set these" line + the one header fix
	require.Contains(t, out[0], "add_header")
	require.Contains(t, out[1], "Content-Security-Policy: default-src")
}
