package meshtastic

import (
	"fmt"
	"sort"
	"strings"

	"go.bug.st/serial/enumerator"
)

// SerialPort is one attached USB serial device a node could be reached on: the
// device path to connect to, and a human label to recognise it by.
type SerialPort struct {
	Path  string `json:"path"`  // e.g. "/dev/cu.usbmodem1101" or "COM3"
	Label string `json:"label"` // e.g. "Espressif Systems LilyGO T-Deck"
}

// SerialPorts lists the USB serial devices attached right now — the candidates
// for the serial side of a Meshtastic read, so the caller can pick one instead
// of typing a device path. Read-only and point-in-time; it enumerates, it does
// not open any port. Non-USB ports (a Mac's debug console, Bluetooth serial)
// are left out: a Meshtastic node attaches over USB or TCP, not those.
func SerialPorts() ([]SerialPort, error) {
	// Passive enumeration only — no filter, which would turn on active USB
	// probing. Probing opens every attached serial device to read its string
	// descriptors, and the library warns that can interfere with a device in
	// use (a console cable live on a switch, a printer mid-job). The product
	// name is read passively from the OS where it's available (macOS IOKit,
	// Linux sysfs); where it isn't, portLabel falls back to the USB IDs.
	details, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, fmt.Errorf("meshtastic: listing serial ports: %w", err)
	}
	return portsFrom(details), nil
}

// portsFrom is the pure part of SerialPorts: filter to USB serial devices, give
// each a label, and sort for a stable order. Separated so it is unit-tested
// without any hardware.
func portsFrom(details []*enumerator.PortDetails) []SerialPort {
	// Non-nil so an empty result marshals as JSON [] (not null) for `mesh ports
	// --json`, and reads as an empty list everywhere.
	ports := []SerialPort{}
	for _, d := range details {
		if d == nil || !d.IsUSB {
			continue
		}
		ports = append(ports, SerialPort{Path: d.Name, Label: portLabel(d)})
	}
	sort.Slice(ports, func(i, j int) bool { return ports[i].Path < ports[j].Path })
	return ports
}

// portLabel picks the most recognisable name for a port: the device's own
// product string, then its manufacturer, then its USB IDs, then a plain
// fallback — never empty, so every entry in the picker reads as something.
func portLabel(d *enumerator.PortDetails) string {
	if s := strings.TrimSpace(d.Product); s != "" {
		return s
	}
	if s := strings.TrimSpace(d.Manufacturer); s != "" {
		return s
	}
	if d.VID != "" || d.PID != "" {
		return fmt.Sprintf("USB serial device %s:%s", d.VID, d.PID)
	}
	return "USB serial device"
}
