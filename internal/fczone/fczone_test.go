package fczone

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// ciscoActive is the verbatim `show zoneset active` output from a real MDS9509
// (HP C-Series compatibility manual, SAN-OS 3.2), re-indented (the parser is
// whitespace-tolerant). Every member here is a logged-in pwwn.
const ciscoActive = `MDS9509# show zoneset active
zoneset name mdscore1 vsan 1
  zone name vz11 vsan 1
  * fcid 0x610400 [pwwn 10:00:00:00:c9:24:3d:90]
  * fcid 0x630500 [pwwn 50:06:01:60:88:02:90:cb]
  zone name vz22 vsan 1
  * fcid 0x6514e2 [pwwn 21:00:00:20:37:a7:ca:b7]
  zone name vz33 vsan 1
  * fcid 0x6105dc [pwwn 21:00:00:20:37:28:31:6d]
  zone name $default_zone$ vsan 1`

func TestCiscoActiveZoneset(t *testing.T) {
	r := Parse(ciscoActive)
	require.Equal(t, "Cisco MDS", r.Vendor)
	require.Len(t, r.Zonesets, 1)
	require.True(t, r.Zonesets[0].Active)
	require.Equal(t, "mdscore1", r.Zonesets[0].Name)

	// vz11 has two logged-in pwwn members (a proper initiator+target pair).
	vz11 := findZone(t, r, "vz11")
	require.Len(t, vz11.Members, 2)
	require.Equal(t, "pwwn", vz11.Members[0].Kind)
	require.Equal(t, "10:00:00:00:c9:24:3d:90", vz11.Members[0].Value)
	require.True(t, vz11.Members[0].Active)

	// vz22 and vz33 are single-member zones → flagged.
	require.True(t, hasFinding(r, "Single-member zone", "vz22"))
	require.True(t, hasFinding(r, "Single-member zone", "vz33"))
	// The default zone is ignored (not flagged as empty).
	require.False(t, hasFinding(r, "Empty zone", "$default_zone$"))
}

// ciscoRunning is running-config-style zoning with fcalias indirection.
const ciscoRunning = `zoneset name prodzs vsan 10
  member zoneA
fcalias name HostA vsan 10
  member pwwn 10:00:00:00:c9:11:22:33
zone name zoneA vsan 10
  member fcalias HostA
  member pwwn 50:06:01:60:88:02:90:cb
zoneset activate name prodzs vsan 10`

func TestCiscoRunningConfigWithAlias(t *testing.T) {
	r := Parse(ciscoRunning)
	require.Equal(t, "Cisco MDS", r.Vendor)
	require.Len(t, r.Aliases, 1)
	require.Equal(t, "HostA", r.Aliases[0].Name)
	require.True(t, r.Zonesets[0].Active, "zoneset activate marks it active")

	zoneA := findZone(t, r, "zoneA")
	require.Len(t, zoneA.Members, 2)
	require.Equal(t, "fcalias", zoneA.Members[0].Kind)
	require.Equal(t, "HostA", zoneA.Members[0].Value)
	// HostA is referenced, so it is NOT reported unused.
	require.False(t, hasFinding(r, "Unused alias", "HostA"))
}

// brocade is the verbatim cfgshow sample from the Prefetch blog.
const brocade = `Defined configuration:
 cfg: SANFabricOne CentOSNode1Zone1; CentOSNode1Zone2
 zone: CentOSNode1Zone1 CentOSNode1Port1; NevadaPort1
 zone: CentOSNode1Zone2 a; b; c
 alias: CentOSNode1Port1 21:00:00:1b:32:04:86:c3
 alias: UnusedAlias 10:00:00:00:c9:00:00:01
Effective configuration:
 cfg: SANFabricOne
 zone: CentOSNode1Zone1 21:00:00:1b:32:04:86:c3 10:00:00:00:c9:3e:4c:eb`

func TestBrocadeCfgshow(t *testing.T) {
	r := Parse(brocade)
	require.Equal(t, "Brocade FOS", r.Vendor)
	require.Len(t, r.Zonesets, 1)
	require.True(t, r.Zonesets[0].Active, "the effective cfg is active")
	require.Equal(t, []string{"CentOSNode1Zone1", "CentOSNode1Zone2"}, r.Zonesets[0].Zones)

	// Zone1 keeps the defined-section members (alias names), not the effective WWNs.
	z1 := findZone(t, r, "CentOSNode1Zone1")
	require.Len(t, z1.Members, 2)
	require.Equal(t, "CentOSNode1Port1", z1.Members[0].Value)

	// Zone2 has three members → flagged as not single-initiator-single-target.
	require.True(t, hasFinding(r, "Zone isn't single-initiator-single-target", "CentOSNode1Zone2"))
	// UnusedAlias is never referenced by a zone.
	require.True(t, hasFinding(r, "Unused alias", "UnusedAlias"))
	// CentOSNode1Port1 IS referenced, so not flagged.
	require.False(t, hasFinding(r, "Unused alias", "CentOSNode1Port1"))
}

func TestWWNMemberParsing(t *testing.T) {
	r := Parse(brocade)
	// Members that look like WWNs are typed as wwn and lowercased.
	for _, a := range r.Aliases {
		if a.Name == "CentOSNode1Port1" {
			require.Equal(t, "wwn", a.Members[0].Kind)
			require.Equal(t, "21:00:00:1b:32:04:86:c3", a.Members[0].Value)
		}
	}
}

func findZone(t *testing.T, r Report, name string) Zone {
	t.Helper()
	for _, z := range r.Zones {
		if z.Name == name {
			return z
		}
	}
	t.Fatalf("zone %q not found in %+v", name, r.Zones)
	return Zone{}
}

func hasFinding(r Report, title, object string) bool {
	for _, f := range r.Findings {
		if f.Title == title && f.Object == object {
			return true
		}
	}
	return false
}
