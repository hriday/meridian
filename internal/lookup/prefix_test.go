package lookup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRipePrefixOverview(t *testing.T) {
	c := ripeServer(t, map[string]string{
		"prefix-overview": `{"data":{"resource":"104.16.0.0/13","announced":true,"asns":[{"asn":13335,"holder":"CLOUDFLARENET, US"}]}}`,
	})
	origins, holder, announced, err := c.ripePrefixOverview(context.Background(), "104.16.0.0/13")
	require.NoError(t, err)
	require.Equal(t, []string{"13335"}, origins)
	require.Equal(t, "CLOUDFLARENET, US", holder)
	require.True(t, announced)
}

func TestRipeRelatedPrefixes(t *testing.T) {
	c := ripeServer(t, map[string]string{
		"related-prefixes": `{"data":{"prefixes":[{"prefix":"104.16.0.0/12","relationship":"less-specific"},{"prefix":"104.16.123.0/24","relationship":"more-specific"}]}}`,
	})
	less, more, err := c.ripeRelatedPrefixes(context.Background(), "104.16.0.0/13")
	require.NoError(t, err)
	require.Equal(t, []string{"104.16.0.0/12"}, less)
	require.Equal(t, []string{"104.16.123.0/24"}, more)
}

func TestRipeRPKIStates(t *testing.T) {
	cv := ripeServer(t, map[string]string{
		"rpki-validation": `{"data":{"status":"valid","validating_roas":[{"prefix":"104.16.0.0/13","max_length":24,"origin":"13335"}]}}`,
	})
	state, roa, err := cv.ripeRPKI(context.Background(), "104.16.0.0/13", "AS13335")
	require.NoError(t, err)
	require.Equal(t, "valid", state)
	require.Contains(t, roa, "104.16.0.0/13")

	ci := ripeServer(t, map[string]string{
		"rpki-validation": `{"data":{"status":"invalid","validating_roas":[]}}`,
	})
	state, _, err = ci.ripeRPKI(context.Background(), "104.16.0.0/24", "AS99999")
	require.NoError(t, err)
	require.Equal(t, "invalid", state)
}
