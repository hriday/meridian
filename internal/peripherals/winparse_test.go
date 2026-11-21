package peripherals

import "testing"

const samplePnPArray = `[
  { "Name": "HID Keyboard Device", "Manufacturer": "(Standard keyboards)", "PNPClass": "Keyboard", "PNPDeviceID": "USB\\VID_046D&PID_C52B\\6&ABC" },
  { "Name": "USB Mass Storage Device", "Manufacturer": "Compatible USB storage device", "PNPClass": "USBSTOR", "PNPDeviceID": "USB\\VID_0781&PID_5583\\4C530" },
  { "Name": "Realtek PCIe GbE", "Manufacturer": "Realtek", "PNPClass": "Net", "PNPDeviceID": "PCI\\VEN_10EC&DEV_8168\\abc" }
]`

func TestParsePnP_array(t *testing.T) {
	devs, err := parsePnP([]byte(samplePnPArray))
	if err != nil {
		t.Fatal(err)
	}
	// The PCI NIC is dropped (not USB); two USB devices remain.
	if len(devs) != 2 {
		t.Fatalf("got %d devices, want 2: %+v", len(devs), devs)
	}
	kb := devs[0]
	if kb.vendorID != 0x046d || kb.productID != 0xc52b || !kb.hasIDs {
		t.Errorf("keyboard IDs wrong: %+v", kb)
	}
	if !hasString(kb.classes, "keyboard") {
		t.Errorf("keyboard classes = %v", kb.classes)
	}
	if !hasString(devs[1].classes, "mass-storage") {
		t.Errorf("storage classes = %v", devs[1].classes)
	}
}

func TestParsePnP_singleObject(t *testing.T) {
	const one = `{ "Name": "USB Receiver", "Manufacturer": "Logitech", "PNPClass": "HIDClass", "PNPDeviceID": "USB\\VID_046D&PID_C52B\\x" }`
	devs, err := parsePnP([]byte(one))
	if err != nil {
		t.Fatal(err)
	}
	if len(devs) != 1 || devs[0].vendorID != 0x046d {
		t.Fatalf("single-object parse wrong: %+v", devs)
	}
}
