package ipspace

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/netip"
	"sort"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/subnet"
)

// Store holds the append-only event log and the records derived from it.
type Store struct {
	actor   string
	events  []Event
	records map[string]Record
}

// NewStore returns an empty store whose events are stamped with actor.
func NewStore(actor string) *Store {
	return &Store{actor: actor, records: map[string]Record{}}
}

// FromEvents rebuilds a store from an existing log.
func FromEvents(actor string, events []Event) *Store {
	s := &Store{actor: actor, events: append([]Event(nil), events...)}
	s.rebuild()
	return s
}

func (s *Store) rebuild() { s.records = replay(s.events) }

// Events returns a copy of the log (for persistence).
func (s *Store) Events() []Event { return append([]Event(nil), s.events...) }

// History returns the log in chronological order.
func (s *Store) History() []Event { return s.Events() }

// Records returns the current records, prefix-sorted.
func (s *Store) Records() []Record {
	out := make([]Record, 0, len(s.records))
	for _, r := range s.records {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return lessPrefix(out[i], out[j]) })
	return out
}

// Tree returns the nested record tree.
func (s *Store) Tree() []TreeNode { return buildTree(s.records) }

// Add records a new block or address, nesting it under the most specific
// containing block and rejecting a duplicate or a range that would swallow
// existing blocks.
func (s *Store) Add(prefix, name string, status Status, tags []string, notes string) (Record, error) {
	p, err := netip.ParsePrefix(normalizePrefix(prefix))
	if err != nil {
		return Record{}, fmt.Errorf("not a valid CIDR or address: %w", err)
	}
	p = p.Masked()
	parentID, err := findParent(s.Records(), p)
	if err != nil {
		return Record{}, err
	}
	rec := Record{
		ID: newID(), Prefix: p.String(), Name: name,
		Status: defStatus(status), Tags: tags, Notes: notes, ParentID: parentID,
	}
	s.appendEvent(Event{Type: EventAdd, Record: rec})
	return s.records[rec.ID], nil
}

// maxCarve caps how many child subnets one carve creates, so a wide split can't
// flood the inventory.
const maxCarve = 1024

// Carve subdivides the block with id into equal child subnets of newPrefixLen
// and adds them (status for each; skipping any that already exist), nesting them
// under the block automatically. It returns the records it created.
func (s *Store) Carve(id string, newPrefixLen int, status Status) ([]Record, error) {
	parent, ok := s.records[id]
	if !ok {
		return nil, fmt.Errorf("no such block")
	}
	subs, err := subnet.Split(parent.Prefix, newPrefixLen)
	if err != nil {
		return nil, err
	}
	if len(subs) > maxCarve {
		return nil, fmt.Errorf("that split makes %d subnets; carve into a shorter prefix (at most %d at once)", len(subs), maxCarve)
	}

	existing := map[string]bool{}
	for _, r := range s.records {
		existing[r.Prefix] = true
	}

	var added []Record
	for _, sn := range subs {
		pfx := sn
		if p, err := netip.ParsePrefix(sn); err == nil {
			pfx = p.Masked().String()
		}
		if existing[pfx] {
			continue
		}
		r, err := s.Add(sn, "", status, nil, "")
		if err != nil {
			continue // skip anything that would conflict
		}
		added = append(added, r)
	}
	if len(added) == 0 {
		return nil, fmt.Errorf("nothing to carve — all %d subnet(s) already exist", len(subs))
	}
	return added, nil
}

// Edit updates a record's mutable fields.
func (s *Store) Edit(id, name string, status Status, tags []string, notes string) error {
	if _, ok := s.records[id]; !ok {
		return fmt.Errorf("no such record %q", id)
	}
	s.appendEvent(Event{Type: EventEdit, Record: Record{
		ID: id, Name: name, Status: defStatus(status), Tags: tags, Notes: notes,
	}})
	return nil
}

// Delete removes a record.
func (s *Store) Delete(id string) error {
	if _, ok := s.records[id]; !ok {
		return fmt.Errorf("no such record %q", id)
	}
	s.appendEvent(Event{Type: EventDelete, Record: Record{ID: id}})
	return nil
}

// Undo drops the most recent event.
func (s *Store) Undo() error {
	if len(s.events) == 0 {
		return fmt.Errorf("nothing to undo")
	}
	s.events = s.events[:len(s.events)-1]
	s.rebuild()
	return nil
}

func (s *Store) appendEvent(e Event) {
	e.ID = newID()
	e.Actor = s.actor
	e.At = time.Now().UTC()
	s.events = append(s.events, e)
	s.rebuild()
}

// findParent returns the ID of the most specific existing block that contains p,
// or an error if p duplicates or would contain an existing block.
func findParent(existing []Record, p netip.Prefix) (string, error) {
	parentID := ""
	parentBits := -1
	for _, e := range existing {
		ep, err := netip.ParsePrefix(e.Prefix)
		if err != nil {
			continue
		}
		ep = ep.Masked()
		if ep == p {
			return "", fmt.Errorf("%s already recorded (%s)", p, e.Name)
		}
		if !ep.Overlaps(p) {
			continue
		}
		if ep.Bits() < p.Bits() { // ep contains p — candidate parent
			if ep.Bits() > parentBits {
				parentBits, parentID = ep.Bits(), e.ID
			}
		} else { // p contains ep
			return "", fmt.Errorf("%s would contain the existing block %s (%s) — add inside it or remove it first", p, ep, e.Name)
		}
	}
	return parentID, nil
}

func normalizePrefix(in string) string {
	in = strings.TrimSpace(in)
	if strings.Contains(in, "/") {
		return in
	}
	if a, err := netip.ParseAddr(in); err == nil {
		return fmt.Sprintf("%s/%d", a.String(), a.BitLen())
	}
	return in
}

func defStatus(s Status) Status {
	if s == "" {
		return Active
	}
	return s
}

func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
