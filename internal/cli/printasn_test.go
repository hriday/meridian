package cli

import (
	"bytes"
	"testing"

	"github.com/hriday/meridian/internal/lookup"
	"github.com/stretchr/testify/require"
)

func TestPrintASNReportPeering(t *testing.T) {
	var out bytes.Buffer
	printASNReport(&out, lookup.ASNReport{
		ASN:    "AS13335",
		Holder: "CLOUDFLARENET, US",
		Peering: &lookup.Peering{
			Type:     "Content",
			Policy:   "Open",
			IXCount:  418,
			FacCount: 224,
			IXPs:     []lookup.PeerIX{{Name: "AMS-IX", Speed: 400000}},
			Facilities: []lookup.PeerFac{
				{Name: "Equinix DC1", City: "Ashburn", Country: "US"},
			},
		},
	})
	s := out.String()
	require.Contains(t, s, "Peering")
	require.Contains(t, s, "Content")
	require.Contains(t, s, "Open")
	require.Contains(t, s, "418") // total IX presences
	require.Contains(t, s, "224") // total facilities
	require.Contains(t, s, "AMS-IX")
}

func TestPrintASNReportNoPeering(t *testing.T) {
	var out bytes.Buffer
	printASNReport(&out, lookup.ASNReport{ASN: "AS65000", Holder: "Example"})
	require.NotContains(t, out.String(), "Peering")
}
