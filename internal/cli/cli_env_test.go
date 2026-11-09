package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/hriday/meridian/internal/dnsx"
	"github.com/hriday/meridian/internal/ntp"
	"github.com/stretchr/testify/require"
)

func TestRunNTPJSON(t *testing.T) {
	old := ntpQuery
	t.Cleanup(func() { ntpQuery = old })
	ntpQuery = func(_ context.Context, server string) (ntp.Reply, error) {
		return ntp.Reply{Server: server, Stratum: 2, RefID: "1.2.3.4"}, nil
	}
	var out, errb bytes.Buffer
	code := Run([]string{"ntp", "time.apple.com", "--json"}, &out, &errb)
	require.Equal(t, 0, code, errb.String())
	require.Contains(t, out.String(), `"stratum":2`)
}

func TestRunDNSJSON(t *testing.T) {
	old := dnsQuery
	t.Cleanup(func() { dnsQuery = old })
	dnsQuery = func(_ context.Context, name, rtype, server string) (dnsx.Result, error) {
		return dnsx.Result{Rcode: "NOERROR", Answers: []dnsx.Record{{Name: name, Type: rtype, Value: "104.16.132.229"}}}, nil
	}
	var out, errb bytes.Buffer
	code := Run([]string{"dns", "cloudflare.com", "--type", "A", "--server", "1.1.1.1", "--json"}, &out, &errb)
	require.Equal(t, 0, code, errb.String())
	require.Contains(t, out.String(), "104.16.132.229")
	require.Contains(t, out.String(), `"rcode":"NOERROR"`)
}
