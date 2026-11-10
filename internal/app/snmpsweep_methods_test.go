package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSnmpSweepInvalidCIDR(t *testing.T) {
	_, err := New().SnmpSweep("nonsense", "public")
	require.Error(t, err)
	require.NotContains(t, err.Error(), "public")
}
