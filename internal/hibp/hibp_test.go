package hibp

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeDoer returns a canned range response per prefix and records requests.
type fakeDoer struct {
	byPrefix map[string]string // prefix -> response body
	reqs     []*http.Request
	status   int
}

func (f *fakeDoer) Do(req *http.Request) (*http.Response, error) {
	f.reqs = append(f.reqs, req)
	status := f.status
	if status == 0 {
		status = 200
	}
	// URL is .../range/PREFIX?mode=ntlm
	prefix := strings.TrimSuffix(strings.TrimPrefix(req.URL.Path, "/range/"), "")
	body := f.byPrefix[prefix]
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}, nil
}

func TestCheckNTLMFoundAndNotFound(t *testing.T) {
	// Two NTLM hashes sharing a prefix + one on another prefix. Suffixes are the
	// 27 chars after the 5-char prefix; build them so lengths are exact.
	suf1 := "7BDB5FDC5E9CBA88547376818D4" // 27
	suf2 := strings.Repeat("A", 27)
	suf3 := strings.Repeat("B", 26) + "0"
	h1 := "32ED8" + suf1
	h2 := "32ED8" + suf2
	h3 := "AAAAA" + suf3
	require.Len(t, h1, 32)
	require.Len(t, h3, 32)

	f := &fakeDoer{byPrefix: map[string]string{
		"32ED8": suf1 + ":17\r\n" + strings.Repeat("0", 27) + ":0\r\n" + strings.Repeat("F", 27) + ":5",
		"AAAAA": suf3 + ":3",
	}}

	got, err := CheckNTLM(context.Background(), f, []string{h1, h2, h3})
	require.NoError(t, err)
	require.Equal(t, 17, got[h1], "h1 is in breaches 17 times")
	require.Equal(t, 0, got[h2], "h2 shares the prefix but isn't in the list")
	require.Equal(t, 3, got[h3])

	// Each distinct prefix is queried exactly once, with the NTLM mode + padding.
	require.Len(t, f.reqs, 2)
	for _, r := range f.reqs {
		require.Equal(t, "true", r.Header.Get("Add-Padding"))
		require.NotEmpty(t, r.Header.Get("User-Agent"))
		require.Equal(t, "ntlm", r.URL.Query().Get("mode"))
	}
}

func TestCheckNTLMCaseInsensitiveInput(t *testing.T) {
	// A lowercase input hash (as the AD parser produces) matches an uppercase API suffix.
	hLower := "32ed87bdb5fdc5e9cba88547376818d4"
	f := &fakeDoer{byPrefix: map[string]string{"32ED8": "7BDB5FDC5E9CBA88547376818D4:9"}}
	got, err := CheckNTLM(context.Background(), f, []string{hLower})
	require.NoError(t, err)
	require.Equal(t, 9, got[hLower])
}

func TestCheckNTLMDedupesPrefixes(t *testing.T) {
	// Two different hashes under one prefix → one request.
	h1 := "ABCDE1111111111111111111111111111"[:32]
	h2 := "ABCDE2222222222222222222222222222"[:32]
	f := &fakeDoer{byPrefix: map[string]string{"ABCDE": ""}}
	_, err := CheckNTLM(context.Background(), f, []string{h1, h2})
	require.NoError(t, err)
	require.Len(t, f.reqs, 1)
}

func TestCheckNTLMSkipsNonHashes(t *testing.T) {
	f := &fakeDoer{byPrefix: map[string]string{}}
	got, err := CheckNTLM(context.Background(), f, []string{"", "notahash", "zzzz"})
	require.NoError(t, err)
	require.Empty(t, got)
	require.Empty(t, f.reqs)
}

func TestCheckNTLMPropagatesHTTPError(t *testing.T) {
	f := &fakeDoer{status: 500, byPrefix: map[string]string{"32ED8": ""}}
	_, err := CheckNTLM(context.Background(), f, []string{"32ED87BDB5FDC5E9CBA88547376818D4"})
	require.Error(t, err)
}
