package app

import (
	"testing"

	"github.com/hriday/engineers_tools/internal/netinput"
	"github.com/stretchr/testify/require"
)

func TestAppSubnetDescribeDelegates(t *testing.T) {
	a := New()
	info, err := a.SubnetDescribe("10.0.0.0/24")
	require.NoError(t, err)
	require.Equal(t, "10.0.0.255", info.Broadcast)
}

func TestAppClassifyDelegates(t *testing.T) {
	a := New()
	r, err := a.Classify("AS13335")
	require.NoError(t, err)
	require.Equal(t, netinput.KindASN, r.Kind)
}

func TestAppSubnetSplitDelegates(t *testing.T) {
	a := New()
	out, err := a.SubnetSplit("10.0.0.0/24", 26)
	require.NoError(t, err)
	require.Len(t, out, 4)
}
