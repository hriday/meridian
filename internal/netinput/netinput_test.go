package netinput_test

import (
	"testing"

	"github.com/hriday/meridian/internal/netinput"
	"github.com/stretchr/testify/require"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		in   string
		kind netinput.Kind
		norm string
	}{
		{"1.1.1.1", netinput.KindIPv4, "1.1.1.1"},
		{"  1.1.1.1 ", netinput.KindIPv4, "1.1.1.1"},
		{"2606:4700:4700::1111", netinput.KindIPv6, "2606:4700:4700::1111"},
		{"1.1.1.0/24", netinput.KindPrefixV4, "1.1.1.0/24"},
		{"2606:4700::/32", netinput.KindPrefixV6, "2606:4700::/32"},
		{"AS13335", netinput.KindASN, "AS13335"},
		{"as13335", netinput.KindASN, "AS13335"},
		{"13335", netinput.KindASN, "AS13335"},
		{"cloudflare.com", netinput.KindDomain, "cloudflare.com"},
		{"CloudFlare.com.", netinput.KindDomain, "cloudflare.com"},
		{"router1", netinput.KindHostname, "router1"},
	}
	for _, c := range cases {
		got, err := netinput.Classify(c.in)
		require.NoError(t, err, c.in)
		require.Equal(t, c.kind, got.Kind, c.in)
		require.Equal(t, c.norm, got.Normalized, c.in)
	}
}

func TestClassifyMalformed(t *testing.T) {
	for _, in := range []string{"", "   ", "...", "1.1.1.1/33", "AS", "99999999999999999999", "has space"} {
		_, err := netinput.Classify(in)
		require.Error(t, err, in)
	}
}
