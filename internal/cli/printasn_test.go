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
		ASN:     "AS13335",
		Holder:  "CLOUDFLARENET, US",
		Peering: &lookup.Peering{
			Type:    "Content",
			Policy:  "Open",
			Website: "https://www.cloudflare.com",
			IXCount: 999, // deliberately disagrees with the list; display must use the list
			IXPs: []lookup.PeerIX{
				{Name: "AMS-IX"}, {Name: "DE-CIX"}, {Name: "LINX"},
				{Name: "Equinix IX"}, {Name: "France-IX"},
			},
			Facilities: []lookup.PeerFac{{Name: "Equinix DC1", City: "Ashburn", Country: "US"}},
		},
	})
	s := out.String()
	require.Contains(t, s, "Peering")
	require.Contains(t, s, "Content")
	require.Contains(t, s, "Open")
	require.Contains(t, s, "https://www.cloudflare.com")
	// Count is derived from the fetched list, not net's ix_count, and the
	// "+N more" is consistent with that same count.
	require.Contains(t, s, "5 IXPs")
	require.Contains(t, s, "AMS-IX")
	require.Contains(t, s, "+2 more")
	require.NotContains(t, s, "999")
	require.Contains(t, s, "1 facility")
}

func TestPrintIPReportPeeringLine(t *testing.T) {
	var out bytes.Buffer
	printIPReport(&out, lookup.IPReport{
		Input:     "104.16.123.45",
		Version:   "IPv4",
		OriginASN: "AS13335",
		Peering: &lookup.Peering{
			Policy:     "Open",
			IXPs:       []lookup.PeerIX{{Name: "AMS-IX"}, {Name: "DE-CIX"}, {Name: "LINX"}},
			Facilities: []lookup.PeerFac{{Name: "Equinix DC1"}, {Name: "Equinix DC2"}},
		},
	})
	s := out.String()
	require.Contains(t, s, "Peering")
	require.Contains(t, s, "3 IXPs")
	require.Contains(t, s, "2 facilities")
	require.Contains(t, s, "Open")
}

func TestPrintIPReportNoPeeringLine(t *testing.T) {
	var out bytes.Buffer
	printIPReport(&out, lookup.IPReport{Input: "192.0.2.1", Version: "IPv4", RoutingNote: "reserved"})
	require.NotContains(t, out.String(), "Peering")
}

func TestPrintASNReportNoPeering(t *testing.T) {
	var out bytes.Buffer
	printASNReport(&out, lookup.ASNReport{ASN: "AS65000", Holder: "Example"})
	require.NotContains(t, out.String(), "Peering")
}

func TestPrintASNReportPeeringNoPresence(t *testing.T) {
	// A network in PeeringDB but present at no IXPs/facilities: the summary
	// still prints, but empty presence rows are suppressed (no "0 IXPs").
	var out bytes.Buffer
	printASNReport(&out, lookup.ASNReport{
		ASN:     "AS65001",
		Peering: &lookup.Peering{Type: "NSP", Policy: "Selective"},
	})
	s := out.String()
	require.Contains(t, s, "Peering")
	require.Contains(t, s, "Selective")
	require.NotContains(t, s, "IXP")
	require.NotContains(t, s, "facilit")
}
