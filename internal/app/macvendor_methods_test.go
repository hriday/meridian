package app

import (
	"context"
	"testing"

	"github.com/hriday/meridian/internal/macvendor"
	"github.com/stretchr/testify/require"
)

func TestMACVendorMethod(t *testing.T) {
	old := macLookup
	macLookup = func(_ context.Context, _ macvendor.Doer, mac string) (macvendor.Result, error) {
		return macvendor.Result{Input: mac, OUI: "FC:FB:FB", Vendor: "Apple, Inc.", Found: true}, nil
	}
	t.Cleanup(func() { macLookup = old })

	r, err := New().MACVendor("fc:fb:fb")
	require.NoError(t, err)
	require.True(t, r.Found)
	require.Equal(t, "Apple, Inc.", r.Vendor)
	require.Equal(t, "FC:FB:FB", r.OUI)
}
