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
	return dedupByVIDPID(out), nil
}

// dedupByVIDPID collapses the several Win32_PnPEntity rows that one physical
// device produces — a composite device's parent plus each of its interfaces all
// carry the same VID/PID — into a single device whose classes are the union of
// the interfaces'. Linux (one sysfs dir per device) and macOS (one tree node
// per device) don't duplicate, so only Windows needs this. Two physically
// identical devices (same VID/PID) collapse into one row; that's an accepted
// trade for an inventory keyed on hardware identity, and noted in the UI copy.
// Entries with no VID/PID are kept as-is.
func dedupByVIDPID(in []rawDevice) []rawDevice {
	byKey := map[uint32]*rawDevice{}
	var order []uint32
	var out []rawDevice
	for _, d := range in {
		if !d.hasIDs {
			out = append(out, d)
			continue
		}
		k := id32(d.vendorID, d.productID)
		ex := byKey[k]
		if ex == nil {
			cp := d
			byKey[k] = &cp
			order = append(order, k)
			continue
		}
		have := map[string]bool{}
		for _, c := range ex.classes {
			have[c] = true
		}
		for _, c := range d.classes {
			if !have[c] {
				ex.classes = append(ex.classes, c)
				have[c] = true
			}
		}
		if cleanManufacturer(ex.vendorName) == "" {
			ex.vendorName = d.vendorName
		}
		if isGenericWinName(ex.name) && !isGenericWinName(d.name) {
			ex.name = d.name
		}
	}
	for _, k := range order {
		out = append(out, *byKey[k])
	}
	return out
}

// isGenericWinName spots Windows' placeholder device names, so dedup prefers a
// more descriptive sibling ("USB Receiver" over "USB Composite Device").
func isGenericWinName(s string) bool {
	low := strings.ToLower(s)
	for _, g := range []string{"composite device", "hid-compliant", "usb input device", "hid keyboard device", "hid-compliant mouse"} {
		if strings.Contains(low, g) {
			return true
		}
	}
	return false
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
	case "diskdrive", "usbstor":
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
