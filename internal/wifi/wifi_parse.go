package wifi

import (
	"strconv"
	"strings"
)

// Pure parsers for the Linux (nmcli) and Windows (netsh) Wi-Fi backends. Kept
// build-tag-free so they're unit-testable on any host; the platform files hold
// only the exec glue.

// parseNmcli parses terse (`-t`) nmcli output
// (ACTIVE,SSID,BSSID,CHAN,FREQ,SIGNAL,SECURITY) into an Info. Terse mode escapes
// literal colons inside a value (e.g. a BSSID) as "\:", which splitNmcli undoes.
func parseNmcli(text string) Info {
	info := Info{LocationStatus: "unsupported"}
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		f := splitNmcli(line)
		if len(f) < 7 {
			continue
		}
		n := Network{
			SSID:     f[1],
			Channel:  atoiSafe(f[3]),
			Band:     bandFromFreq(f[4]),
			RSSI:     signalPctToDBM(atoiSafe(f[5])),
			Security: strings.TrimSpace(f[6]),
		}
		if f[0] == "yes" {
			info.Connected = true
			info.Current = n
		} else {
			info.Neighbors = append(info.Neighbors, n)
		}
	}
	finalizeSignalOnly(&info)
	info.Note = "On Linux, Wi-Fi details come from nmcli: signal is an approximate dBm derived from NetworkManager's quality %, and noise / SNR / channel-width aren't reported."
	return info
}

// splitNmcli splits one terse nmcli line on unescaped ':' and unescapes "\:" /
// "\\" inside field values.
func splitNmcli(line string) []string {
	var out []string
	var cur strings.Builder
	for i := 0; i < len(line); i++ {
		switch {
		case line[i] == '\\' && i+1 < len(line):
			cur.WriteByte(line[i+1])
			i++
		case line[i] == ':':
			out = append(out, cur.String())
			cur.Reset()
		default:
			cur.WriteByte(line[i])
		}
	}
	out = append(out, cur.String())
	return out
}

// bandFromFreq maps an nmcli FREQ like "5180 MHz" to a band label.
func bandFromFreq(freq string) string {
	fields := strings.Fields(freq)
	if len(fields) == 0 {
		return ""
	}
	switch mhz := atoiSafe(fields[0]); {
	case mhz >= 5955:
		return "6GHz"
	case mhz >= 4900:
		return "5GHz"
	case mhz > 0:
		return "2GHz"
	}
	return ""
}

// parseNetsh parses `netsh wlan show interfaces` (the connected network) and
// `show networks mode=bssid` (all visible networks). The connected SSID from the
// interface output selects which network is "current".
func parseNetsh(iface, networks string) Info {
	info := Info{LocationStatus: "unsupported"}

	connectedSSID := ""
	for _, line := range strings.Split(iface, "\n") {
		if k, v, ok := netshKV(line); ok && k == "SSID" {
			connectedSSID = v
			break
		}
	}

	var cur *Network
	flush := func() {
		if cur == nil {
			return
		}
		if cur.SSID != "" && cur.SSID == connectedSSID && !info.Connected {
			info.Connected = true
			info.Current = *cur
		} else {
			info.Neighbors = append(info.Neighbors, *cur)
		}
		cur = nil
	}

	for _, line := range strings.Split(networks, "\n") {
		k, v, ok := netshKV(line)
		if !ok {
			continue
		}
		switch {
		case strings.HasPrefix(k, "SSID") && !strings.HasPrefix(k, "SSID name"):
			flush()
			cur = &Network{SSID: v}
		case cur == nil:
			// header lines before the first SSID block
		case k == "Authentication":
			cur.Security = v
		case strings.HasPrefix(k, "Signal"):
			cur.RSSI = signalPctToDBM(atoiSafe(strings.TrimSuffix(v, "%")))
		case k == "Channel":
			cur.Channel = atoiSafe(v)
			cur.Band = bandFromChannel(cur.Channel)
		case strings.HasPrefix(k, "Radio type"):
			cur.PHYMode = v
		}
	}
	flush()
	finalizeSignalOnly(&info)
	info.Note = "On Windows, Wi-Fi details come from netsh: signal is an approximate dBm derived from the quality %, and noise / SNR / channel-width aren't reported. Field labels are English-locale."
	return info
}

// netshKV splits a "  Key                   : Value" netsh line.
func netshKV(line string) (key, val string, ok bool) {
	i := strings.Index(line, ":")
	if i < 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:i])
	val = strings.TrimSpace(line[i+1:])
	if key == "" {
		return "", "", false
	}
	return key, val, true
}

// bandFromChannel maps a Wi-Fi channel number to a band (netsh gives channel,
// not frequency). 6GHz channels overlap numerically with 5GHz, so this is a
// best-effort 2.4/5GHz split.
func bandFromChannel(ch int) string {
	switch {
	case ch >= 1 && ch <= 14:
		return "2GHz"
	case ch >= 32 && ch <= 196:
		return "5GHz"
	}
	return ""
}

// signalPctToDBM converts a 0–100 quality percentage (nmcli / netsh) to an
// approximate dBm. 0 maps to 0 (unknown).
func signalPctToDBM(pct int) int {
	if pct <= 0 {
		return 0
	}
	return pct/2 - 100
}

func atoiSafe(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}
