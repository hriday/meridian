// Package peripherals takes a point-in-time inventory of the devices attached
// to this machine over USB — mice, keyboards, wireless receivers (the 2.4 GHz
// dongles that are NOT Bluetooth), flash drives, serial adapters, cameras — and
// reports each with its vendor, kind and a few honest security call-outs. It
// keeps no history: read what's plugged in now, move on.
//
// What it is NOT: it cannot see other people's wireless devices in the air. A
// 2.4 GHz dongle (Logitech Unifying and friends) uses a proprietary radio this
// machine has no receiver for; the only way it appears here is once it is
// physically plugged into a USB port. Over-the-air surveying of those devices
// needs dedicated RF hardware, which Meridian deliberately does not require.
//
// The security flags are heuristics, never verdicts. A malicious "keyboard"
// (BadUSB) presents to the OS as a genuine keyboard — nothing in enumeration
// can prove intent — so the flags point at things worth a look ("this receiver
// can host devices you can't see", "unrecognized vendor", "a device that can
// type from a source you may not know"), and a clean survey says "nothing stood
// out", never "safe".
package peripherals

import (
	"context"
	"sort"
	"time"
)

// Survey is one point-in-time inventory of the attached USB peripherals.
type Survey struct {
	Devices []Device `json:"devices"` // ranked: anything flagged first
	Count   int      `json:"count"`   // devices listed (buses/root hubs excluded)
	Flagged int      `json:"flagged"` // how many carry at least one flag
	TakenAt string   `json:"takenAt"` // RFC3339 UTC instant of this survey
	Note    string   `json:"note,omitempty"`
}

// Device is one attached peripheral.
type Device struct {
	Name      string `json:"name"`                // product name as the OS reports it
	Vendor    string `json:"vendor"`              // resolved vendor, or "" when unidentified
	VendorID  string `json:"vendorId,omitempty"`  // e.g. "0x046d"
	ProductID string `json:"productId,omitempty"` // e.g. "0xc52b"
	Kind      string `json:"kind"`                // human label: "Keyboard", "Wireless receiver", …
	Receiver  bool   `json:"receiver"`            // a dongle that can pair hidden input devices
	Input     bool   `json:"input"`               // a human-input device (keyboard/mouse/keypad)
	Serial    string `json:"serial,omitempty"`    // serial number, if the OS exposes one
	Flags     []Flag `json:"flags,omitempty"`     // security call-outs, highest concern first
}

// Flag is one security call-out on a device. Level is "watch" (worth verifying)
// or "info" (context, not a concern).
type Flag struct {
	Level string `json:"level"`
	Text  string `json:"text"`
}

// rawDevice is one enumerated device before classification — the shared shape
// every platform reader produces, and the input to the pure assembler.
type rawDevice struct {
	name       string
	vendorID   uint16
	productID  uint16
	hasIDs     bool
	vendorName string   // OS-reported manufacturer string, if any
	serial     string
	classes    []string // normalized: hub, keyboard, mouse, keypad, hid, mass-storage, audio, video, wireless, vendor
}

// Scan enumerates the USB devices attached to this host and returns them
// classified and ranked. It is read-only and instantaneous — it asks the OS
// what is attached, it does not probe or open any device.
func Scan(ctx context.Context) (Survey, error) {
	raw, note, err := enumerate(ctx)
	if err != nil {
		return Survey{}, err
	}
	return assemble(raw, note), nil
}

// assemble classifies each raw device, drops buses/root hubs, attaches flags and
// ranks flagged devices first. It is pure, so classification, flagging and
// ranking are all testable without any hardware.
func assemble(raw []rawDevice, note string) Survey {
	s := Survey{
		TakenAt: time.Now().UTC().Format(time.RFC3339),
		Note:    note,
	}
	for _, r := range raw {
		if isBusOrRootHub(r) {
			continue
		}
		s.Devices = append(s.Devices, classify(r))
	}
	rankDevices(s.Devices)
	s.Count = len(s.Devices)
	for _, d := range s.Devices {
		if len(d.Flags) > 0 {
			s.Flagged++
		}
	}
	return s
}

// rankDevices floats the devices worth attention to the top: any "watch" flag
// first, then any flag at all, then by kind weight (receivers and input devices
// before bulk storage/hubs), then by name for stability.
func rankDevices(ds []Device) {
	sort.SliceStable(ds, func(i, j int) bool {
		a, b := ds[i], ds[j]
		if wa, wb := hasWatch(a), hasWatch(b); wa != wb {
			return wa
		}
		if fa, fb := len(a.Flags) > 0, len(b.Flags) > 0; fa != fb {
			return fa
		}
		if ka, kb := kindWeight(a), kindWeight(b); ka != kb {
			return ka < kb
		}
		return a.Name < b.Name
	})
}

func hasWatch(d Device) bool {
	for _, f := range d.Flags {
		if f.Level == "watch" {
			return true
		}
	}
	return false
}
