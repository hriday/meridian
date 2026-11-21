package peripherals

import "testing"

const sampleSystemProfiler = `{
  "SPUSBDataType": [
    {
      "_name": "USB31Bus",
      "_items": [
        {
          "_name": "USB Receiver",
          "manufacturer": "Logitech",
          "product_id": "0xc52b",
          "vendor_id": "0x046d  (Logitech Inc.)",
          "serial_num": "ABC123"
        },
        {
          "_name": "Keyboard Hub",
          "_items": [
            {
              "_name": "Apple Internal Keyboard / Trackpad",
              "product_id": "0x0250",
              "vendor_id": "apple_vendor_id",
              "manufacturer": "Apple Inc."
            }
          ]
        },
        {
          "_name": "Ultra Fit",
          "product_id": "0x5583",
          "vendor_id": "0x0781  (SanDisk Corp.)",
          "manufacturer": "SanDisk",
          "Media": [ { "bsd_name": "disk4", "size": "64 GB" } ]
        }
      ]
    }
  ]
}`

func TestParseSystemProfiler(t *testing.T) {
	devs, err := parseSystemProfiler([]byte(sampleSystemProfiler))
	if err != nil {
		t.Fatal(err)
	}
	// Bus and the id-less "Keyboard Hub" are skipped; three real devices remain.
	if len(devs) != 3 {
		t.Fatalf("got %d devices, want 3: %+v", len(devs), devs)
	}
	rcv := devs[0]
	if rcv.vendorID != 0x046d || rcv.productID != 0xc52b || !rcv.hasIDs {
		t.Errorf("receiver IDs wrong: %+v", rcv)
	}
	if rcv.vendorName != "Logitech" || rcv.serial != "ABC123" {
		t.Errorf("receiver fields wrong: %+v", rcv)
	}
	if len(rcv.classes) != 0 {
		t.Errorf("receiver should have no system_profiler class, got %v", rcv.classes)
	}
	apple := devs[1]
	if apple.vendorID != 0x05ac || !apple.hasIDs {
		t.Errorf("apple_vendor_id should map to 0x05ac: %+v", apple)
	}
	flash := devs[2]
	if !hasString(flash.classes, "mass-storage") {
		t.Errorf("a node with a Media array should be mass-storage, got %v", flash.classes)
	}
}

const sampleIOReg = `
+-o Apple Internal Keyboard  <class IOHIDDevice, id 0x100>
  {
    "VendorID" = 1452
    "ProductID" = 592
    "PrimaryUsagePage" = 1
    "PrimaryUsage" = 6
  }
+-o USB Optical Mouse  <class IOHIDDevice, id 0x101>
  {
    "VendorID" = 1133
    "ProductID" = 49271
    "PrimaryUsagePage" = 1
    "PrimaryUsage" = 2
  }
`

func TestParseIOHIDClasses(t *testing.T) {
	m := parseIOHIDClasses([]byte(sampleIOReg))
	kb := m[id32(1452, 592)]
	if !hasString(kb, "keyboard") || !hasString(kb, "hid") {
		t.Errorf("keyboard classes = %v, want keyboard+hid", kb)
	}
	mouse := m[id32(1133, 49271)]
	if !hasString(mouse, "mouse") {
		t.Errorf("mouse classes = %v, want mouse", mouse)
	}
}

func hasString(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
