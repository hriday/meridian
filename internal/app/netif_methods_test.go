package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInterfacesMethod(t *testing.T) {
	ifaces, err := New().Interfaces()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(ifaces), 1) // loopback always exists
}
