package threatintel

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeDoer struct {
	body   string
	status int
	req    *http.Request
}

func (f *fakeDoer) Do(req *http.Request) (*http.Response, error) {
	f.req = req
	st := f.status
	if st == 0 {
		st = 200
	}
	return &http.Response{StatusCode: st, Body: io.NopCloser(strings.NewReader(f.body)), Header: make(http.Header)}, nil
}

// A trimmed-down copy of the abuse.ch Feodo Tracker ipblocklist.json shape.
const feodo = `[
  {"ip_address":"162.243.103.246","port":8080,"status":"offline","hostname":null,"as_number":14061,"as_name":"DIGITALOCEAN-ASN","country":"US","first_seen":"2022-06-04 21:24:53","last_online":"2026-03-07","malware":"Emotet"},
  {"ip_address":"91.240.118.168","port":443,"status":"online","hostname":"bad.example","as_number":51381,"as_name":"EVIL-AS","country":"RU","first_seen":"2024-01-02 00:00:00","last_online":"2026-11-10","malware":"Dridex"}
]`

func TestCheckFlagsKnownC2(t *testing.T) {
	f := &fakeDoer{body: feodo}
	hits, err := Check(context.Background(), f, []string{"91.240.118.168", "8.8.8.8"})
	require.NoError(t, err)
	require.Len(t, hits, 2, "one hit per input IP, in order")

	require.Equal(t, "91.240.118.168", hits[0].IP)
	require.True(t, hits[0].Listed, "known C2 flagged")
	require.Equal(t, "Dridex", hits[0].Malware)
	require.Equal(t, "online", hits[0].Status)
	require.Equal(t, "RU", hits[0].Country)
	require.Equal(t, "EVIL-AS", hits[0].ASName)
	require.Equal(t, 443, hits[0].Port)

	require.Equal(t, "8.8.8.8", hits[1].IP)
	require.False(t, hits[1].Listed, "clean IP not flagged")
	require.Empty(t, hits[1].Malware)
}

func TestCheckDedupesInput(t *testing.T) {
	f := &fakeDoer{body: feodo}
	hits, err := Check(context.Background(), f, []string{"8.8.8.8", "8.8.8.8", "  8.8.8.8  "})
	require.NoError(t, err)
	require.Len(t, hits, 1, "duplicate inputs collapse to one row")
}

func TestCheckNoIPs(t *testing.T) {
	f := &fakeDoer{body: feodo}
	_, err := Check(context.Background(), f, []string{"  ", "not-an-ip"})
	require.Error(t, err, "nothing valid to check")
	require.Nil(t, f.req, "no feed fetched when there's nothing to look up")
}

func TestCheckFetchError(t *testing.T) {
	f := &fakeDoer{status: 503, body: ""}
	_, err := Check(context.Background(), f, []string{"8.8.8.8"})
	require.Error(t, err)
}

func TestCheckBadJSON(t *testing.T) {
	f := &fakeDoer{body: "<html>nope</html>"}
	_, err := Check(context.Background(), f, []string{"8.8.8.8"})
	require.Error(t, err)
}

func TestExtractIPs(t *testing.T) {
	got := ExtractIPs("91.240.118.168, 8.8.8.8\n10.0.0.1\nnonsense 2001:db8::1")
	require.Equal(t, []string{"91.240.118.168", "8.8.8.8", "10.0.0.1", "2001:db8::1"}, got)
}
