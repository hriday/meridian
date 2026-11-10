package app

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTLSCheckMethod(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	t.Cleanup(srv.Close)
	// srv.URL is https://127.0.0.1:<port>; TLSCheck takes host[:port].
	host := srv.URL[len("https://"):]
	_, port, err := net.SplitHostPort(host)
	require.NoError(t, err)

	rep, err := New().TLSCheck("127.0.0.1:" + port)
	require.NoError(t, err)
	require.NotEmpty(t, rep.Verdict)
	require.NotEmpty(t, rep.Subject)
}
