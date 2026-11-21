package peripherals

import (
	"fmt"
	"strings"
)

// vendors maps a subset of well-established USB vendor IDs to names. It is
// deliberately curated and high-confidence: an unknown ID is left unresolved
// (and surfaces as an "unrecognized vendor" flag) rather than guessed, so a
// device is never mislabelled. The full USB-IF list is enormous and changes
// often; expand this only with IDs confirmed against the registry.
// https://www.usb.org/developers
var vendors = map[uint16]string{
	0x046d: "Logitech",
	0x05ac: "Apple",
	0x045e: "Microsoft",
	0x1532: "Razer",
	0x1038: "SteelSeries",
	0x0b05: "ASUS",
	0x413c: "Dell",
	0x03f0: "HP",
	0x17ef: "Lenovo",
	0x04f2: "Chicony",
	0x04ca: "Lite-On",
	0x8087: "Intel",
	0x0a5c: "Broadcom",
	0x0bda: "Realtek",
	0x05e3: "Genesys Logic",
	0x2109: "VIA Labs",
	0x1a40: "Terminus (hub)",
	0x1d6b: "Linux Foundation",
	0x1050: "Yubico",
	0x096e: "Feitian",
	0x0781: "SanDisk",
	0x0951: "Kingston",
	0x090c: "Silicon Motion",
	0x18a5: "Verbatim",
	0x152d: "JMicron",
	0x174c: "ASMedia",
	0x0bc2: "Seagate",
	0x1058: "Western Digital",
	0x059f: "LaCie",
	0x0930: "Toshiba",
	0x154b: "PNY",
	0x2341: "Arduino",
	0x239a: "Adafruit",
	0x303a: "Espressif",
	0x2886: "Seeed Studio",
	0x1915: "Nordic Semiconductor",
	0x10c4: "Silicon Labs",
	0x1a86: "QinHeng (CH34x)",
	0x0403: "FTDI",
	0x067b: "Prolific",
	0x04b4: "Cypress",
	0x0483: "STMicroelectronics",
	0x2e8a: "Raspberry Pi",
	0x1b1c: "Corsair",
	0x3434: "Keychron",
	0x320f: "Glorious",
	0x046a: "Cherry",
	0x04d9: "Holtek",
	0x0c45: "Microdia",
	0x05a9: "OmniVision",
	0x1bcf: "Sunplus",
	0x0bff: "Jabra",
	0x047f: "Plantronics",
	0x0d8c: "C-Media",
}

// serialVIDs are the vendors behind the common USB-to-serial bridge chips.
// A device from one of these with no more specific class is most likely a
// serial adapter — and, relevant to Meridian, how most Meshtastic radios and
// microcontrollers attach. Not a security concern on its own.
var serialVIDs = map[uint16]bool{
	0x10c4: true, // Silicon Labs CP210x
	0x1a86: true, // QinHeng CH340/CH9102
	0x0403: true, // FTDI
	0x067b: true, // Prolific PL2303
}

// knownReceiverPIDs are 2.4 GHz wireless receivers whose product name is often
// generic ("USB Receiver"), so we pin them by ID. These dongles can have input
// devices paired to them that never appear in the USB tree themselves.
var knownReceiverPIDs = map[uint32]bool{
	id32(0x046d, 0xc52b): true, // Logitech Unifying
	id32(0x046d, 0xc52f): true, // Logitech Nano
	id32(0x046d, 0xc532): true, // Logitech Unifying
	id32(0x046d, 0xc534): true, // Logitech Nano (dual)
	id32(0x046d, 0xc548): true, // Logitech Bolt
	id32(0x046d, 0xc53a): true, // Logitech Powerplay
	id32(0x046d, 0xc53f): true, // Logitech Lightspeed
	id32(0x046d, 0xc542): true, // Logitech receiver
	id32(0x045e, 0x02fe): true, // Microsoft 2.4 GHz transceiver
	id32(0x045e, 0x02f0): true, // Microsoft Nano transceiver
}

// id32 packs a vendor and product ID into one key.
func id32(vid, pid uint16) uint32 { return uint32(vid)<<16 | uint32(pid) }

// vendorName resolves a device's vendor: the curated table first (canonical,
// consistent), then the OS-reported manufacturer string, then "" (unknown).
func vendorName(r rawDevice) string {
	if r.hasIDs {
		if n := vendors[r.vendorID]; n != "" {
			return n
		}
	}
	if m := cleanManufacturer(r.vendorName); m != "" {
		return m
	}
	return ""
}

// vendorKnown reports whether we can identify who makes the device at all —
// either a curated-table hit or a real OS manufacturer string. When false, the
// device earns an "unrecognized vendor" flag.
func vendorKnown(r rawDevice) bool {
	if r.hasIDs {
		if _, ok := vendors[r.vendorID]; ok {
			return true
		}
	}
	return cleanManufacturer(r.vendorName) != ""
}

// cleanManufacturer drops the placeholder manufacturer strings the OS fills in
// when it doesn't actually know (Windows' "(Standard system devices)", a bare
// hex echo of the VID, etc.), so they don't count as "known".
func cleanManufacturer(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	low := strings.ToLower(s)
	switch {
	case strings.HasPrefix(low, "(standard"):
		return ""
	case strings.Contains(low, "generic"):
		return ""
	case strings.HasPrefix(low, "0x"), strings.HasPrefix(low, "vid_"):
		return ""
	case low == "unknown", low == "n/a":
		return ""
	}
	return s
}

func hexID(v uint16, present bool) string {
	if !present {
		return ""
	}
	return fmt.Sprintf("0x%04x", v)
}
