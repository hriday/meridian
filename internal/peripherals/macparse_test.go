package peripherals

import (
	"os"
	"testing"
)

// sampleIOUSBHost mirrors the real shape of `ioreg -r -c IOUSBHostDevice -l`
// (names and serials made generic): a host controller (ignored — not an
// IOUSBHostDevice), an external hub, a mouse, an Espressif CDC board of the
// kind most Meshtastic radios use (interface classes 2/10 → serial), and a
// mass-storage stick (interface class 8). Interface children carry the real
// class for composite devices; indentation nests them under their device.
const sampleIOUSBHost = `+-o AppleT6000USBXHCI@01000000  <class AppleUSBXHCITR6000, id 0x100000001, registered, matched, active, busy 0, retain 20>
  {
    "IOClass" = "AppleUSBXHCITR6000"
  }
  | +-o USB2.0 Hub@00100000  <class IOUSBHostDevice, id 0x100000002, registered, matched, active, busy 0, retain 53>
  |   {
  |     "idVendor" = 8457
  |     "idProduct" = 10263
  |     "bDeviceClass" = 9
  |     "USB Product Name" = "USB2.0 Hub"
  |     "USB Vendor Name" = "VIA Labs, Inc."
  |     "USB Serial Number" = "000000000"
  |   }
  |   | +-o USB Optical Mouse@00134000  <class IOUSBHostDevice, id 0x100000003, registered, matched, active, busy 0, retain 60>
  |   |   {
  |   |     "idVendor" = 1133
  |   |     "idProduct" = 49271
  |   |     "bDeviceClass" = 0
  |   |     "USB Product Name" = "USB Optical Mouse"
  |   |     "USB Vendor Name" = "Logitech"
  |   |   }
  |   |   | +-o IOUSBHostInterface@0  <class IOUSBHostInterface, id 0x100000004, registered, matched, active, busy 0, retain 11>
  |   |   |   {
  |   |   |     "idVendor" = 1133
  |   |   |     "idProduct" = 49271
  |   |   |     "bInterfaceClass" = 3
  |   |   |   }
  |   | +-o Espressif Systems LilyGO T-Deck@02100000  <class IOUSBHostDevice, id 0x100000005, registered, matched, active, busy 0, retain 46>
  |   |   {
  |   |     "idVendor" = 12346
  |   |     "idProduct" = 4097
  |   |     "bDeviceClass" = 239
  |   |     "kUSBProductString" = "Espressif Systems LilyGO T-Deck (16 MB FLASH, 8 MB PSRAM)"
  |   |     "USB Product Name" = "Espressif Systems LilyGO T_Deck"
  |   |     "kUSBVendorString" = "Espressif Systems"
  |   |     "USB Serial Number" = "A1B2C3D4E5F6"
  |   |   }
  |   |   | +-o IOUSBHostInterface@0  <class IOUSBHostInterface, id 0x100000006, registered, matched, active, busy 0, retain 11>
  |   |   |   {
  |   |   |     "bInterfaceClass" = 2
  |   |   |     "bInterfaceSubClass" = 2
  |   |   |   }
  |   |   | +-o IOUSBHostInterface@2  <class IOUSBHostInterface, id 0x100000007, registered, matched, active, busy 0, retain 11>
  |   |   |   {
  |   |   |     "bInterfaceClass" = 10
  |   |   |     "bInterfaceSubClass" = 0
  |   |   |   }
  |   | +-o Ultra Fit@02200000  <class IOUSBHostDevice, id 0x100000008, registered, matched, active, busy 0, retain 40>
  |   |   {
  |   |     "idVendor" = 1921
  |   |     "idProduct" = 21891
  |   |     "bDeviceClass" = 0
  |   |     "USB Product Name" = "Ultra Fit"
  |   |     "USB Vendor Name" = "SanDisk"
  |   |     "USB Serial Number" = "4C530001234567"
  |   |   }
  |   |   | +-o IOUSBHostInterface@0  <class IOUSBHostInterface, id 0x100000009, registered, matched, active, busy 0, retain 11>
  |   |   |   {
  |   |   |     "bInterfaceClass" = 8
  |   |   |   }
`

