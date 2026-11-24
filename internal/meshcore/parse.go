package meshcore

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
)

// RadioConfig is the node's LoRa setup, from the SELF_INFO response. MeshCore
// has no named presets on the wire — two nodes must match these raw parameters
// (and be in range) to hear each other.
type RadioConfig struct {
	FreqMHz      float64 `json:"freqMHz"`
	BandwidthKHz float64 `json:"bandwidthKHz"`
	SpreadFactor int     `json:"spreadFactor"`
	CodingRate   int     `json:"codingRate"`
	TxPowerDBm   int     `json:"txPowerDBm"`
}

// Contact is one entry in the node's contact list — another node it has heard
// advertise. MeshCore does not carry per-contact SNR or hop count (those appear
// only on received messages), so a contact reports identity, type, its stored
// return path length, when it was last heard, and position if advertised.
type Contact struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`     // "Companion", "Repeater", "Room server", …
	PathLen     int     `json:"pathLen"`  // stored return-path length; -1 = none (reached by flood)
	LastSeen    int64   `json:"lastSeen"` // last advert, epoch seconds; 0 = unknown
	HasPosition bool    `json:"hasPosition"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
}

// selfInfo is the parsed SELF_INFO response: the connected node's own identity,
// radio config and advertised position.
type selfInfo struct {
	Name        string
	PubKey      string // hex, first 8 bytes
	Radio       RadioConfig
	HasPosition bool
	Lat, Lon    float64
}

// deviceInfo is the parsed DEVICE_INFO response — only the fields the snapshot
// surfaces. (The frame also carries firmware-ver, max-contacts and max-channels;
// add them here if a consumer ever needs them.)
type deviceInfo struct {
	Model   string
	Version string // semantic version string
}

// parseSelfInfo reads a RESP_SELF_INFO payload (code byte first). Fields past the
// fixed header are bounds-checked so a short or older-firmware frame degrades
// instead of panicking.
func parseSelfInfo(p []byte) (selfInfo, error) {
	if len(p) < 1 || p[0] != respSelfInfo {
		return selfInfo{}, fmt.Errorf("meshcore: not a self-info frame")
	}
	var si selfInfo
	if len(p) > 2 {
		si.Radio.TxPowerDBm = int(int8(p[2]))
	}
	if len(p) >= 36 {
		si.PubKey = hex.EncodeToString(p[4:12]) // first 8 bytes is enough to identify
	}
	if len(p) >= 44 {
		lat := int32(binary.LittleEndian.Uint32(p[36:40]))
		lon := int32(binary.LittleEndian.Uint32(p[40:44]))
		if lat != 0 || lon != 0 {
			si.HasPosition = true
			si.Lat = float64(lat) / 1e6
			si.Lon = float64(lon) / 1e6
		}
	}
	if len(p) >= 58 {
		// /1000 follows the wiki ("radio_freq divided by 1000", "radio_bw kHz×1000"):
		// freq is stored in kHz → MHz, bandwidth in Hz → kHz. UNVERIFIED against real
		// firmware — confirm the units on a hardware read before trusting these.
		si.Radio.FreqMHz = float64(binary.LittleEndian.Uint32(p[48:52])) / 1000
		si.Radio.BandwidthKHz = float64(binary.LittleEndian.Uint32(p[52:56])) / 1000
		si.Radio.SpreadFactor = int(p[56])
		si.Radio.CodingRate = int(p[57])
	}
	if len(p) > 58 {
		si.Name = cstr(p[58:])
	}
	return si, nil
}

// parseDeviceInfo reads a RESP_DEVICE_INFO payload. Everything past offset 2 is
// present only on firmware ver 3+, so each field is bounds-checked.
func parseDeviceInfo(p []byte) (deviceInfo, error) {
	if len(p) < 1 || p[0] != respDeviceInfo {
		return deviceInfo{}, fmt.Errorf("meshcore: not a device-info frame")
	}
	var di deviceInfo
	if len(p) >= 60 {
		di.Model = cstr(p[20:60])
	}
	if len(p) >= 80 {
		di.Version = cstr(p[60:80])
	}
	return di, nil
}

// parseContact reads a RESP_CONTACT payload into a Contact.
func parseContact(p []byte) (Contact, error) {
	if len(p) < 1 || p[0] != respContact {
		return Contact{}, fmt.Errorf("meshcore: not a contact frame")
	}
	var c Contact
	c.PathLen = -1
	if len(p) >= 34 {
		c.Type = advTypeName(p[33])
	}
	if len(p) >= 36 {
		c.PathLen = int(int8(p[35]))
	}
	if len(p) >= 132 {
		c.Name = cstr(p[100:132])
	}
	if len(p) >= 136 {
		c.LastSeen = int64(binary.LittleEndian.Uint32(p[132:136]))
	}
	if len(p) >= 144 {
		lat := int32(binary.LittleEndian.Uint32(p[136:140]))
		lon := int32(binary.LittleEndian.Uint32(p[140:144]))
		if lat != 0 || lon != 0 {
			c.HasPosition = true
			c.Lat = float64(lat) / 1e6
			c.Lon = float64(lon) / 1e6
		}
	}
	return c, nil
}

// advTypeName maps a MeshCore ADV_TYPE_* code to a human label.
func advTypeName(t byte) string {
	switch t {
	case 1:
		return "Companion"
	case 2:
		return "Repeater"
	case 3:
		return "Room server"
	case 4:
		return "Sensor"
	default:
		return "Node"
	}
}

// cstr reads a null-terminated, UTF-8 string from a fixed-width field.
func cstr(b []byte) string {
	if i := indexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	return strings.TrimSpace(string(b))
}

func indexByte(b []byte, c byte) int {
	for i := range b {
		if b[i] == c {
			return i
		}
	}
	return -1
}
