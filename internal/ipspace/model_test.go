package ipspace

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestReplayAddEditDelete(t *testing.T) {
	t0 := time.Unix(1000, 0).UTC()
	events := []Event{
		{ID: "e1", Type: "add", Actor: "alice", At: t0, Record: Record{ID: "A", Prefix: "10.0.0.0/8", Name: "Corp"}},
		{ID: "e2", Type: "edit", Actor: "bob", At: t0.Add(time.Minute), Record: Record{ID: "A", Name: "Corp HQ", Status: "active"}},
		{ID: "e3", Type: "add", Actor: "alice", At: t0.Add(2 * time.Minute), Record: Record{ID: "B", Prefix: "10.1.0.0/16", Name: "HQ", ParentID: "A"}},
	}
	m := replay(events)
	require.Len(t, m, 2)
	require.Equal(t, "Corp HQ", m["A"].Name)
	require.Equal(t, "alice", m["A"].CreatedBy)
	require.Equal(t, "bob", m["A"].ChangedBy)
	require.Equal(t, t0, m["A"].CreatedAt)
	require.Equal(t, "A", m["B"].ParentID)

	withDelete := append(events, Event{ID: "e4", Type: "delete", Actor: "bob", At: t0, Record: Record{ID: "B"}})
	require.Len(t, replay(withDelete), 1)
}

func TestBuildTreeNesting(t *testing.T) {
	m := map[string]Record{
		"A": {ID: "A", Prefix: "10.0.0.0/8"},
		"B": {ID: "B", Prefix: "10.1.0.0/16", ParentID: "A"},
		"C": {ID: "C", Prefix: "10.2.0.0/16", ParentID: "A"},
	}
	tree := buildTree(m)
	require.Len(t, tree, 1)
	require.Equal(t, "A", tree[0].Record.ID)
	require.Len(t, tree[0].Children, 2)
	require.Equal(t, "B", tree[0].Children[0].Record.ID) // 10.1 sorts before 10.2
	require.Equal(t, "C", tree[0].Children[1].Record.ID)
}

func TestBuildTreeSurfacesOrphans(t *testing.T) {
	// B's parent A was deleted; B (and its subtree) must still show, as a root.
	m := map[string]Record{
		"B": {ID: "B", Prefix: "10.1.0.0/16", ParentID: "A"},
		"C": {ID: "C", Prefix: "10.1.10.0/24", ParentID: "B"},
	}
	tree := buildTree(m)
	require.Len(t, tree, 1)
	require.Equal(t, "B", tree[0].Record.ID)
	require.Len(t, tree[0].Children, 1)
	require.Equal(t, "C", tree[0].Children[0].Record.ID)
}

func TestReplayDeleteUnknownIsNoop(t *testing.T) {
	require.Len(t, replay([]Event{{Type: "delete", Record: Record{ID: "ghost"}}}), 0)
}
