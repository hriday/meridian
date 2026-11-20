package wifi

import "testing"

// When an in-process CoreWLAN scan returns real neighbor SSIDs, they replace
// the redacted system_profiler list and co-channel is recomputed against the
// connected channel.
func TestMergeScan_ReplacesNeighborsAndRecomputesCoChannel(t *testing.T) {
	info := Info{
		Current:   Network{SSID: "home", Channel: 149},
		Neighbors: []Network{{SSID: "<redacted>", Channel: 11}, {SSID: "<redacted>", Channel: 149}},
	}
	scanned := []Network{
		{SSID: "neighbor-a", Channel: 11, Band: "2GHz", RSSI: -60},
		{SSID: "neighbor-b", Channel: 149, Band: "5GHz", RSSI: -42},
		{SSID: "neighbor-c", Channel: 149, Band: "5GHz", RSSI: -50},
	}
	mergeScan(&info, scanned)
	if len(info.Neighbors) != 3 || info.Neighbors[0].SSID != "neighbor-a" {
		t.Fatalf("scan SSIDs did not replace redacted list: %+v", info.Neighbors)
	}
	if info.CoChannel != 2 {
		t.Errorf("want 2 co-channel neighbors on ch 149, got %d", info.CoChannel)
	}
	if info.Neighbors[0].CoChannel || !info.Neighbors[1].CoChannel || !info.Neighbors[2].CoChannel {
		t.Errorf("co-channel flags wrong: %+v", info.Neighbors)
	}
}

// An empty scan (scan unsupported or unauthorized) keeps the existing
// neighbors untouched, but co-channel is still recomputed.
func TestMergeScan_EmptyKeepsExistingNeighbors(t *testing.T) {
	info := Info{
		Current:   Network{Channel: 6},
		Neighbors: []Network{{SSID: "<redacted>", Channel: 6}},
	}
	mergeScan(&info, nil)
	if len(info.Neighbors) != 1 || info.Neighbors[0].SSID != "<redacted>" {
		t.Fatalf("empty scan must keep existing neighbors: %+v", info.Neighbors)
	}
	if info.CoChannel != 1 || !info.Neighbors[0].CoChannel {
		t.Errorf("co-channel not recomputed on kept neighbors: %+v", info)
	}
}

// Trimmed from real `system_profiler SPAirPortDataType` on macOS 26 (Tahoe).
const sample = `Wi-Fi:

      Software Versions:
          CoreWLAN: 16.0 (1657)
      Interfaces:
        en0:
          Card Type: Wi-Fi  (0x14E4, 0x4387)
          MAC Address: be:b6:72:3b:b0:ad
          Country Code: CA
          Status: Connected
          Current Network Information:
            <redacted>:
              PHY Mode: 802.11ax
              Channel: 149 (5GHz, 40MHz)
              Country Code: CA
              Network Type: Infrastructure
              Security: WPA2 Personal
              Signal / Noise: -48 dBm / -94 dBm
              Transmit Rate: 573
              MCS Index: 11
          Other Local Wi-Fi Networks:
            <redacted>:
              PHY Mode: 802.11a/n/ac/ax
              Channel: 157 (5GHz, 80MHz)
              Security: WPA2 Personal
              Signal / Noise: -74 dBm / -96 dBm
            <redacted>:
              PHY Mode: 802.11a/n/ac/ax
              Channel: 149 (5GHz, 40MHz)
              Security: WPA2 Personal
              Signal / Noise: -43 dBm / -96 dBm
            <redacted>:
              PHY Mode: 802.11b/g/n
              Channel: 11 (2GHz, 20MHz)
              Security: WPA2 Personal
              Signal / Noise: -33 dBm / -91 dBm
        awdl0:
          MAC Address: c6:88:d2:cb:e5:22
          Current Network Information:
              Network Type: Infrastructure
`

func TestParseAirport(t *testing.T) {
	info := parseAirport(sample)

	if info.Interface != "en0" || !info.Connected {
		t.Fatalf("interface/connected wrong: %q %v", info.Interface, info.Connected)
	}
	c := info.Current
	if c.Channel != 149 || c.Band != "5GHz" || c.Width != "40MHz" {
		t.Fatalf("current channel/band/width: %+v", c)
	}
	if c.RSSI != -48 || c.Noise != -94 || c.SNR != 46 {
		t.Fatalf("current signal/noise/snr: rssi=%d noise=%d snr=%d", c.RSSI, c.Noise, c.SNR)
	}
	if c.PHYMode != "802.11ax" || c.Security != "WPA2 Personal" || c.TxRate != 573 {
		t.Fatalf("current phy/security/tx: %+v", c)
	}
	if len(info.Neighbors) != 3 {
		t.Fatalf("want 3 neighbors (awdl0 excluded), got %d", len(info.Neighbors))
	}
	// One neighbor is co-channel (149).
	if info.CoChannel != 1 {
		t.Fatalf("want 1 co-channel neighbor, got %d", info.CoChannel)
	}
	if !info.SSIDRedacted {
		t.Fatal("expected SSIDRedacted true")
	}
}
