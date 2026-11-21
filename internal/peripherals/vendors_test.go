package peripherals

import "testing"

func TestCleanManufacturer(t *testing.T) {
	cases := map[string]string{
		"Logitech":                  "Logitech",
		"(Standard system devices)": "",
		"Generic USB Hub":           "",
		"0x046d":                    "",
		"VID_046D":                  "",
		"Unknown":                   "",
		"  ":                        "",
		"Acme Corp":                 "Acme Corp",
	}
	for in, want := range cases {
		if got := cleanManufacturer(in); got != want {
			t.Errorf("cleanManufacturer(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestVendorKnown(t *testing.T) {
	if !vendorKnown(rawDevice{vendorID: 0x046d, hasIDs: true}) {
		t.Error("curated VID should be known even without a manufacturer string")
	}
	if vendorKnown(rawDevice{vendorID: 0x9abc, hasIDs: true}) {
		t.Error("unknown VID with no manufacturer should be unknown")
	}
	if !vendorKnown(rawDevice{vendorID: 0x9abc, hasIDs: true, vendorName: "Acme"}) {
		t.Error("a real manufacturer string should count as known")
	}
	if vendorKnown(rawDevice{vendorID: 0x9abc, hasIDs: true, vendorName: "(Standard system devices)"}) {
		t.Error("a placeholder manufacturer string should not count as known")
	}
}

func TestVendorName_prefersTable(t *testing.T) {
	if got := vendorName(rawDevice{vendorID: 0x046d, hasIDs: true, vendorName: "Logitech Inc."}); got != "Logitech" {
		t.Errorf("vendorName = %q, want curated %q", got, "Logitech")
	}
	if got := vendorName(rawDevice{vendorID: 0x9abc, hasIDs: true, vendorName: "Acme"}); got != "Acme" {
		t.Errorf("vendorName fallback = %q, want %q", got, "Acme")
	}
	if got := vendorName(rawDevice{vendorID: 0x9abc, hasIDs: true}); got != "" {
		t.Errorf("unknown vendor should be empty, got %q", got)
	}
}
