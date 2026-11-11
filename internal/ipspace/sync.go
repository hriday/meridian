package ipspace

import (
	"net/netip"
	"sort"
)

// This file holds the sync merge core: reconciling a local event log with a
// remote one against a common base (the log as of the last sync). Because the
// store is an append-only event log keyed by unique event IDs, merging is a
// union of events; a genuine conflict exists only when both sides changed the
// SAME record since the base. Everything here is pure — no network, no storage.

// MergeResult is the outcome of reconciling two event logs.
type MergeResult struct {
	Merged            []Event           `json:"merged"`            // de-duplicated, time-ordered union to push
	Incoming          []Change          `json:"incoming"`          // record changes made only on the remote
	Outgoing          []Change          `json:"outgoing"`          // record changes made only locally
	Conflicts         []Conflict        `json:"conflicts"`         // records changed on both sides since base
	DuplicatePrefixes []PrefixCollision `json:"duplicatePrefixes"` // same prefix on >1 record after merge
}

// Change is one record-level difference.
type Change struct {
	Kind   string `json:"kind"`           // "add" | "edit" | "delete"
	Record Record `json:"record"`         // the record after the change (before, for a delete)
	Prev   Record `json:"prev,omitempty"` // the prior state, for an "edit"
}

// Conflict is a record changed on both sides since the base.
type Conflict struct {
	ID     string `json:"id"`
	Prefix string `json:"prefix"`
	Base   Record `json:"base"`   // common ancestor (zero if it did not exist at base)
	Local  Record `json:"local"`  // your version (zero if you deleted it)
	Remote Record `json:"remote"` // their version (zero if they deleted it)
}

// PrefixCollision is one prefix held by more than one record after the merge.
type PrefixCollision struct {
	Prefix  string   `json:"prefix"`
	Records []Record `json:"records"`
}

// Merge reconciles local and remote event logs against base (the log as of the
// last sync; nil for a first sync).
func Merge(base, local, remote []Event) MergeResult {
	localIDs := eventIDSet(local)
	remoteIDs := eventIDSet(remote)

	var incomingEvents, outgoingEvents []Event
	for _, e := range remote {
		if !localIDs[e.ID] {
			incomingEvents = append(incomingEvents, e)
		}
	}
	for _, e := range local {
		if !remoteIDs[e.ID] {
			outgoingEvents = append(outgoingEvents, e)
		}
	}

	beforeBase := replay(sortedLog(base))
	afterIncoming := replay(sortedLog(concatLogs(base, incomingEvents)))
	afterOutgoing := replay(sortedLog(concatLogs(base, outgoingEvents)))

	incoming := recordDiff(beforeBase, afterIncoming)
	outgoing := recordDiff(beforeBase, afterOutgoing)

	localRecs := replay(sortedLog(local))
	remoteRecs := replay(sortedLog(remote))

	// A record changed on both sides is a conflict, removed from incoming/outgoing.
	changedRemote := changedIDs(incoming)
	changedLocal := changedIDs(outgoing)
	isConflict := map[string]bool{}
	for id := range changedRemote {
		if changedLocal[id] {
			isConflict[id] = true
		}
	}

	var res MergeResult
	for id := range isConflict {
		res.Conflicts = append(res.Conflicts, Conflict{
			ID:     id,
			Prefix: firstPrefix(id, localRecs, remoteRecs, beforeBase),
			Base:   beforeBase[id],
			Local:  localRecs[id],
			Remote: remoteRecs[id],
		})
	}
	for _, c := range incoming {
		if !isConflict[c.Record.ID] {
			res.Incoming = append(res.Incoming, c)
		}
	}
	for _, c := range outgoing {
		if !isConflict[c.Record.ID] {
			res.Outgoing = append(res.Outgoing, c)
		}
	}

	res.Merged = unionLogs(local, remote)
	res.DuplicatePrefixes = duplicatePrefixes(replay(res.Merged))

	sort.Slice(res.Conflicts, func(i, j int) bool { return lessByPrefix(res.Conflicts[i].Prefix, res.Conflicts[j].Prefix) })
	sortChanges(res.Incoming)
	sortChanges(res.Outgoing)
	return res
}

func eventIDSet(evs []Event) map[string]bool {
	m := make(map[string]bool, len(evs))
	for _, e := range evs {
		m[e.ID] = true
	}
	return m
}

func concatLogs(a, b []Event) []Event {
	out := make([]Event, 0, len(a)+len(b))
	out = append(out, a...)
	out = append(out, b...)
	return out
}

func sortedLog(evs []Event) []Event {
	out := make([]Event, len(evs))
	copy(out, evs)
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].At.Equal(out[j].At) {
			return out[i].At.Before(out[j].At)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// unionLogs returns every distinct event (by ID) across the logs, time-ordered.
func unionLogs(logs ...[]Event) []Event {
	seen := map[string]bool{}
	var out []Event
	for _, log := range logs {
		for _, e := range log {
			if !seen[e.ID] {
				seen[e.ID] = true
				out = append(out, e)
			}
		}
	}
	return sortedLog(out)
}

// recordDiff reports how the record set changed from before to after.
func recordDiff(before, after map[string]Record) []Change {
	var out []Change
	for id, r := range after {
		if b, ok := before[id]; !ok {
			out = append(out, Change{Kind: "add", Record: r})
		} else if !recordEqual(b, r) {
			out = append(out, Change{Kind: "edit", Record: r, Prev: b})
		}
	}
	for id, b := range before {
		if _, ok := after[id]; !ok {
			out = append(out, Change{Kind: "delete", Record: b})
		}
	}
	return out
}

func changedIDs(changes []Change) map[string]bool {
	m := map[string]bool{}
	for _, c := range changes {
		m[c.Record.ID] = true
	}
	return m
}

func recordEqual(a, b Record) bool {
	return a.Prefix == b.Prefix && a.Name == b.Name && a.Status == b.Status &&
		a.Notes == b.Notes && a.ParentID == b.ParentID && sameStrings(a.Tags, b.Tags)
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
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

// duplicatePrefixes groups records by prefix and returns any prefix held by more
// than one record.
func duplicatePrefixes(records map[string]Record) []PrefixCollision {
	byPrefix := map[string][]Record{}
	for _, r := range records {
		if r.Prefix != "" {
			byPrefix[r.Prefix] = append(byPrefix[r.Prefix], r)
		}
	}
	var out []PrefixCollision
	for p, rs := range byPrefix {
		if len(rs) > 1 {
			sort.Slice(rs, func(i, j int) bool { return rs[i].ID < rs[j].ID })
			out = append(out, PrefixCollision{Prefix: p, Records: rs})
		}
	}
	sort.Slice(out, func(i, j int) bool { return lessByPrefix(out[i].Prefix, out[j].Prefix) })
	return out
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
