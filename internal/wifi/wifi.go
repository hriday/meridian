// Package wifi reports the local Wi-Fi/RF environment: the connected network's
// channel, band, width, signal/noise (SNR), PHY mode and security, plus the
// neighboring APs and which of them share your channel (co-channel contention).
//
// On macOS it parses `system_profiler SPAirPortDataType` (no cgo; the old
// `airport` CLI is gone on recent macOS). SSIDs are redacted by macOS unless the
// app has Location Services — the RF metrics are unaffected. This file holds the
// pure parser so the whole decode is testable without Wi-Fi hardware.
package wifi

import (
	"strconv"
	"strings"
)

// Network is one Wi-Fi network (the connected one, or a neighbor).
type Network struct {
	SSID      string `json:"ssid"`
	Channel   int    `json:"channel"`
	Band      string `json:"band"`  // 2GHz / 5GHz / 6GHz
	Width     string `json:"width"` // 20MHz / 40MHz / 80MHz / …
	PHYMode   string `json:"phyMode"`
	Security  string `json:"security"`
	RSSI      int    `json:"rssi"`  // dBm
	Noise     int    `json:"noise"` // dBm
	SNR       int    `json:"snr"`   // dB (rssi - noise)
	TxRate    int    `json:"txRate,omitempty"`
	CoChannel bool   `json:"coChannel"` // neighbor shares the connected network's channel
}

// Info is the whole Wi-Fi snapshot.
type Info struct {
	Interface   string    `json:"interface"`
	Connected   bool      `json:"connected"`
	Current     Network   `json:"current"`
	Neighbors   []Network `json:"neighbors"`
	CoChannel   int       `json:"coChannel"` // # neighbors on your channel
	SSIDRedacted bool     `json:"ssidRedacted"`
	// LocationStatus is the app's macOS Location Services authorization
	// ("authorized", "denied", "notDetermined", …). SSIDs stay redacted until
	// it's "authorized"; the UI uses this to offer a grant button.
	LocationStatus string `json:"locationStatus"`
	Note        string    `json:"note"`
}

// LocationStatus reports the app's Location Services authorization (macOS).
func LocationStatus() string { return locationStatus() }

// RequestLocation asks macOS for Location Services authorization so SSIDs can
// be shown. Shows the system prompt the first time; returns immediately.
func RequestLocation() { requestLocation() }

// parseAirport parses `system_profiler SPAirPortDataType` output. It reads the
// first Wi-Fi interface (the connected one) and stops at the next interface.
func parseAirport(text string) Info {
	var info Info
	section := ""      // "", "current", "neighbors"
	inInterfaces := false
	ifaceSeen := false
	var cur *Network

	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		t := strings.TrimSpace(line)

		switch t {
		case "Interfaces:":
			inInterfaces = true
			continue
		case "Software Versions:", "Wi-Fi:":
			continue
		case "Current Network Information:":
			section, cur = "current", nil
			continue
		case "Other Local Wi-Fi Networks:":
			section, cur = "neighbors", nil
			continue
		}

		if k, v, hasVal := splitKV(t); hasVal {
			if cur != nil {
				setField(cur, k, v)
			} else if k == "Status" && strings.Contains(v, "Connected") {
				info.Connected = true
			}
			continue
		}

		// A no-value "Name:" line.
		name := strings.TrimSuffix(t, ":")
		if inInterfaces && indent <= 8 {
			if ifaceSeen {
				break // the next interface — we only report the first (connected) one
			}
			ifaceSeen = true
			info.Interface = name
			section, cur = "", nil
			continue
		}
		switch section {
		case "current":
			info.Current.SSID = name
			cur = &info.Current
		case "neighbors":
			info.Neighbors = append(info.Neighbors, Network{SSID: name})
			cur = &info.Neighbors[len(info.Neighbors)-1]
		}
	}

	finalize(&info)
	return info
}

func finalize(info *Info) {
	info.Current.SNR = snr(info.Current)
	ch := info.Current.Channel
	for i := range info.Neighbors {
		info.Neighbors[i].SNR = snr(info.Neighbors[i])
		if ch != 0 && info.Neighbors[i].Channel == ch {
			info.Neighbors[i].CoChannel = true
			info.CoChannel++
		}
	}
	if strings.Contains(info.Current.SSID, "redacted") {
		info.SSIDRedacted = true
		info.Note = "macOS redacts SSIDs until you grant Meridian Location Services access. The RF metrics (channel, signal, SNR) are unaffected."
	}
}

// finalizeSignalOnly flags co-channel neighbors when the platform reports a
// signal but no noise (so SNR can't be computed) — used by the Linux/Windows
// backends.
func finalizeSignalOnly(info *Info) {
	ch := info.Current.Channel
	for i := range info.Neighbors {
		if ch != 0 && info.Neighbors[i].Channel == ch {
			info.Neighbors[i].CoChannel = true
			info.CoChannel++
		}
	}
}

// mergeScan overlays an in-process CoreWLAN scan onto info: when the scan
// returned any networks, its real-SSID neighbors replace the (possibly
// redacted) system_profiler list; an empty scan leaves them as-is. Either way
// co-channel is recomputed against the connected channel. Pure (no cgo).
func mergeScan(info *Info, scanned []Network) {
	if len(scanned) > 0 {
		info.Neighbors = scanned
	}
	info.CoChannel = 0
	ch := info.Current.Channel
	for i := range info.Neighbors {
		info.Neighbors[i].CoChannel = ch != 0 && info.Neighbors[i].Channel == ch
		if info.Neighbors[i].CoChannel {
			info.CoChannel++
		}
	}
}

func snr(n Network) int {
	if n.RSSI == 0 && n.Noise == 0 {
		return 0
	}
	return n.RSSI - n.Noise
}

func setField(n *Network, key, val string) {
	switch key {
	case "PHY Mode":
		n.PHYMode = val
	case "Security":
		n.Security = val
	case "Transmit Rate":
		n.TxRate = atoi(val)
	case "Channel":
		n.Channel, n.Band, n.Width = parseChannel(val)
	case "Signal / Noise":
		n.RSSI, n.Noise = parseSignalNoise(val)
	}
}

// parseChannel parses "149 (5GHz, 40MHz)" → (149, "5GHz", "40MHz").
func parseChannel(v string) (ch int, band, width string) {
	num := v
	if i := strings.IndexByte(v, '('); i >= 0 {
		num = v[:i]
		inner := strings.TrimSuffix(strings.TrimSpace(v[i+1:]), ")")
		parts := strings.SplitN(inner, ",", 2)
		if len(parts) >= 1 {
			band = strings.TrimSpace(parts[0])
		}
		if len(parts) == 2 {
			width = strings.TrimSpace(parts[1])
		}
	}
	return atoi(strings.TrimSpace(num)), band, width
}

// parseSignalNoise parses "-48 dBm / -94 dBm" → (-48, -94).
func parseSignalNoise(v string) (rssi, noise int) {
	parts := strings.SplitN(v, "/", 2)
	if len(parts) >= 1 {
		rssi = dbm(parts[0])
	}
	if len(parts) == 2 {
		noise = dbm(parts[1])
	}
	return
}

func dbm(s string) int {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "dBm")
	return atoi(strings.TrimSpace(s))
}

func splitKV(t string) (key, val string, hasVal bool) {
	i := strings.Index(t, ":")
	if i < 0 {
		return "", "", false
	}
	key = strings.TrimSpace(t[:i])
	val = strings.TrimSpace(t[i+1:])
	return key, val, val != ""
}

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}
