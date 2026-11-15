package lookup

import "testing"

const radbSample = `route:          104.16.0.0/13
descr:          Cloudflare
origin:         AS13335
mnt-by:         MAINT-AS13335
source:         RADB

route6:         2606:4700::/32
origin:         AS13335
source:         RIPE

inetnum:        104.16.0.0 - 104.31.255.255
netname:        CLOUDFLARENET
`

func TestParseIRRRoutes(t *testing.T) {
	got := parseIRRRoutes(radbSample)
	if len(got) != 2 {
		t.Fatalf("want 2 route objects, got %d: %+v", len(got), got)
	}
	if got[0].Prefix != "104.16.0.0/13" || got[0].Origin != "AS13335" || got[0].Source != "RADB" {
		t.Fatalf("route[0] wrong: %+v", got[0])
	}
	if got[1].Prefix != "2606:4700::/32" || got[1].Origin != "AS13335" || got[1].Source != "RIPE" {
		t.Fatalf("route[1] (route6) wrong: %+v", got[1])
	}
}

func TestParseIRRRoutesExcludesRPKIPseudoAndComments(t *testing.T) {
	raw := `route:          1.1.1.0/24
origin:         AS13335
source:         APNIC

route:          1.1.1.0/24
origin:         AS13335
source:         RPKI  # Trust Anchor: apnic
`
	got := parseIRRRoutes(raw)
	if len(got) != 1 {
		t.Fatalf("want 1 real route object (RPKI pseudo excluded), got %d: %+v", len(got), got)
	}
	if got[0].Source != "APNIC" {
		t.Fatalf("source comment not stripped / wrong object: %+v", got[0])
	}
}

func TestIRRState(t *testing.T) {
	routes := []IRRRoute{{Prefix: "104.16.0.0/13", Origin: "AS13335", Source: "RADB"}}
	cases := []struct {
		name    string
		routes  []IRRRoute
		origins []string
		want    string
	}{
		{"valid with AS prefix", routes, []string{"AS13335"}, "valid"},
		{"valid bare number", routes, []string{"13335"}, "valid"},
		{"origin mismatch", routes, []string{"AS9999"}, "origin-mismatch"},
		{"no route objects", nil, []string{"AS13335"}, "not-found"},
	}
	for _, tc := range cases {
		if got := irrState(tc.routes, tc.origins); got != tc.want {
			t.Errorf("%s: irrState = %q, want %q", tc.name, got, tc.want)
		}
	}
}
