package peripherals

import (
	"testing"
	"testing/fstest"
)

func TestParseSysfs(t *testing.T) {
	fsys := fstest.MapFS{
		// A Logitech receiver exposing a keyboard interface.
		"1-1/idVendor":                   {Data: []byte("046d\n")},
		"1-1/idProduct":                  {Data: []byte("c52b\n")},
		"1-1/manufacturer":               {Data: []byte("Logitech\n")},
		"1-1/product":                    {Data: []byte("USB Receiver\n")},
		"1-1/serial":                     {Data: []byte("ABC123\n")},
		"1-1/1-1:1.0/bInterfaceClass":    {Data: []byte("03\n")},
		"1-1/1-1:1.0/bInterfaceProtocol": {Data: []byte("01\n")},
		// A root hub (should still parse here; assemble() is what drops it).
		"usb1/idVendor":  {Data: []byte("1d6b\n")},
		"usb1/idProduct": {Data: []byte("0003\n")},
		// A bare mass-storage device.
		"2-1/idVendor":                {Data: []byte("0781\n")},
		"2-1/idProduct":               {Data: []byte("5583\n")},
		"2-1/product":                 {Data: []byte("Ultra Fit\n")},
		"2-1/2-1:1.0/bInterfaceClass": {Data: []byte("08\n")},
	}
	devs, err := parseSysfs(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if len(devs) != 3 {
		t.Fatalf("got %d devices, want 3: %+v", len(devs), devs)
	}
	byVID := map[uint16]rawDevice{}
	for _, d := range devs {
		byVID[d.vendorID] = d
	}
	rcv := byVID[0x046d]
	if rcv.productID != 0xc52b || rcv.name != "USB Receiver" || rcv.serial != "ABC123" {
		t.Errorf("receiver fields wrong: %+v", rcv)
	}
	if !hasString(rcv.classes, "keyboard") || !hasString(rcv.classes, "hid") {
		t.Errorf("receiver classes = %v, want keyboard+hid", rcv.classes)
	}
	store := byVID[0x0781]
	if !hasString(store.classes, "mass-storage") {
		t.Errorf("storage classes = %v, want mass-storage", store.classes)
	}

	// End-to-end: through assemble, the root hub is dropped.
	s := assemble(devs, "")
	if s.Count != 2 {
		t.Errorf("assemble Count = %d, want 2 (root hub dropped)", s.Count)
	}
}
