package ipspace

import "testing"

func findNode(nodes []TreeNode, prefix string) (TreeNode, bool) {
	for _, n := range nodes {
		if n.Record.Prefix == prefix {
			return n, true
		}
		if c, ok := findNode(n.Children, prefix); ok {
			return c, true
		}
	}
	return TreeNode{}, false
}

func TestCarveFullUtilization(t *testing.T) {
	s := NewStore("t")
	parent, err := s.Add("10.0.0.0/16", "campus", Active, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	added, err := s.Carve(parent.ID, 24, Reserved)
	if err != nil {
		t.Fatal(err)
	}
	if len(added) != 256 {
		t.Fatalf("carving /16 into /24 should make 256 children, got %d", len(added))
	}
	n, ok := findNode(s.Tree(), "10.0.0.0/16")
	if !ok {
		t.Fatal("parent not in tree")
	}
	if n.UtilPct < 99.9 {
		t.Fatalf("a fully-carved /16 should be ~100%% utilized, got %.2f", n.UtilPct)
	}
	if len(n.Children) != 256 {
		t.Fatalf("parent should have 256 children, got %d", len(n.Children))
	}
}

func TestPartialUtilization(t *testing.T) {
	s := NewStore("t")
	if _, err := s.Add("10.0.0.0/24", "net", Active, nil, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Add("10.0.0.0/26", "a", Active, nil, ""); err != nil { // 1/4 of the /24
		t.Fatal(err)
	}
	n, _ := findNode(s.Tree(), "10.0.0.0/24")
	if n.UtilPct < 24 || n.UtilPct > 26 {
		t.Fatalf("a /26 fills 1/4 of a /24; util = %.2f, want ~25", n.UtilPct)
	}
}

func TestCarveSkipsExisting(t *testing.T) {
	s := NewStore("t")
	parent, _ := s.Add("192.168.0.0/22", "block", Active, nil, "")
	s.Add("192.168.1.0/24", "taken", Active, nil, "") // one of the four /24s already exists
	added, err := s.Carve(parent.ID, 24, Reserved)
	if err != nil {
		t.Fatal(err)
	}
	if len(added) != 3 { // /22 → four /24s, one already there
		t.Fatalf("want 3 new /24s (one pre-existing skipped), got %d", len(added))
	}
}
