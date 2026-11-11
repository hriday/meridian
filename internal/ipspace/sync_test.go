package ipspace

import (
	"testing"
	"time"
)

var (
	t1 = time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 = time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC)
	t3 = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
)

func addEv(eid, actor string, at time.Time, rid, prefix, name string) Event {
	return Event{ID: eid, Type: EventAdd, Actor: actor, At: at,
		Record: Record{ID: rid, Prefix: prefix, Name: name, Status: Active}}
}
func editName(eid, actor string, at time.Time, rid, name string) Event {
	return Event{ID: eid, Type: EventEdit, Actor: actor, At: at,
		Record: Record{ID: rid, Name: name, Status: Active}}
}
func delEv(eid, actor string, at time.Time, rid string) Event {
	return Event{ID: eid, Type: EventDelete, Actor: actor, At: at, Record: Record{ID: rid}}
}

// Two people add different records → both sides' adds merge, no conflict.
func TestMergeDisjointAdds(t *testing.T) {
	local := []Event{addEv("e1", "me", t1, "A", "10.0.0.0/8", "corp")}
	remote := []Event{addEv("e2", "jo", t2, "B", "10.1.0.0/16", "branch")}

	res := Merge(nil, local, remote)
	if len(res.Conflicts) != 0 {
		t.Fatalf("want no conflicts, got %d", len(res.Conflicts))
	}
	if len(res.Incoming) != 1 || res.Incoming[0].Record.ID != "B" || res.Incoming[0].Kind != "add" {
		t.Errorf("incoming: want [add B], got %+v", res.Incoming)
	}
	if len(res.Outgoing) != 1 || res.Outgoing[0].Record.ID != "A" {
		t.Errorf("outgoing: want [add A], got %+v", res.Outgoing)
	}
	if len(res.Merged) != 2 {
		t.Errorf("merged: want 2 events, got %d", len(res.Merged))
	}
}

// Remote edits a record; local untouched → one incoming edit.
func TestMergeRemoteEditOnly(t *testing.T) {
	base := []Event{addEv("e1", "me", t1, "R", "10.0.0.0/24", "web")}
	local := base
	remote := append(cloneLog(base), editName("e2", "jo", t2, "R", "web-prod"))

	res := Merge(base, local, remote)
	if len(res.Conflicts) != 0 || len(res.Outgoing) != 0 {
		t.Fatalf("want no conflicts/outgoing; got c=%d o=%d", len(res.Conflicts), len(res.Outgoing))
	}
	if len(res.Incoming) != 1 || res.Incoming[0].Kind != "edit" || res.Incoming[0].Record.Name != "web-prod" {
		t.Errorf("incoming: want [edit R→web-prod], got %+v", res.Incoming)
	}
}

// Both sides edit the same record → a conflict carrying base/local/remote, and
// it is NOT also listed under incoming/outgoing.
func TestMergeConflictingEdits(t *testing.T) {
	base := []Event{addEv("e1", "me", t1, "R", "10.0.0.0/24", "web")}
	local := append(cloneLog(base), editName("e2", "me", t2, "R", "web-X"))
	remote := append(cloneLog(base), editName("e3", "jo", t3, "R", "web-Y"))

	res := Merge(base, local, remote)
	if len(res.Incoming) != 0 || len(res.Outgoing) != 0 {
		t.Errorf("conflicted record must not appear in incoming/outgoing; got i=%d o=%d", len(res.Incoming), len(res.Outgoing))
	}
	if len(res.Conflicts) != 1 {
		t.Fatalf("want 1 conflict, got %d", len(res.Conflicts))
	}
	c := res.Conflicts[0]
	if c.ID != "R" || c.Base.Name != "web" || c.Local.Name != "web-X" || c.Remote.Name != "web-Y" {
		t.Errorf("conflict mismatch: %+v", c)
	}
	if c.Prefix != "10.0.0.0/24" {
		t.Errorf("conflict prefix: want 10.0.0.0/24, got %q", c.Prefix)
	}
}

// Remote deletes a record → an incoming delete.
func TestMergeRemoteDelete(t *testing.T) {
	base := []Event{addEv("e1", "me", t1, "R", "10.0.0.0/24", "web")}
	local := base
	remote := append(cloneLog(base), delEv("e2", "jo", t2, "R"))

	res := Merge(base, local, remote)
	if len(res.Incoming) != 1 || res.Incoming[0].Kind != "delete" || res.Incoming[0].Record.ID != "R" {
		t.Errorf("incoming: want [delete R], got %+v", res.Incoming)
	}
}

// Delete on one side, edit on the other, is a conflict.
func TestMergeDeleteVsEdit(t *testing.T) {
	base := []Event{addEv("e1", "me", t1, "R", "10.0.0.0/24", "web")}
	local := append(cloneLog(base), delEv("e2", "me", t2, "R"))
	remote := append(cloneLog(base), editName("e3", "jo", t3, "R", "web-Y"))

	res := Merge(base, local, remote)
	if len(res.Conflicts) != 1 || res.Conflicts[0].ID != "R" {
		t.Fatalf("want 1 conflict for R, got %+v", res.Conflicts)
	}
}

