package peripherals

import (
	"strings"
	"testing"
)

func deviceByName(s Survey, name string) *Device {
	for i := range s.Devices {
		if s.Devices[i].Name == name {
			return &s.Devices[i]
		}
	}
	return nil
}

func TestAssemble_classifiesRanksAndFlags(t *testing.T) {
	raw := []rawDevice{
		{name: "USB Bus"},                                                                                    // no IDs → dropped
		{name: "root hub", vendorID: 0x1d6b, productID: 0x0003, hasIDs: true},                                 // root hub → dropped
		{name: "USB Receiver", vendorID: 0x046d, productID: 0xc52b, hasIDs: true, vendorName: "Logitech"},     // receiver
		{name: "Apple Internal Keyboard", vendorID: 0x05ac, productID: 0x0250, hasIDs: true, vendorName: "Apple Inc.", classes: []string{"keyboard", "hid"}},
		{name: "USB Keyboard", vendorID: 0x9999, productID: 0x1111, hasIDs: true, classes: []string{"keyboard", "hid"}}, // unknown-vendor keyboard
		{name: "Ducky", vendorID: 0x8888, productID: 0x2222, hasIDs: true, classes: []string{"keyboard", "mass-storage"}}, // BadUSB shape
		{name: "Ultra Fit", vendorID: 0x0781, productID: 0x5583, hasIDs: true, vendorName: "SanDisk", classes: []string{"mass-storage"}},
	}
	s := assemble(raw, "")

	if s.Count != 5 {
		t.Fatalf("Count = %d, want 5 (bus + root hub dropped)", s.Count)
	}
	// Flagged: receiver (info), unknown keyboard (watch), Ducky (watch) = 3.
	if s.Flagged != 3 {
		t.Errorf("Flagged = %d, want 3", s.Flagged)
	}
	// Watch-flagged devices rank first.
	if !hasWatch(s.Devices[0]) || !hasWatch(s.Devices[1]) {
		t.Errorf("watch-flagged devices should rank first; got %q, %q", s.Devices[0].Name, s.Devices[1].Name)
	}

	rcv := deviceByName(s, "USB Receiver")
	if rcv == nil || !rcv.Receiver || rcv.Kind != "Wireless receiver" {
		t.Fatalf("receiver misclassified: %+v", rcv)
	}
	if len(rcv.Flags) != 1 || rcv.Flags[0].Level != "info" {
		t.Errorf("receiver should carry one info flag, got %+v", rcv.Flags)
	}

	apple := deviceByName(s, "Apple Internal Keyboard")
	if apple == nil || apple.Kind != "Keyboard" || len(apple.Flags) != 0 {
		t.Errorf("known-vendor keyboard should be clean: %+v", apple)
	}

	ducky := deviceByName(s, "Ducky")
	if ducky == nil || !hasWatch(*ducky) {
		t.Fatalf("BadUSB-shape device not flagged: %+v", ducky)
	}
	if ducky.Flags[0].Level != "watch" || !strings.Contains(ducky.Flags[0].Text, "BadUSB") {
		t.Errorf("BadUSB flag should lead and name the shape, got %+v", ducky.Flags)
	}

	unk := deviceByName(s, "USB Keyboard")
	if unk == nil || len(unk.Flags) != 2 { // keyboard-from-unknown + unrecognized-vendor
		t.Errorf("unknown-vendor keyboard should carry two watch flags, got %+v", unk)
	}
}

func TestClassify_kinds(t *testing.T) {
	cases := []struct {
		r    rawDevice
		kind string
	}{
		{rawDevice{vendorID: 0x10c4, productID: 0xea60, hasIDs: true, name: "CP2102 UART"}, "USB-serial adapter"},
		{rawDevice{hasIDs: true, classes: []string{"mouse", "hid"}}, "Mouse"},
		{rawDevice{hasIDs: true, classes: []string{"video"}, name: "FaceTime HD"}, "Camera / video"},
		{rawDevice{hasIDs: true, classes: []string{"hub"}, name: "USB2.0 Hub"}, "USB hub"},
		{rawDevice{hasIDs: true, name: "Mystery"}, "USB device"},
	}
	for _, c := range cases {
		if got := classify(c.r).Kind; got != c.kind {
			t.Errorf("classify(%q).Kind = %q, want %q", c.r.name, got, c.kind)
		}
	}
}

func TestReceiver_byNameAndPID(t *testing.T) {
	if !isReceiver(rawDevice{name: "Logitech Unifying Receiver"}) {
		t.Error("should detect receiver by name")
	}
	if !isReceiver(rawDevice{vendorID: 0x046d, productID: 0xc548, hasIDs: true, name: "USB Input Device"}) {
		t.Error("should detect Bolt receiver by PID even with a generic name")
	}
	if isReceiver(rawDevice{name: "SanDisk Ultra"}) {
		t.Error("a flash drive is not a receiver")
	}
}
