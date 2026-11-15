package wifi

import "testing"

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
