package meshtastic

import "testing"

import "go.bug.st/serial/enumerator"

func TestPortsFrom(t *testing.T) {
	in := []*enumerator.PortDetails{
		{Name: "/dev/cu.debug-console", IsUSB: false},
		{Name: "/dev/cu.Bluetooth-Incoming-Port", IsUSB: false},
		{Name: "/dev/cu.usbmodem1101", IsUSB: true, VID: "303A", PID: "1001",
			Product: "Espressif Systems LilyGO T-Deck", Manufacturer: "Espressif Systems"},
		{Name: "/dev/cu.usbserial-10", IsUSB: true, VID: "10C4", PID: "EA60", Manufacturer: "Silicon Labs"},
		{Name: "/dev/cu.usbserial-20", IsUSB: true, VID: "1A86", PID: "7523"},
		nil,
	}
	got := portsFrom(in)

	// Only the three USB ports survive; the console, Bluetooth serial and nil are dropped.
	if len(got) != 3 {
		t.Fatalf("got %d ports, want 3: %+v", len(got), got)
	}
	// Sorted by path.
	if got[0].Path != "/dev/cu.usbmodem1101" || got[1].Path != "/dev/cu.usbserial-10" || got[2].Path != "/dev/cu.usbserial-20" {
		t.Errorf("ports not sorted by path: %+v", got)
	}
	// Label falls back product -> manufacturer -> VID:PID.
	if got[0].Label != "Espressif Systems LilyGO T-Deck" {
		t.Errorf("product label wrong: %q", got[0].Label)
	}
	if got[1].Label != "Silicon Labs" {
		t.Errorf("manufacturer-fallback label wrong: %q", got[1].Label)
	}
	if got[2].Label != "USB serial device 1A86:7523" {
		t.Errorf("VID:PID-fallback label wrong: %q", got[2].Label)
	}
}

func TestPortLabelEmpty(t *testing.T) {
	if got := portLabel(&enumerator.PortDetails{Name: "/dev/ttyUSB9", IsUSB: true}); got != "USB serial device" {
		t.Errorf("empty-everything label = %q, want generic fallback", got)
	}
}

// With no USB ports, portsFrom must return a non-nil empty slice so `mesh ports
// --json` emits [] rather than null.
func TestPortsFromEmptyIsNonNil(t *testing.T) {
	if got := portsFrom(nil); got == nil || len(got) != 0 {
		t.Errorf("portsFrom(nil) = %#v, want non-nil empty slice", got)
	}
	got := portsFrom([]*enumerator.PortDetails{{Name: "/dev/cu.Bluetooth-Incoming-Port", IsUSB: false}})
	if got == nil || len(got) != 0 {
		t.Errorf("portsFrom(non-USB only) = %#v, want non-nil empty slice", got)
	}
}
