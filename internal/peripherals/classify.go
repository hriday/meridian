package peripherals

import "strings"

// classify turns one raw device into a Device: resolves the vendor, works out
// what kind of thing it is, and attaches any security flags. Pure.
func classify(r rawDevice) Device {
	d := Device{
		Name:      strings.TrimSpace(r.name),
		Vendor:    vendorName(r),
		VendorID:  hexID(r.vendorID, r.hasIDs),
		ProductID: hexID(r.productID, r.hasIDs),
		Serial:    strings.TrimSpace(r.serial),
	}
	if d.Name == "" {
		d.Name = "(unnamed device)"
	}
	d.Receiver = isReceiver(r)
	d.Input = hasAnyClass(r, "keyboard", "mouse", "keypad")
	d.Kind = kindLabel(r, d.Receiver)
	d.Flags = flagsFor(r, d.Receiver)
	return d
}

// isReceiver spots a 2.4 GHz wireless receiver — the non-Bluetooth dongle a
// mouse/keyboard pairs to. These are flagged because they can host input
// devices that never appear in the USB tree on their own.
func isReceiver(r rawDevice) bool {
	if r.hasIDs && knownReceiverPIDs[id32(r.vendorID, r.productID)] {
		return true
	}
	// Match specific dongle phrases, not a bare "receiver" — that would snag IR
	// remotes ("Infrared Receiver"), audio receivers and the like, and hang a
	// security flag on them. Known Logitech/Microsoft dongles are caught by PID
	// above; this is the fallback for unlisted ones, which name themselves plainly.
	n := strings.ToLower(r.name)
	for _, kw := range []string{"unifying", "bolt receiver", "wireless receiver", "usb receiver", "nano transceiver", "nano receiver", "wireless dongle"} {
		if strings.Contains(n, kw) {
			return true
		}
	}
	return false
}

// kindLabel is the human description shown for the device.
func kindLabel(r rawDevice, receiver bool) string {
	switch {
	case receiver:
		return "Wireless receiver"
	case hasAnyClass(r, "keyboard"):
		return "Keyboard"
	case hasAnyClass(r, "mouse"):
		return "Mouse"
	case hasAnyClass(r, "keypad"):
		return "Keypad"
	case hasAnyClass(r, "mass-storage"):
		return "Storage"
	case hasAnyClass(r, "video"):
		return "Camera / video"
	case hasAnyClass(r, "audio"):
		return "Audio"
	case hasAnyClass(r, "serial") || (r.hasIDs && serialVIDs[r.vendorID]):
		return "USB-serial adapter"
	case hasAnyClass(r, "hub"):
		return "USB hub"
	case hasAnyClass(r, "hid"):
		return "Input device (HID)"
	default:
		return "USB device"
	}
}

// flagsFor attaches the security call-outs for a device, highest concern first.
// Every flag is a prompt to verify, never a verdict — see the package doc.
func flagsFor(r rawDevice, receiver bool) []Flag {
	var fs []Flag

	kbd := hasAnyClass(r, "keyboard")
	store := hasAnyClass(r, "mass-storage")
	known := vendorKnown(r)

	// A keyboard bundled with storage is the classic BadUSB / "rubber ducky"
	// shape. Highest-concern, so first.
	if kbd && store {
		fs = append(fs, Flag{"watch",
			"Combines a keyboard with storage — that's the shape of a BadUSB / \"rubber ducky\" device. Verify it's a legitimate combo device before trusting it."})
	} else if kbd && !known {
		fs = append(fs, Flag{"watch",
			"A keyboard from an unrecognized vendor — a device that can type keystrokes into your machine. Confirm it's a keyboard you added yourself."})
	}

	if !known {
		fs = append(fs, Flag{"watch",
			"Unrecognized vendor — Meridian can't identify who makes this. Confirm you know what it is before trusting it."})
	}

	if receiver {
		fs = append(fs, Flag{"info",
			"Wireless receiver — it can have input devices paired to it that won't appear in this list. If you don't recognize the dongle, unplug it."})
	}

	return fs
}

// isBusOrRootHub reports devices that are plumbing, not peripherals: USB host
// controllers / buses (no vendor ID) and the kernel's root hubs (Linux
// Foundation, VID 0x1d6b). These are noise in a "what's attached" view.
func isBusOrRootHub(r rawDevice) bool {
	if !r.hasIDs {
		return true
	}
	if r.vendorID == 0x1d6b { // Linux Foundation root hubs
		return true
	}
	return false
}

// kindWeight orders devices within the same flag tier: the things a person
// usually cares about (receivers, input, serial radios) above bulk storage,
// hubs and unknowns.
func kindWeight(d Device) int {
	switch d.Kind {
	case "Wireless receiver":
		return 0
	case "Keyboard", "Mouse", "Keypad", "Input device (HID)":
		return 1
	case "USB-serial adapter":
		return 2
	case "Camera / video", "Audio":
		return 3
	case "Storage":
		return 4
	case "USB hub":
		return 6
	default:
		return 5
	}
}

func hasAnyClass(r rawDevice, want ...string) bool {
	for _, c := range r.classes {
		for _, w := range want {
			if c == w {
				return true
			}
		}
	}
	return false
}
