package hibp

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type jsonDoer struct {
	body   string
	status int
	req    *http.Request
}

func (d *jsonDoer) Do(req *http.Request) (*http.Response, error) {
	d.req = req
	st := d.status
	if st == 0 {
		st = 200
	}
	return &http.Response{StatusCode: st, Body: io.NopCloser(strings.NewReader(d.body)), Header: make(http.Header)}, nil
}

func TestBreachesParsesAndStripsHTML(t *testing.T) {
	d := &jsonDoer{body: `[
		{"Name":"Adobe","Title":"Adobe","Domain":"adobe.com","BreachDate":"2013-10-04","PwnCount":152445165,
		 "Description":"In <a href=\"x\">October 2013</a>, 153 million accounts &amp; more.","DataClasses":["Email addresses","Passwords"],
		 "IsVerified":true,"IsSensitive":false,"IsMalware":false},
		{"Name":"Ashley","Title":"Ashley Madison","Domain":"ashleymadison.com","BreachDate":"2015-07-19","PwnCount":30811934,
		 "Description":"Plain","DataClasses":["Email addresses"],"IsVerified":true,"IsSensitive":true,"IsMalware":false}]`}

	got, err := Breaches(context.Background(), d)
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, "Adobe", got[0].Title)
	require.Equal(t, "adobe.com", got[0].Domain)
	require.Equal(t, 152445165, got[0].PwnCount)
	require.ElementsMatch(t, []string{"Email addresses", "Passwords"}, got[0].DataClasses)
	require.Equal(t, "In October 2013, 153 million accounts & more.", got[0].Description, "HTML stripped and entities decoded")
	require.True(t, got[1].Sensitive)
	// Required User-Agent header.
	require.NotEmpty(t, d.req.Header.Get("User-Agent"))
}

func TestLatestBreach(t *testing.T) {
	d := &jsonDoer{body: `{"Name":"Recent","Title":"Recent Co","Domain":"recent.co","BreachDate":"2025-01-01","PwnCount":1000,"Description":"<b>Big</b>","DataClasses":["Email addresses"],"IsVerified":true}`}
	b, err := LatestBreach(context.Background(), d)
	require.NoError(t, err)
	require.Equal(t, "Recent Co", b.Title)
	require.Equal(t, "Big", b.Description)
}

func TestBreachesHTTPError(t *testing.T) {
	d := &jsonDoer{status: 403, body: ""}
	_, err := Breaches(context.Background(), d)
	require.Error(t, err)
}

func TestStripHTML(t *testing.T) {
	require.Equal(t, "hello world", stripHTML("<p>hello   <b>world</b></p>"))
	require.Equal(t, "a & b", stripHTML("a &amp; b"))
}
