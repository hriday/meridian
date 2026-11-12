package xon

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
	err    error
}

func (f *fakeDoer) Do(req *http.Request) (*http.Response, error) {
	f.req = req
	if f.err != nil {
		return nil, f.err
	}
	st := f.status
	if st == 0 {
		st = 200
	}
	return &http.Response{StatusCode: st, Body: io.NopCloser(strings.NewReader(f.body)), Header: make(http.Header)}, nil
}

func TestCheckEmailFound(t *testing.T) {
	f := &fakeDoer{body: `{"breaches":[["Adobe","Dropbox","Adobe","LinkedIn"]]}`}
	got, err := CheckEmail(context.Background(), f, "jane@example.com")
	require.NoError(t, err)
	require.Equal(t, []string{"Adobe", "Dropbox", "LinkedIn"}, got, "deduped, order preserved")
	require.NotEmpty(t, f.req.Header.Get("User-Agent"))
	require.Contains(t, f.req.URL.Path, "jane@example.com")
}

func TestCheckEmailNotFound(t *testing.T) {
	f := &fakeDoer{status: 404, body: `{"Error":"Not found","email":null}`}
	got, err := CheckEmail(context.Background(), f, "nobody@nowhere.invalid")
	require.NoError(t, err)
	require.Empty(t, got, "a clean email is an empty result, not an error")
}

func TestCheckEmailEmptyBreaches(t *testing.T) {
	f := &fakeDoer{body: `{"breaches":[[]]}`}
	got, err := CheckEmail(context.Background(), f, "x@y.com")
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestCheckEmailValidation(t *testing.T) {
	f := &fakeDoer{}
	_, err := CheckEmail(context.Background(), f, "   ")
	require.Error(t, err)
	_, err = CheckEmail(context.Background(), f, "notanemail")
	require.Error(t, err)
	require.Nil(t, f.req, "no request is made for an invalid address")
}

func TestCheckEmailRateLimited(t *testing.T) {
	f := &fakeDoer{status: 429, body: ""}
	_, err := CheckEmail(context.Background(), f, "x@y.com")
	require.Error(t, err)
	require.Contains(t, err.Error(), "rate-limited")
}
