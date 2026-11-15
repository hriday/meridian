package lanrecon

import (
	"testing"

	"github.com/hriday/meridian/internal/arpscan"
	"github.com/hriday/meridian/internal/ipspace"
)

func sweep(hosts ...arpscan.Host) arpscan.Result {
	return arpscan.Result{Interface: "en0", Subnet: "192.168.1.0/24", Hosts: hosts}
}

func rec(id, prefix, name string, status ipspace.Status, tags ...string) ipspace.Record {
	return ipspace.Record{ID: id, Prefix: prefix, Name: name, Status: status, Tags: tags}
}

func find(r Report, ip string) (Finding, bool) {
	for _, f := range r.Findings {
		if f.IP == ip {
			return f, true
		}
	}
	return Finding{}, false
}

func TestReconcileNew(t *testing.T) {
	r := Reconcile(sweep(arpscan.Host{IP: "192.168.1.50", MAC: "aa:bb:cc:dd:ee:ff", Vendor: "Acme"}), nil)
	f, ok := find(r, "192.168.1.50")
	if !ok || f.Category != New {
		t.Fatalf("want New, got %+v (ok=%v)", f, ok)
	}
	if f.MAC != "aa:bb:cc:dd:ee:ff" || f.Vendor != "Acme" {
		t.Fatalf("observed data not carried: %+v", f)
	}
	if r.New != 1 {
		t.Fatalf("New count = %d, want 1", r.New)
	}
}

func TestReconcileInSync(t *testing.T) {
	records := []ipspace.Record{rec("r1", "192.168.1.50/32", "server", ipspace.Active, "mac:aa:bb:cc:dd:ee:ff")}
	r := Reconcile(sweep(arpscan.Host{IP: "192.168.1.50", MAC: "AA:BB:CC:DD:EE:FF"}), records)
	f, _ := find(r, "192.168.1.50")
	if f.Category != InSync {
		t.Fatalf("want InSync (case-insensitive MAC match), got %q", f.Category)
	}
	if f.RecordID != "r1" || f.Name != "server" {
		t.Fatalf("inventory linkage missing: %+v", f)
	}
}

func TestReconcileMACChange(t *testing.T) {
	records := []ipspace.Record{rec("r1", "192.168.1.50/32", "server", ipspace.Active, "mac:aa:bb:cc:dd:ee:ff")}
	r := Reconcile(sweep(arpscan.Host{IP: "192.168.1.50", MAC: "11:22:33:44:55:66"}), records)
	f, _ := find(r, "192.168.1.50")
	if f.Category != MACChange {
		t.Fatalf("want MACChange, got %q", f.Category)
	}
	if f.KnownMAC != "aa:bb:cc:dd:ee:ff" || f.MAC != "11:22:33:44:55:66" {
		t.Fatalf("want known vs observed MACs, got known=%q obs=%q", f.KnownMAC, f.MAC)
	}
}

func TestReconcileMissing(t *testing.T) {
	records := []ipspace.Record{
		rec("r1", "192.168.1.60/32", "printer", ipspace.Active, "mac:de:ad:be:ef:00:01"),
		rec("r2", "192.168.1.61/32", "old", ipspace.Deprecated, "mac:de:ad:be:ef:00:02"), // deprecated → not flagged
		rec("r3", "10.0.0.5/32", "elsewhere", ipspace.Active), // outside swept subnet → not flagged
	}
	r := Reconcile(sweep(arpscan.Host{IP: "192.168.1.50", MAC: "aa:bb:cc:dd:ee:ff"}), records)
	f, ok := find(r, "192.168.1.60")
	if !ok || f.Category != Missing {
		t.Fatalf("want Missing for .60, got %+v (ok=%v)", f, ok)
	}
	if _, ok := find(r, "192.168.1.61"); ok {
		t.Fatalf("deprecated record should not be flagged Missing")
	}
	if _, ok := find(r, "10.0.0.5"); ok {
		t.Fatalf("record outside swept subnet should not be flagged Missing")
	}
}

func TestReconcileBareIPPrefixAndNoMACTag(t *testing.T) {
	// A record stored as a bare address (no /32) with no mac tag still matches by
	// IP and counts as in-sync (apply can backfill the MAC later).
	records := []ipspace.Record{rec("r1", "192.168.1.50", "host", ipspace.Active)}
	r := Reconcile(sweep(arpscan.Host{IP: "192.168.1.50", MAC: "aa:bb:cc:dd:ee:ff"}), records)
	f, _ := find(r, "192.168.1.50")
	if f.Category != InSync || f.RecordID != "r1" {
		t.Fatalf("want InSync linked to r1, got %+v", f)
	}
}
