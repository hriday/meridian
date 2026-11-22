package peripherals

import (
	"regexp"
	"strconv"
	"strings"
)

// macparse holds the macOS USB readers. Both parse `ioreg` output and are
// build-tag-free so they are unit-tested on every platform:
//
//   - parseIOUSBHostDevices reads the device tree (`ioreg -r -c IOUSBHostDevice
//     -l`) — the inventory: every attached device with its IDs, names, serial
//     and interface classes.
//   - parseIOHIDClasses reads the HID tree (`ioreg -r -c IOHIDDevice -l`) — the
//     precise keyboard/mouse/keypad flags, which the device tree can't give.
//
// ioreg is the authoritative source and is present on every macOS version.
// `system_profiler SPUSBDataType`, which this used to read, returns an empty
// list on recent macOS (Sequoia/Tahoe) even with devices plugged in — so it is
// no longer used.

// usbNodeHeader matches one node header line in `ioreg -l` output: the leading
// tree art (spaces and pipes), "+-o", the node name, then the "<class NAME"
// token. Capture 1 is the tree art (its length is the node's depth), capture 2
// the name, capture 3 the IOKit class.
var usbNodeHeader = regexp.MustCompile(`^([ |]*)\+-o (.*?)  <class (\w+)`)

// usbProp matches one `"Key" = value` property line inside a node body. The
// leading [ |]* skips ioreg's tree art (spaces and pipes) before the key.
var usbProp = regexp.MustCompile(`^[ |]*"([^"]+)"\s*=\s*(.*)$`)

// usbNode is one parsed ioreg node: its depth (indentation width), IOKit class,
// display name and the raw property lines in its body.
type usbNode struct {
	depth int
	class string
	name  string
	body  []string
}

// parseIOUSBHostDevices parses `ioreg -r -c IOUSBHostDevice -l` into raw
// devices. Each IOUSBHostDevice node becomes one device; its USB interface
// children (IOUSBHostInterface) contribute interface classes — mass storage,
// HID, serial/CDC, audio, video — attributed to their nearest ancestor device
// via the indentation depth. Nodes that are neither (host controllers, driver
// nubs) are skipped but still correctly close out their subtree.
func parseIOUSBHostDevices(b []byte) []rawDevice {
	nodes := splitIORegNodes(b)
	var devs []rawDevice
	// stack of enclosing devices, by depth, so an interface attaches to the
	// device that encloses it (hubs nest, so the nearest ancestor wins).
	type frame struct{ depth, idx int }
	var stack []frame
	for _, n := range nodes {
		for len(stack) > 0 && stack[len(stack)-1].depth >= n.depth {
			stack = stack[:len(stack)-1]
		}
		switch n.class {
		case "IOUSBHostDevice":
			devs = append(devs, deviceFromNode(n))
			stack = append(stack, frame{n.depth, len(devs) - 1})
		case "IOUSBHostInterface":
			if len(stack) == 0 {
				continue
			}
			if c := usbClassName(nodeInt(n, "bInterfaceClass")); c != "" {
				owner := &devs[stack[len(stack)-1].idx]
				owner.classes = unionClasses(owner.classes, []string{c})
			}
		}
	}
	return devs
}

// splitIORegNodes breaks ioreg output into nodes: each header line starts a new
// node, and the lines after it (until the next header) are its body.
func splitIORegNodes(b []byte) []usbNode {
	var nodes []usbNode
	for _, line := range strings.Split(string(b), "\n") {
		if m := usbNodeHeader.FindStringSubmatch(line); m != nil {
			nodes = append(nodes, usbNode{
				depth: len(m[1]),
				name:  strings.TrimSpace(m[2]),
				class: m[3],
			})
			continue
		}
		if len(nodes) > 0 {
			nodes[len(nodes)-1].body = append(nodes[len(nodes)-1].body, line)
		}
	}
	return nodes
}

