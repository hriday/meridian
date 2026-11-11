package ipspace

import (
	"net/netip"
	"sort"
)

// This file holds the sync merge core: reconciling a local event log with a
// remote one against a common base (the log as of the last sync). The store is
// an append-only event log whose causal order IS its slice order, so this code
// never re-sorts a log by wall-clock (which would risk ordering an edit before
// its add and silently dropping it on replay). Instead it classifies changes by
// comparing the replayed record STATE of base/local/remote, and builds the
// pushed log by preserving each side's append order. Everything here is pure —
// no network, no storage.

// MergeResult is the outcome of reconciling two event logs.
type MergeResult struct {
	Merged            []Event           `json:"merged"`            // causal-ordered union to push
	Incoming          []Change          `json:"incoming"`          // record changes made only on the remote
	Outgoing          []Change          `json:"outgoing"`          // record changes made only locally
	Conflicts         []Conflict        `json:"conflicts"`         // records changed differently on both sides
	DuplicatePrefixes []PrefixCollision `json:"duplicatePrefixes"` // same prefix on >1 record after merge
}

// Change is one record-level difference from the base.
type Change struct {
	Kind   string `json:"kind"`           // "add" | "edit" | "delete"
	Record Record `json:"record"`         // the record after the change (before, for a delete)
	Prev   Record `json:"prev,omitempty"` // the prior state, for an "edit"
}

// Conflict is a record changed differently on both sides since the base.
type Conflict struct {
	ID            string `json:"id"`
	Prefix        string `json:"prefix"`
	Base          Record `json:"base"`   // common ancestor (zero if it did not exist at base)
	Local         Record `json:"local"`  // your version (zero if you deleted it)
	Remote        Record `json:"remote"` // their version (zero if they deleted it)
	LocalDeleted  bool   `json:"localDeleted"`
	RemoteDeleted bool   `json:"remoteDeleted"`
}

// PrefixCollision is one prefix held by more than one record after the merge.
type PrefixCollision struct {
	Prefix  string   `json:"prefix"`
	Records []Record `json:"records"`
}

// Merge reconciles local and remote event logs against base (the log as of the
// last sync; nil for a first sync). Each log must be in append (causal) order.
func Merge(base, local, remote []Event) MergeResult {
	baseRecs := replay(base)
	localRecs := replay(local)
	remoteRecs := replay(remote)

	var res MergeResult
	for _, id := range sortedIDs(baseRecs, localRecs, remoteRecs) {
		b, bok := baseRecs[id]
		l, lok := localRecs[id]
		r, rok := remoteRecs[id]
		changedLocal := !stateEqual(b, bok, l, lok)
		changedRemote := !stateEqual(b, bok, r, rok)

		switch {
		case changedLocal && changedRemote:
			if stateEqual(l, lok, r, rok) {
				continue // both sides converged (incl. both deleted) — agreement
			}
			res.Conflicts = append(res.Conflicts, Conflict{
				ID:            id,
				Prefix:        firstPrefix(id, localRecs, remoteRecs, baseRecs),
				Base:          b,
				Local:         l,
				Remote:        r,
				LocalDeleted:  !lok,
				RemoteDeleted: !rok,
			})
		case changedRemote:
			res.Incoming = append(res.Incoming, changeOf(b, bok, r, rok))
		case changedLocal:
			res.Outgoing = append(res.Outgoing, changeOf(b, bok, l, lok))
		}
	}

	res.Merged = causalUnion(base, local, remote)
	res.DuplicatePrefixes = duplicatePrefixes(replay(res.Merged))

	sort.SliceStable(res.Conflicts, func(i, j int) bool {
		if res.Conflicts[i].Prefix != res.Conflicts[j].Prefix {
			return lessByPrefix(res.Conflicts[i].Prefix, res.Conflicts[j].Prefix)
		}
		return res.Conflicts[i].ID < res.Conflicts[j].ID
	})
	sortChanges(res.Incoming)
	sortChanges(res.Outgoing)
	return res
}

