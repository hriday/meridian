package peripherals

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

// parseSystemProfiler turns the JSON from `system_profiler SPUSBDataType -json`
// into raw devices. It walks the USB tree, emitting one device per node that
// carries a vendor or product ID (bus/controller nodes carry neither and are
// skipped, but still recursed into). Classes are left empty here — macOS does
// not expose HID interface detail in this feed; parseIOHIDClasses fills that in
// from a second source. Build-tag-free so it is unit-tested on every platform.
func parseSystemProfiler(b []byte) ([]rawDevice, error) {
	var root map[string]any
	if err := json.Unmarshal(b, &root); err != nil {
		return nil, err
	}
	var out []rawDevice
	walkUSB(root["SPUSBDataType"], &out)
	return out, nil
}

func walkUSB(node any, out *[]rawDevice) {
	switch v := node.(type) {
	case []any:
		for _, it := range v {
			walkUSB(it, out)
		}
	case map[string]any:
		vraw, _ := v["vendor_id"].(string)
		praw, _ := v["product_id"].(string)
		if vraw != "" || praw != "" {
			vid, vok := parseMacID(vraw)
			pid, pok := parseMacID(praw)
			name, _ := v["_name"].(string)
			mfr, _ := v["manufacturer"].(string)
			serial, _ := v["serial_num"].(string)
			var classes []string
			// system_profiler hangs a "Media" array (and a "bsd_name") off USB
			// mass-storage devices. It's the one interface class the USB feed
			// exposes; HID classes come separately from ioreg. Detecting it here
			// is what lets the Storage kind and the keyboard+storage (BadUSB)
			// flag fire on macOS.
			if isMacStorage(v) {
				classes = append(classes, "mass-storage")
			}
			*out = append(*out, rawDevice{
				name:       name,
				vendorID:   vid,
				productID:  pid,
				hasIDs:     vok && pok,
				vendorName: mfr,
				serial:     serial,
				classes:    classes,
			})
		}
		walkUSB(v["_items"], out)
	}
}

// parseMacID reads an ID from a system_profiler field, which is usually
// "0x046d  (Logitech Inc.)" but may be the symbolic "apple_vendor_id".
func parseMacID(s string) (uint16, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	if strings.EqualFold(s, "apple_vendor_id") {
		return 0x05ac, true
	}
	if i := strings.Index(s, "0x"); i >= 0 {
		hexs := s[i+2:]
		if j := strings.IndexFunc(hexs, func(r rune) bool { return !isHex(r) }); j >= 0 {
			hexs = hexs[:j]
		}
		if n, err := strconv.ParseUint(hexs, 16, 16); err == nil {
			return uint16(n), true
		}
	}
	return 0, false
}

func isHex(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

// isMacStorage reports whether a system_profiler USB node is a mass-storage
// device — it carries a "Media" array or a "bsd_name" (the disk device node).
func isMacStorage(v map[string]any) bool {
	if _, ok := v["Media"]; ok {
		return true
	}
	if b, ok := v["bsd_name"].(string); ok && b != "" {
		return true
	}
	return false
}

var ioregNode = regexp.MustCompile(`(?m)\+-o `)
var ioregInt = regexp.MustCompile(`"(VendorID|ProductID|PrimaryUsagePage|PrimaryUsage)"\s*=\s*(\d+)`)

// parseIOHIDClasses reads `ioreg -r -c IOHIDDevice -l` output and returns, per
// (vendor,product) pair, the HID input classes that device presents — so a
// keyboard reads as a keyboard and a mouse as a mouse. A composite device shows
// several IOHIDDevice nodes with the same IDs; their classes are unioned.
func parseIOHIDClasses(b []byte) map[uint32][]string {
	out := map[uint32]map[string]bool{}
	for _, chunk := range ioregNode.Split(string(b), -1) {
		var vid, pid, page, usage int
		var haveV, haveP bool
		for _, m := range ioregInt.FindAllStringSubmatch(chunk, -1) {
			n, _ := strconv.Atoi(m[2])
			switch m[1] {
			case "VendorID":
				vid, haveV = n, true
			case "ProductID":
				pid, haveP = n, true
			case "PrimaryUsagePage":
				page = n
			case "PrimaryUsage":
				usage = n
			}
		}
		if !haveV || !haveP {
			continue
		}
		key := id32(uint16(vid), uint16(pid))
		if out[key] == nil {
			out[key] = map[string]bool{"hid": true}
		}
		if page == 1 { // Generic Desktop
			switch usage {
			case 6:
				out[key]["keyboard"] = true
			case 7:
				out[key]["keypad"] = true
			case 2:
				out[key]["mouse"] = true
			}
		}
	}
	res := map[uint32][]string{}
	for k, set := range out {
		for c := range set {
			res[k] = append(res[k], c)
		}
	}
	return res
}
