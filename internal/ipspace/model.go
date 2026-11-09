// Package ipspace is a local, passphrase-encrypted record of your own IP space:
// a nested tree of blocks/subnets/addresses whose source of truth is an
// append-only event log (every add/edit/delete stamped with an actor and time).
// The tree state is derived by replaying the log; this file holds the model.
package ipspace

import (
	"net/netip"
	"sort"
	"time"
)

// Status is a record's lifecycle state.
type Status string

const (
	Active     Status = "active"
	Reserved   Status = "reserved"
	Deprecated Status = "deprecated"
)

// Record is one block, subnet, or address in the tree.
type Record struct {
	ID        string    `json:"id"`
	Prefix    string    `json:"prefix"` // CIDR; a single address is a host prefix (/32, /128)
	Name      string    `json:"name"`
	Status    Status    `json:"status"`
	Tags      []string  `json:"tags"`
	Notes     string    `json:"notes"`
	ParentID  string    `json:"parentId"`
	CreatedBy string    `json:"createdBy"`
	ChangedBy string    `json:"changedBy"`
	CreatedAt time.Time `json:"createdAt"`
	ChangedAt time.Time `json:"changedAt"`
}

// EventType is the kind of change an Event records.
type EventType string

const (
	EventAdd    EventType = "add"
	EventEdit   EventType = "edit"
	EventDelete EventType = "delete"
)

// Event is one entry in the append-only log.
type Event struct {
	ID     string    `json:"id"`
	Type   EventType `json:"type"`
	Actor  string    `json:"actor"`
	At     time.Time `json:"at"`
	Record Record    `json:"record"`
}

// TreeNode is a record with its nested children.
type TreeNode struct {
	Record   Record     `json:"record"`
	Children []TreeNode `json:"children"`
}

// replay derives the current records by applying the log in order.
func replay(events []Event) map[string]Record {
	m := map[string]Record{}
	for _, e := range events {
		switch e.Type {
		case EventAdd:
			r := e.Record
			r.CreatedBy, r.ChangedBy = e.Actor, e.Actor
			r.CreatedAt, r.ChangedAt = e.At, e.At
			m[r.ID] = r
		case EventEdit:
			r, ok := m[e.Record.ID]
			if !ok {
				continue
			}
			r.Name = e.Record.Name
			r.Status = e.Record.Status
			r.Tags = e.Record.Tags
			r.Notes = e.Record.Notes
			r.ChangedBy = e.Actor
			r.ChangedAt = e.At
			m[r.ID] = r
		case EventDelete:
			delete(m, e.Record.ID)
		}
	}
	return m
}

// buildTree assembles records into parent→child trees (roots have no parent),
// each level sorted numerically by prefix.
func buildTree(records map[string]Record) []TreeNode {
	childrenOf := map[string][]Record{}
	for _, r := range records {
		// A record whose parent no longer exists (the parent was deleted) is
		// surfaced as a root rather than vanishing from the tree.
		parent := r.ParentID
		if parent != "" {
			if _, ok := records[parent]; !ok {
				parent = ""
			}
		}
		childrenOf[parent] = append(childrenOf[parent], r)
	}
	var build func(parentID string) []TreeNode
	build = func(parentID string) []TreeNode {
		kids := childrenOf[parentID]
		sort.Slice(kids, func(i, j int) bool { return lessPrefix(kids[i], kids[j]) })
		out := make([]TreeNode, 0, len(kids))
		for _, r := range kids {
			out = append(out, TreeNode{Record: r, Children: build(r.ID)})
		}
		return out
	}
	return build("")
}

func lessPrefix(a, b Record) bool {
	pa, ea := netip.ParsePrefix(a.Prefix)
	pb, eb := netip.ParsePrefix(b.Prefix)
	if ea != nil || eb != nil {
		return a.Prefix < b.Prefix
	}
	if c := pa.Addr().Compare(pb.Addr()); c != 0 {
		return c < 0
	}
	return pa.Bits() < pb.Bits()
}
