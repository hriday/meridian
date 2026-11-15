package wifi

import "testing"

// Realistic terse nmcli output (BSSID colons escaped as \:, one active network).
const nmcliOut = `yes:HomeNet:AA\:BB\:CC\:DD\:EE\:FF:36:5180 MHz:78:WPA2
no:HomeNet-Guest:AA\:BB\:CC\:DD\:EE\:01:36:5180 MHz:60:WPA2
no:Neighbor:11\:22\:33\:44\:55\:66:6:2437 MHz:45:WPA1 WPA2
no::--:1:2412 MHz:20:
`

func TestParseNmcli(t *testing.T) {
	info := parseNmcli(nmcliOut)
	if !info.Connected || info.Current.SSID != "HomeNet" {
		t.Fatalf("current = %q connected=%v", info.Current.SSID, info.Connected)
	}
	if info.Current.Channel != 36 || info.Current.Band != "5GHz" {
		t.Errorf("chan/band = %d/%s", info.Current.Channel, info.Current.Band)
	}
	// SIGNAL 78 → ~ -61 dBm.
	if info.Current.RSSI != 78/2-100 {
		t.Errorf("rssi = %d", info.Current.RSSI)
	}
	if len(info.Neighbors) != 3 {
		t.Fatalf("neighbors = %d, want 3", len(info.Neighbors))
	}
	// The guest AP shares channel 36 with the connected network → co-channel.
	var guest *Network
	for i := range info.Neighbors {
		if info.Neighbors[i].SSID == "HomeNet-Guest" {
			guest = &info.Neighbors[i]
		}
	}
	if guest == nil || !guest.CoChannel {
		t.Error("HomeNet-Guest should be flagged co-channel on ch 36")
	}
	if info.CoChannel != 1 {
		t.Errorf("co-channel count = %d, want 1", info.CoChannel)
	}
	if info.LocationStatus != "unsupported" {
		t.Errorf("locationStatus = %q", info.LocationStatus)
	}
}

func TestSplitNmcliUnescapesColons(t *testing.T) {
	f := splitNmcli(`yes:Net:AA\:BB\:CC:6:2437 MHz:50:WPA2`)
	if len(f) != 7 {
		t.Fatalf("fields = %d, want 7", len(f))
	}
	if f[2] != "AA:BB:CC" {
		t.Errorf("bssid = %q, want AA:BB:CC", f[2])
	}
}

// Realistic netsh output (English locale).
const netshIface = `
There is 1 interface on the system:

    Name                   : Wi-Fi
    State                  : connected
    SSID                   : HomeNet
    BSSID                  : aa:bb:cc:dd:ee:ff
    Signal                 : 82%
    Channel                : 36
`

const netshNetworks = `
Interface name : Wi-Fi
There are 2 networks currently visible.

SSID 1 : HomeNet
    Authentication          : WPA2-Personal
    Encryption              : CCMP
    BSSID 1                 : aa:bb:cc:dd:ee:ff
         Signal             : 82%
         Radio type         : 802.11ax
         Channel            : 36

SSID 2 : Neighbor
    Authentication          : WPA2-Personal
    Encryption              : CCMP
    BSSID 1                 : 11:22:33:44:55:66
         Signal             : 40%
         Radio type         : 802.11n
         Channel            : 6
`

func TestParseNetsh(t *testing.T) {
	info := parseNetsh(netshIface, netshNetworks)
	if !info.Connected || info.Current.SSID != "HomeNet" {
		t.Fatalf("current = %q connected=%v", info.Current.SSID, info.Connected)
	}
	if info.Current.Channel != 36 || info.Current.Band != "5GHz" {
		t.Errorf("chan/band = %d/%s", info.Current.Channel, info.Current.Band)
	}
	if info.Current.Security != "WPA2-Personal" {
		t.Errorf("security = %q", info.Current.Security)
	}
	if info.Current.PHYMode != "802.11ax" {
		t.Errorf("phy = %q", info.Current.PHYMode)
	}
	if info.Current.RSSI != 82/2-100 {
		t.Errorf("rssi = %d", info.Current.RSSI)
	}
	if len(info.Neighbors) != 1 || info.Neighbors[0].SSID != "Neighbor" {
		t.Fatalf("neighbors = %+v", info.Neighbors)
	}
	if info.Neighbors[0].Channel != 6 || info.Neighbors[0].Band != "2GHz" {
		t.Errorf("neighbor chan/band = %d/%s", info.Neighbors[0].Channel, info.Neighbors[0].Band)
	}
}