func TestParseIOUSBHostDevices(t *testing.T) {
	devs := parseIOUSBHostDevices([]byte(sampleIOUSBHost))
	// The host controller is not an IOUSBHostDevice; four real devices remain.
	if len(devs) != 4 {
		t.Fatalf("got %d devices, want 4: %+v", len(devs), devs)
	}
	byVID := map[uint16]rawDevice{}
	for _, d := range devs {
		byVID[d.vendorID] = d
	}

	// The Meshtastic board: present, fully identified, and its CDC interfaces
	// make it a serial device — the regression this fix exists for.
	esp, ok := byVID[0x303a]
	if !ok {
		t.Fatalf("Espressif CDC device (0x303a) missing from the inventory: %+v", devs)
	}
	if !esp.hasIDs || esp.productID != 0x1001 {
		t.Errorf("Espressif IDs wrong: %+v", esp)
	}
	if esp.name != "Espressif Systems LilyGO T-Deck (16 MB FLASH, 8 MB PSRAM)" {
		t.Errorf("Espressif should take the full kUSBProductString name, got %q", esp.name)
	}
	if esp.serial != "A1B2C3D4E5F6" {
		t.Errorf("Espressif serial wrong: %q", esp.serial)
	}
	if !hasString(esp.classes, "serial") {
		t.Errorf("Espressif CDC interfaces should make it serial, got %v", esp.classes)
	}

	if hub := byVID[8457]; !hasString(hub.classes, "hub") || hub.serial != "" {
		t.Errorf("hub should be class hub with its all-zero serial dropped: %+v", hub)
	}
	if mouse := byVID[1133]; !hasString(mouse.classes, "hid") {
		t.Errorf("mouse interface class 3 should be hid, got %v", mouse.classes)
	}
	if stick := byVID[1921]; !hasString(stick.classes, "mass-storage") {
		t.Errorf("storage interface class 8 should be mass-storage, got %v", stick.classes)
	}
}

// TestAssembleFromIOReg checks the end-to-end classification: the Meshtastic
// board must show up and read as a serial adapter, not vanish.
func TestAssembleFromIOReg(t *testing.T) {
	s := assemble(parseIOUSBHostDevices([]byte(sampleIOUSBHost)), "")
	if s.Count != 4 {
		t.Fatalf("survey count = %d, want 4", s.Count)
	}
	var esp *Device
	for i := range s.Devices {
		if s.Devices[i].VendorID == "0x303a" {
			esp = &s.Devices[i]
		}
	}
	if esp == nil {
		t.Fatalf("the Meshtastic board is missing from the survey: %+v", s.Devices)
	}
	if esp.Kind != "USB-serial adapter" {
		t.Errorf("Meshtastic board kind = %q, want \"USB-serial adapter\"", esp.Kind)
	}
	if esp.Vendor != "Espressif" {
		t.Errorf("Meshtastic board vendor = %q, want \"Espressif\"", esp.Vendor)
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

// TestParseRealIORegFixture runs the parser against a real, unmodified-shape
// `ioreg -r -c IOUSBHostDevice -l` capture (serials redacted) from a Mac with
// the T-Deck, a Targus/DisplayLink dock, hubs, a mouse and a keyboard attached.
// It exercises the actual tree — driver nubs interposed between devices and
// interfaces, single-line IOMatchedPersonality dicts that embed their own
// bInterfaceClass — not a tidy synthetic one, so a drift in the depth/stack
// attribution or the property shadowing guard would be caught here.
func TestParseRealIORegFixture(t *testing.T) {
	b, err := os.ReadFile("testdata/ioreg_real.txt")
	if err != nil {
		t.Fatal(err)
	}
	devs := parseIOUSBHostDevices(b)
	if len(devs) != 10 {
		t.Fatalf("got %d devices, want 10 (one per IOUSBHostDevice node)", len(devs))
	}
	byVID := map[uint16]rawDevice{}
	for _, d := range devs {
		byVID[d.vendorID] = d
	}

	// The T-Deck's CDC interface is ACM (subclass 2): a real serial port.
	esp, ok := byVID[0x303a]
	if !ok {
		t.Fatalf("T-Deck (0x303a) missing from the real-fixture inventory")
	}
	if !hasString(esp.classes, "serial") || hasString(esp.classes, "network") {
		t.Errorf("T-Deck should be serial, not network: %v", esp.classes)
	}
	if esp.name != "Espressif Systems LilyGO T-Deck (16 MB FLASH, 8 MB PSRAM)" || esp.serial != "REDACTED" {
		t.Errorf("T-Deck name/serial parsed wrong: name=%q serial=%q", esp.name, esp.serial)
	}

	// The dock's CDC interface is NCM (subclass 13): USB-Ethernet, never serial.
	dock, ok := byVID[0x17e9]
	if !ok {
		t.Fatalf("Targus/DisplayLink dock (0x17e9) missing from the real-fixture inventory")
	}
	if hasString(dock.classes, "serial") {
		t.Errorf("dock CDC-NCM must not be classed as a serial adapter: %v", dock.classes)
	}
	if !hasString(dock.classes, "network") {
		t.Errorf("dock should carry a network class from its NCM interface: %v", dock.classes)
	}

	// End to end: the board reads as a serial adapter, the dock does not.
	s := assemble(devs, "")
	for _, d := range s.Devices {
		if d.VendorID == "0x303a" && d.Kind != "USB-serial adapter" {
			t.Errorf("T-Deck kind = %q, want \"USB-serial adapter\"", d.Kind)
		}
		if d.VendorID == "0x17e9" && d.Kind == "USB-serial adapter" {
			t.Errorf("dock must not be labelled a USB-serial adapter")
		}
	}
}

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
