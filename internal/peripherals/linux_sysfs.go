package peripherals

import (
	"io/fs"
	"strconv"
	"strings"
)

// parseSysfs reads USB devices from a /sys/bus/usb/devices view. Each device
// directory holds idVendor/idProduct/manufacturer/product/serial files and one
// subdirectory per interface (named "<dev>:<cfg>.<if>") holding the interface
// class and protocol. Taking an fs.FS keeps it pure and unit-testable with an
// in-memory tree. Build-tag-free so it is tested on every platform.
func parseSysfs(fsys fs.FS) ([]rawDevice, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}
	var out []rawDevice
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := e.Name()
		vhex := readTrim(fsys, dir+"/idVendor")
		phex := readTrim(fsys, dir+"/idProduct")
		if vhex == "" { // not a device dir (interfaces live under the device)
			continue
		}
		vid, vok := parseHex16(vhex)
		pid, pok := parseHex16(phex)
		rd := rawDevice{
			name:       readTrim(fsys, dir+"/product"),
			vendorID:   vid,
			productID:  pid,
			hasIDs:     vok && pok,
			vendorName: readTrim(fsys, dir+"/manufacturer"),
			serial:     readTrim(fsys, dir+"/serial"),
			classes:    sysfsClasses(fsys, dir),
		}
		out = append(out, rd)
	}
	return out, nil
}

// sysfsClasses collects the normalized classes across a device's interfaces.
func sysfsClasses(fsys fs.FS, dir string) []string {
	set := map[string]bool{}
	ifs, err := fs.ReadDir(fsys, dir)
	if err == nil {
		for _, sub := range ifs {
			if !sub.IsDir() || !strings.HasPrefix(sub.Name(), dir+":") {
				continue
			}
			cls := readTrim(fsys, dir+"/"+sub.Name()+"/bInterfaceClass")
			proto := readTrim(fsys, dir+"/"+sub.Name()+"/bInterfaceProtocol")
			if c := usbClass(cls, proto); c != "" {
				set[c] = true
				if c == "keyboard" || c == "mouse" || c == "keypad" {
					set["hid"] = true
				}
			}
		}
	}
	// Fall back to the device-level class when a device exposes no interfaces
	// we recognized (e.g. a bare hub).
	if len(set) == 0 {
		if c := usbClass(readTrim(fsys, dir+"/bDeviceClass"), ""); c != "" {
			set[c] = true
		}
	}
	var out []string
	for c := range set {
		out = append(out, c)
	}
	return out
}

// usbClass maps a USB interface/device class (and HID protocol) to our
// normalized class name. Class and protocol are 2-digit hex strings.
func usbClass(class, proto string) string {
	switch strings.ToLower(strings.TrimSpace(class)) {
	case "03": // HID
		switch strings.TrimSpace(proto) {
		case "01":
			return "keyboard"
		case "02":
			return "mouse"
		default:
			return "hid"
		}
	case "08":
		return "mass-storage"
	case "01":
		return "audio"
	case "0e":
		return "video"
	case "09":
		return "hub"
	case "02", "0a":
		return "serial"
	case "", "00":
		return ""
	default:
		return "vendor"
	}
}

func readTrim(fsys fs.FS, name string) string {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func parseHex16(s string) (uint16, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	n, err := strconv.ParseUint(s, 16, 16)
	if err != nil {
		return 0, false
	}
	return uint16(n), true
}
