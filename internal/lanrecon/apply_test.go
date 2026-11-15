package lanrecon

import (
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/ipspace"
)

func hasTag(tags []string, want string) bool {
	for _, t := range tags {
		if strings.EqualFold(t, want) {
			return true
		}
	}
	return false
}

func recByPrefix(store *ipspace.Store, prefix string) (ipspace.Record, bool) {
	for _, r := range store.Records() {
		if r.Prefix == prefix {
			return r, true
		}
	}
	return ipspace.Record{}, false
}

func TestApplyImportsNew(t *testing.T) {
	store := ipspace.NewStore("tester")
	findings := []Finding{{IP: "192.168.1.50", MAC: "aa:bb:cc:dd:ee:ff", Vendor: "Acme", Category: New}}

	res, err := Apply(store, findings, ApplyOptions{ImportNew: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Imported != 1 {
		t.Fatalf("Imported = %d, want 1", res.Imported)
	}
	r, ok := recByPrefix(store, "192.168.1.50/32")
	if !ok {
		t.Fatal("new host not added as /32")
	}
	if !hasTag(r.Tags, "mac:aa:bb:cc:dd:ee:ff") || !hasTag(r.Tags, "vendor:Acme") {
		t.Fatalf("mac/vendor tags missing: %v", r.Tags)
	}
}

func TestApplyImportNewDisabled(t *testing.T) {
	store := ipspace.NewStore("tester")
	findings := []Finding{{IP: "192.168.1.50", MAC: "aa:bb:cc:dd:ee:ff", Category: New}}
	res, err := Apply(store, findings, ApplyOptions{ImportNew: false})
	if err != nil {
		t.Fatal(err)
	}
	if res.Imported != 0 || len(store.Records()) != 0 {
		t.Fatalf("nothing should be imported when disabled: %+v / %d records", res, len(store.Records()))
	}
}

func TestApplyUpdatesMAC(t *testing.T) {
	store := ipspace.NewStore("tester")
	orig, err := store.Add("192.168.1.50/32", "server", ipspace.Active, []string{"mac:aa:bb:cc:dd:ee:ff", "role:db"}, "")
	if err != nil {
		t.Fatal(err)
	}
	findings := []Finding{{IP: "192.168.1.50", MAC: "11:22:33:44:55:66", Category: MACChange, RecordID: orig.ID, KnownMAC: "aa:bb:cc:dd:ee:ff"}}

	res, err := Apply(store, findings, ApplyOptions{UpdateMAC: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Updated != 1 {
		t.Fatalf("Updated = %d, want 1", res.Updated)
	}
	r, _ := recByPrefix(store, "192.168.1.50/32")
	if !hasTag(r.Tags, "mac:11:22:33:44:55:66") || hasTag(r.Tags, "mac:aa:bb:cc:dd:ee:ff") {
		t.Fatalf("mac tag not replaced: %v", r.Tags)
	}
	if !hasTag(r.Tags, "role:db") {
		t.Fatalf("unrelated tag dropped: %v", r.Tags)
	}
	if r.Name != "server" {
		t.Fatalf("name changed unexpectedly: %q", r.Name)
	}
}

func TestApplyDeprecatesMissing(t *testing.T) {
	store := ipspace.NewStore("tester")
	orig, _ := store.Add("192.168.1.60/32", "printer", ipspace.Active, []string{"mac:de:ad:be:ef:00:01"}, "")
	findings := []Finding{{IP: "192.168.1.60", Category: Missing, RecordID: orig.ID}}

	res, err := Apply(store, findings, ApplyOptions{DeprecateMissing: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Deprecated != 1 {
		t.Fatalf("Deprecated = %d, want 1", res.Deprecated)
	}
	r, _ := recByPrefix(store, "192.168.1.60/32")
	if r.Status != ipspace.Deprecated {
		t.Fatalf("status = %q, want deprecated", r.Status)
	}
}
