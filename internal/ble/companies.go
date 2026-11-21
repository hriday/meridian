package ble

import "fmt"

// companies maps a subset of well-established Bluetooth SIG company identifiers
// to names. It is deliberately small and high-confidence: an unknown ID is
// rendered as its raw hex (companyLabel) rather than guessed, so a device is
// never mislabelled. The full assigned-numbers list is large and changes often;
// expand this only with IDs confirmed against the SIG registry.
// https://www.bluetooth.com/specifications/assigned-numbers/
var companies = map[uint16]string{
	0x0000: "Ericsson",
	0x0001: "Nokia",
	0x0002: "Intel",
	0x0003: "IBM",
	0x0006: "Microsoft",
	0x0008: "Motorola",
	0x000A: "Cambridge Silicon Radio",
	0x000F: "Broadcom",
	0x004C: "Apple",
	0x0059: "Nordic Semiconductor",
	0x0075: "Samsung Electronics",
	0x0087: "Garmin",
	0x00E0: "Google",
}

// companyLabel returns the known company name for id, or its hex form (e.g.
// "0x0157") when the ID isn't in the curated table — honest about the raw value
// rather than inventing a vendor.
func companyLabel(id uint16) string {
	if name, ok := companies[id]; ok {
		return name
	}
	return fmt.Sprintf("0x%04X", id)
}
