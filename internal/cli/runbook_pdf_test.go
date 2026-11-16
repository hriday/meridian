package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderRunbookPDF(t *testing.T) {
	d := RunbookData{
		Profile: "org-recon", Name: "Org Recon", Target: "example.com", Generated: "2026-11-16T00:00:00Z",
		Steps: []RunbookStep{
			{Command: "dnsaudit", Args: []string{"example.com"}, Output: "ok DNSSEC validated\nwarn SPF ~all", OK: true},
			{Command: "axfr", Args: []string{"example.com"}, Output: "", OK: false},
			// A very long unbroken token must not overflow the cell.
			{Command: "lookup", Args: []string{"example.com"}, Output: strings.Repeat("A", 400) + "\n✓ done · ok", OK: true},
		},
	}
	out, err := RenderRunbookPDF(d)
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(out, []byte("%PDF")), "output should be a PDF document")
	require.Greater(t, len(out), 800, "a real multi-step PDF is more than a stub")
}

func TestWrapLong(t *testing.T) {
	require.Equal(t, "abc", wrapLong("abc", 10))
	require.Equal(t, "abcde\nfghij\nk", wrapLong("abcdefghijk", 5))
	require.Equal(t, "ab\ncd", wrapLong("ab\ncd", 5), "existing newlines are preserved")
}