// sortedIDs returns the union of record IDs across the maps, deterministically
// ordered (by prefix then ID) so results don't depend on map iteration.
func sortedIDs(maps ...map[string]Record) []string {
	seen := map[string]bool{}
	var ids []string
	for _, m := range maps {
		for id := range m {
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	prefixOf := func(id string) string { return firstPrefix(id, maps...) }
	sort.Slice(ids, func(i, j int) bool {
		pi, pj := prefixOf(ids[i]), prefixOf(ids[j])
		if pi != pj {
			return lessByPrefix(pi, pj)
		}
		return ids[i] < ids[j]
	})
	return ids
}

// stateEqual reports whether two record states (each present-or-absent) are the
// same: both absent, or both present and equal.
func stateEqual(a Record, aok bool, b Record, bok bool) bool {
	if aok != bok {
		return false
	}
	if !aok {
		return true
	}
	return recordEqual(a, b)
}

// changeOf describes the transition from a base state to a side's state.
func changeOf(before Record, bok bool, after Record, aok bool) Change {
	switch {
	case !bok && aok:
		return Change{Kind: "add", Record: after}
	case bok && !aok:
		return Change{Kind: "delete", Record: before}
	default:
		return Change{Kind: "edit", Record: after, Prev: before}
	}
}

// causalUnion returns every distinct event across the logs, preserving causal
// order: base first (in its order), then each side's base-new events in that
// side's append order, interleaved by timestamp but never reordered within a
// side — so an add always precedes the edits/deletes of its record.
func causalUnion(base, local, remote []Event) []Event {
	inBase := eventIDSet(base)
	var localNew, remoteNew []Event
	for _, e := range local {
		if !inBase[e.ID] {
			localNew = append(localNew, e)
		}
	}
	for _, e := range remote {
		if !inBase[e.ID] {
			remoteNew = append(remoteNew, e)
		}
	}

	out := make([]Event, 0, len(base)+len(localNew)+len(remoteNew))
	out = append(out, base...)
	seen := map[string]bool{}
	for id := range inBase {
		seen[id] = true
	}
	i, j := 0, 0
	emit := func(e Event) {
		if !seen[e.ID] {
			seen[e.ID] = true
			out = append(out, e)
		}
	}
	for i < len(localNew) || j < len(remoteNew) {
		if j >= len(remoteNew) || (i < len(localNew) && !localNew[i].At.After(remoteNew[j].At)) {
			emit(localNew[i]) // local first on a timestamp tie
			i++
		} else {
			emit(remoteNew[j])
			j++
		}
	}
	return out
}

func eventIDSet(evs []Event) map[string]bool {
	m := make(map[string]bool, len(evs))
	for _, e := range evs {
		m[e.ID] = true
	}
	return m
}

func recordEqual(a, b Record) bool {
	return a.Prefix == b.Prefix && a.Name == b.Name && a.Status == b.Status &&
		a.Notes == b.Notes && a.ParentID == b.ParentID && sameTagSet(a.Tags, b.Tags)
}

// sameTagSet compares tags as a set (order-insensitive).
func sameTagSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	ca := append([]string(nil), a...)
	cb := append([]string(nil), b...)
	sort.Strings(ca)
	sort.Strings(cb)
	for i := range ca {
		if ca[i] != cb[i] {
			return false
		}
	}
	return true
}

func firstPrefix(id string, maps ...map[string]Record) string {
	for _, m := range maps {
		if r, ok := m[id]; ok && r.Prefix != "" {
			return r.Prefix
		}
	}
	return ""
}

// duplicatePrefixes groups records by canonicalized prefix and returns any
// prefix held by more than one record.
func duplicatePrefixes(records map[string]Record) []PrefixCollision {
	groups := map[string][]Record{}
	display := map[string]string{}
	for _, r := range records {
		if r.Prefix == "" {
			continue
		}
		key := canonPrefix(r.Prefix)
		groups[key] = append(groups[key], r)
		if _, ok := display[key]; !ok {
			display[key] = r.Prefix
		}
	}
	var out []PrefixCollision
	for key, rs := range groups {
		if len(rs) > 1 {
			sort.Slice(rs, func(i, j int) bool { return rs[i].ID < rs[j].ID })
			out = append(out, PrefixCollision{Prefix: display[key], Records: rs})
		}
	}
	sort.Slice(out, func(i, j int) bool { return lessByPrefix(out[i].Prefix, out[j].Prefix) })
	return out
}

func canonPrefix(p string) string {
	if pp, err := netip.ParsePrefix(p); err == nil {
		return pp.Masked().String()
	}
	return p
}

// lessByPrefix orders two CIDR strings numerically (address then mask), falling
// back to string order for unparseable values.
func lessByPrefix(a, b string) bool {
	pa, ea := netip.ParsePrefix(a)
	pb, eb := netip.ParsePrefix(b)
	if ea != nil || eb != nil {
		return a < b
	}
	if c := pa.Addr().Compare(pb.Addr()); c != 0 {
		return c < 0
	}
	return pa.Bits() < pb.Bits()
}

func sortChanges(cs []Change) {
	sort.SliceStable(cs, func(i, j int) bool {
		if cs[i].Record.Prefix != cs[j].Record.Prefix {
			return lessByPrefix(cs[i].Record.Prefix, cs[j].Record.Prefix)
		}
		return cs[i].Record.ID < cs[j].Record.ID
	})
}
