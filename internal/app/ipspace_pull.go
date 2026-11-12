package app

import (
	"fmt"

	"github.com/hriday/meridian/internal/ipspace"
)

// Cross-inventory pull copies selected records from one inventory into the
// active one — "reference, and merge later". Both must be unlocked. It's an
// explicit, reviewed copy (nothing is pulled automatically), and it never
// touches the source. Records are added to the active inventory like any other
// add, so a prefix the active already holds is reported as skipped rather than
// overwritten.

// PullReport is the outcome of a cross-inventory pull.
type PullReport struct {
	Added   int      `json:"added"`
	Skipped []string `json:"skipped"` // "prefix — reason"
}

// IPSpaceInventoryTree returns another unlocked inventory's record tree, for
// choosing what to pull. Empty when that inventory is locked.
func (a *App) IPSpaceInventoryTree(id string) []ipspace.TreeNode {
	a.ipMu.Lock()
	defer a.ipMu.Unlock()
	a.ensureInv()
	store, _, _, ok := a.targetLive(id)
	if !ok {
		return nil
	}
	return store.Tree()
}

// IPSpacePullRecords copies the given prefixes from a source inventory into the
// active one. The source must be a different, unlocked inventory. Each record is
// added to the active inventory; a prefix it already holds (or that overlaps) is
// skipped and reported, never overwritten.
func (a *App) IPSpacePullRecords(sourceID string, prefixes []string) (PullReport, error) {
	a.ipMu.Lock()
	defer a.ipMu.Unlock()
	a.ensureInv()
	if err := a.requireUnlocked(); err != nil {
		return PullReport{}, err
	}
	if sourceID == a.activeID {
		return PullReport{}, fmt.Errorf("pick a different inventory to pull from")
	}
	src, _, _, ok := a.targetLive(sourceID)
	if !ok {
		return PullReport{}, fmt.Errorf("unlock that inventory first, then pull from it")
	}
	byPrefix := map[string]ipspace.Record{}
	for _, r := range src.Records() {
		byPrefix[r.Prefix] = r
	}
	var rep PullReport
	for _, p := range prefixes {
		r, found := byPrefix[p]
		if !found {
			rep.Skipped = append(rep.Skipped, p+" — not found in the source")
			continue
		}
		if _, err := a.ipStore.Add(r.Prefix, r.Name, r.Status, r.Tags, r.Notes); err != nil {
			rep.Skipped = append(rep.Skipped, r.Prefix+" — "+err.Error())
			continue
		}
		rep.Added++
	}
	if rep.Added > 0 {
		if err := a.ipSave(); err != nil {
			return rep, err
		}
	}
	return rep, nil
}