// deviceFromNode builds a rawDevice from one IOUSBHostDevice node's properties.
func deviceFromNode(n usbNode) rawDevice {
	props := nodeProps(n)
	vid, vok := parseDecID(props["idVendor"])
	pid, pok := parseDecID(props["idProduct"])
	// kUSBProductString is the full name; "USB Product Name" is often truncated.
	name := firstNonEmpty(props["kUSBProductString"], props["USB Product Name"], stripAddr(n.name))
	vendor := firstNonEmpty(props["kUSBVendorString"], props["USB Vendor Name"])
	serial := firstNonEmpty(props["USB Serial Number"], props["kUSBSerialNumberString"])

	var classes []string
	// The device-level class covers hubs (9) and some single-function devices;
	// composite devices (class 0 / 239) carry their real classes on interfaces.
	if c := usbClassName(nodeIntFrom(props, "bDeviceClass")); c != "" {
		classes = append(classes, c)
	}
	return rawDevice{
		name:       name,
		vendorID:   vid,
		productID:  pid,
		hasIDs:     vok && pok,
		vendorName: vendor,
		serial:     cleanSerial(serial),
		classes:    classes,
	}
}

// nodeProps parses a node's body into a key -> value map, unquoting string
// values and leaving numbers as their text. Structured values (dicts, arrays,
// data) are kept verbatim; callers only read the scalar keys they need.
func nodeProps(n usbNode) map[string]string {
	props := map[string]string{}
	for _, line := range n.body {
		m := usbProp.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		val := strings.TrimSpace(m[2])
		if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
			val = val[1 : len(val)-1]
		}
		if _, seen := props[m[1]]; !seen { // keep the first (the device's own)
			props[m[1]] = val
		}
	}
	return props
}

// nodeInt reads one integer property directly from a node's body.
func nodeInt(n usbNode, key string) int { return nodeIntFrom(nodeProps(n), key) }

// nodeIntFrom reads one integer property from an already-parsed prop map,
// returning -1 when absent or non-numeric (no USB class is -1).
func nodeIntFrom(props map[string]string, key string) int {
	if v, ok := props[key]; ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
	}
	return -1
}

// usbClassName maps a USB class code (device bDeviceClass or interface
// bInterfaceClass) to the normalized class the classifier understands. Codes
// without a classifier bucket (printer, etc.) return "".
func usbClassName(code int) string {
	switch code {
	case 1:
		return "audio"
	case 2, 10: // CDC comms + CDC data — serial adapters, most Meshtastic radios
		return "serial"
	case 3:
		return "hid"
	case 6, 14: // still image (PTP cameras) + UVC video
		return "video"
	case 8:
		return "mass-storage"
	case 9:
		return "hub"
	case 224:
		return "wireless"
	default:
		return ""
	}
}

// parseDecID reads a decimal USB ID (ioreg reports idVendor/idProduct in base 10).
func parseDecID(s string) (uint16, bool) {
	if n, err := strconv.ParseUint(strings.TrimSpace(s), 10, 16); err == nil {
		return uint16(n), true
	}
	return 0, false
}

// stripAddr drops the "@location" suffix ioreg appends to a node's name.
func stripAddr(s string) string {
	if i := strings.IndexByte(s, '@'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

// cleanSerial drops the all-zero placeholder serial that hubs and anonymous
// devices report, so it isn't shown as if it were a real serial number.
func cleanSerial(s string) string {
	t := strings.TrimSpace(s)
	if t == "" || strings.Trim(t, "0") == "" {
		return ""
	}
	return t
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

// unionClasses merges two class lists, dropping duplicates and preserving order.
func unionClasses(a, b []string) []string {
	have := map[string]bool{}
	out := make([]string, 0, len(a)+len(b))
	for _, c := range append(append([]string{}, a...), b...) {
		if !have[c] {
			have[c] = true
			out = append(out, c)
		}
	}
	return out
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
