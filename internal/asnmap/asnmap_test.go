package asnmap

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/dnsx"
	"github.com/stretchr/testify/require"
)

func txtResult(v string) dnsx.Result {
	return dnsx.Result{Rcode: "NOERROR", Answers: []dnsx.Record{{Type: "TXT", Value: v}}}
}

func TestParseOriginTXT(t *testing.T) {
	asn, prefix := parseOriginTXT(`"13335 | 104.16.0.0/13 | US | arin | 2010-07-14"`)
	require.Equal(t, "13335", asn)
	require.Equal(t, "104.16.0.0/13", prefix)
}

func TestParseASNameTXT(t *testing.T) {
	require.Equal(t, "CLOUDFLARENET, US", parseASNameTXT(`"13335 | US | arin | 2010-07-14 | CLOUDFLARENET, US"`))
}

func TestLookupFull(t *testing.T) {
	q := func(_ context.Context, name, _, _ string) (dnsx.Result, error) {
		if strings.Contains(name, "origin") {
			return txtResult(`"13335 | 104.16.0.0/13 | US | arin"`), nil
		}
		return txtResult(`"13335 | US | arin | 2010-07-14 | CLOUDFLARENET, US"`), nil
	}
	info, err := Lookup(context.Background(), q, "104.16.123.45")
	require.NoError(t, err)
	require.Equal(t, "13335", info.ASN)
	require.Equal(t, "CLOUDFLARENET, US", info.Owner)
}

func TestLookupSkipsPrivate(t *testing.T) {
	q := func(_ context.Context, _, _, _ string) (dnsx.Result, error) {
		t.Fatal("should not query DNS for a private address")
		return dnsx.Result{}, nil
	}
	info, err := Lookup(context.Background(), q, "10.1.2.3")
	require.NoError(t, err)
	require.Empty(t, info.ASN)
}

func TestLookupPartialOnError(t *testing.T) {
	q := func(_ context.Context, name, _, _ string) (dnsx.Result, error) {
		if strings.Contains(name, "origin") {
			return txtResult(`"13335 | 104.16.0.0/13 | US | arin"`), nil
		}
		return dnsx.Result{}, fmt.Errorf("boom")
	}
	info, err := Lookup(context.Background(), q, "104.16.123.45")
	require.NoError(t, err)
	require.Equal(t, "13335", info.ASN)
	require.Empty(t, info.Owner)
}

func TestLookupV6UsesOrigin6(t *testing.T) {
	var sawName string
	q := func(_ context.Context, name, _, _ string) (dnsx.Result, error) {
		if strings.Contains(name, "origin") {
			sawName = name
			return txtResult(`"13335 | 2606:4700::/32 | US | arin"`), nil
		}
		return txtResult(`"13335 | US | arin | x | CLOUDFLARENET, US"`), nil
	}
	info, err := Lookup(context.Background(), q, "2606:4700::6810:84e5")
	require.NoError(t, err)
	require.Equal(t, "13335", info.ASN)
	require.Contains(t, sawName, "origin6.asn.cymru.com")
}
