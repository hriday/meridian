package peripherals

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

// pnpEntity mirrors the fields we select from Win32_PnPEntity.
type pnpEntity struct {
	Name         string `json:"Name"`
	Manufacturer string `json:"Manufacturer"`
	PNPClass     string `json:"PNPClass"`
	PNPDeviceID  string `json:"PNPDeviceID"`
	DeviceID     string `json:"DeviceID"`
}

var winVID = regexp.MustCompile(`VID_([0-9A-Fa-f]{4})`)
var winPID = regexp.MustCompile(`PID_([0-9A-Fa-f]{4})`)

// parsePnP reads the JSON from `Get-CimInstance Win32_PnPEntity | ... |
// ConvertTo-Json` (an array, or a bare object when a single device matched) and
// returns raw devices. Vendor/product come from the VID_/PID_ in the device ID;
// classes from PNPClass. Build-tag-free so it is unit-tested on every platform.
func parsePnP(b []byte) ([]rawDevice, error) {
	b = []byte(strings.TrimSpace(string(b)))
	if len(b) == 0 {
		return nil, nil
	}
	var ents []pnpEntity
	if b[0] == '[' {
		if err := json.Unmarshal(b, &ents); err != nil {
			return nil, err
		}
	} else {
		var one pnpEntity
		if err := json.Unmarshal(b, &one); err != nil {
			return nil, err
		}
		ents = []pnpEntity{one}
	}

	var out []rawDevice
	for _, e := range ents {
		idstr := e.PNPDeviceID
		if idstr == "" {
			idstr = e.DeviceID
		}
		if !strings.HasPrefix(strings.ToUpper(idstr), "USB") {
			continue // only USB-attached devices
		}
		vid, vok := matchHex16(winVID, idstr)
		pid, pok := matchHex16(winPID, idstr)
		out = append(out, rawDevice{
			name:       e.Name,
			vendorID:   vid,
			productID:  pid,
			hasIDs:     vok && pok,
			vendorName: e.Manufacturer,
			classes:    pnpClasses(e.PNPClass, e.Name),
		})
	}
	return out, nil
}

// pnpClasses maps a Windows PNPClass (and name, as a fallback) to our classes.
func pnpClasses(class, name string) []string {
	switch strings.ToLower(strings.TrimSpace(class)) {
	case "keyboard":
		return []string{"keyboard", "hid"}
	case "mouse":
		return []string{"mouse", "hid"}
	case "hidclass":
		return []string{"hid"}
	case "diskdrive", "usbstor", "wpd":
		return []string{"mass-storage"}
	case "camera", "image":
		return []string{"video"}
	case "audioendpoint", "media":
		return []string{"audio"}
	}
	if strings.Contains(strings.ToLower(name), "hub") {
		return []string{"hub"}
	}
	return nil
}

func matchHex16(re *regexp.Regexp, s string) (uint16, bool) {
	m := re.FindStringSubmatch(s)
	if m == nil {
		return 0, false
	}
	n, err := strconv.ParseUint(m[1], 16, 16)
	if err != nil {
		return 0, false
	}
	return uint16(n), true
}