// The merged log is the de-duplicated, time-ordered union of both logs.
func TestMergeMergedLogUnion(t *testing.T) {
	base := []Event{addEv("e1", "me", t1, "R", "10.0.0.0/24", "web")}
	local := append(cloneLog(base), editName("e2", "me", t2, "R", "web2"))
	remote := append(cloneLog(base), addEv("e3", "jo", t3, "S", "10.0.1.0/24", "db"))

	res := Merge(base, local, remote)
	if len(res.Merged) != 3 {
		t.Fatalf("merged: want 3 unique events, got %d", len(res.Merged))
	}
	// ordered by time: e1, e2, e3
	if res.Merged[0].ID != "e1" || res.Merged[1].ID != "e2" || res.Merged[2].ID != "e3" {
		t.Errorf("merged not time-ordered: %s %s %s", res.Merged[0].ID, res.Merged[1].ID, res.Merged[2].ID)
	}
}

// Two independently-created records with the SAME prefix (different IDs) are not
// an ID conflict, but the collision is surfaced so the user can reconcile.
func TestMergeDuplicatePrefix(t *testing.T) {
	local := []Event{addEv("e1", "me", t1, "A", "10.1.0.0/24", "a")}
	remote := []Event{addEv("e2", "jo", t2, "B", "10.1.0.0/24", "b")}

	res := Merge(nil, local, remote)
	if len(res.Conflicts) != 0 {
		t.Errorf("same-prefix/different-id is not an ID conflict; got %d", len(res.Conflicts))
	}
	if len(res.DuplicatePrefixes) != 1 || res.DuplicatePrefixes[0].Prefix != "10.1.0.0/24" {
		t.Fatalf("want duplicate-prefix 10.1.0.0/24, got %+v", res.DuplicatePrefixes)
	}
	if len(res.DuplicatePrefixes[0].Records) != 2 {
		t.Errorf("want 2 records colliding, got %d", len(res.DuplicatePrefixes[0].Records))
	}
}

// An add then an edit of the same record sharing a timestamp, where the edit's
// event ID sorts before the add's: a (At, ID) sort would run the edit first and
// drop it. The edit must survive.
func TestMergeEqualTimestampAddThenEditNotDropped(t *testing.T) {
	local := []Event{
		{ID: "zz", Type: EventAdd, Actor: "me", At: t1, Record: Record{ID: "R", Prefix: "10.0.0.0/24", Name: "web", Status: Active}},
		{ID: "aa", Type: EventEdit, Actor: "me", At: t1, Record: Record{ID: "R", Name: "web-prod", Status: Active}},
	}
	res := Merge(nil, local, nil)
	if len(res.Outgoing) != 1 || res.Outgoing[0].Record.Name != "web-prod" {
		t.Fatalf("the rename was dropped; outgoing=%+v", res.Outgoing)
	}
	if replay(res.Merged)["R"].Name != "web-prod" {
		t.Errorf("merged log replays to %q, want web-prod", replay(res.Merged)["R"].Name)
	}
}

// Remote's clock is behind, so its edit of a base-added record carries an
// earlier timestamp than the add. The edit must still apply.
func TestMergeClockSkewEditNotDropped(t *testing.T) {
	base := []Event{addEv("e1", "me", t2, "R", "10.0.0.0/24", "web")}       // add @ t2
	remote := append(cloneLog(base), editName("e2", "jo", t1, "R", "web-prod")) // edit @ t1
	res := Merge(base, base, remote)
	if len(res.Incoming) != 1 || res.Incoming[0].Record.Name != "web-prod" {
		t.Fatalf("skewed edit dropped; incoming=%+v", res.Incoming)
	}
	if replay(res.Merged)["R"].Name != "web-prod" {
		t.Errorf("merged replays to %q, want web-prod", replay(res.Merged)["R"].Name)
	}
}

// Both sides delete the same record → agreement, not a conflict.
func TestMergeBothDeleteNoConflict(t *testing.T) {
	base := []Event{addEv("e1", "me", t1, "R", "10.0.0.0/24", "web")}
	local := append(cloneLog(base), delEv("e2", "me", t2, "R"))
	remote := append(cloneLog(base), delEv("e3", "jo", t2, "R"))
	res := Merge(base, local, remote)
	if len(res.Conflicts) != 0 {
		t.Errorf("both deleting R is agreement, not a conflict; got %+v", res.Conflicts)
	}
}

// Both sides make the SAME edit → they converge, not a conflict.
func TestMergeConvergentEditNoConflict(t *testing.T) {
	base := []Event{addEv("e1", "me", t1, "R", "10.0.0.0/24", "web")}
	local := append(cloneLog(base), editName("e2", "me", t2, "R", "web-prod"))
	remote := append(cloneLog(base), editName("e3", "jo", t2, "R", "web-prod"))
	res := Merge(base, local, remote)
	if len(res.Conflicts) != 0 {
		t.Errorf("identical edits converge, not a conflict; got %+v", res.Conflicts)
	}
}

func cloneLog(evs []Event) []Event {
	out := make([]Event, len(evs))
	copy(out, evs)
	return out
}
