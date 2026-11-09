package ipspace

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddNestsAndGuardsOverlap(t *testing.T) {
	s := NewStore("alice")
	a, err := s.Add("10.0.0.0/8", "Corp", Active, nil, "")
	require.NoError(t, err)
	require.Equal(t, "", a.ParentID)

	b, err := s.Add("10.1.0.0/16", "HQ", Active, nil, "")
	require.NoError(t, err)
	require.Equal(t, a.ID, b.ParentID) // nested under the /8

	_, err = s.Add("10.1.0.0/16", "dup", Active, nil, "")
	require.Error(t, err) // exact duplicate
	_, err = s.Add("10.0.0.0/8", "dup2", Active, nil, "")
	require.Error(t, err)
	_, err = s.Add("10.0.0.0/7", "big", Active, nil, "")
	require.Error(t, err) // would contain the existing /8

	require.Len(t, s.Records(), 2) // errors didn't mutate state
}

func TestAddHostAddress(t *testing.T) {
	s := NewStore("a")
	blk, err := s.Add("10.1.0.0/16", "HQ", Active, nil, "")
	require.NoError(t, err)
	h, err := s.Add("10.1.10.5", "web-01", Active, nil, "")
	require.NoError(t, err)
	require.Equal(t, "10.1.10.5/32", h.Prefix)
	require.Equal(t, blk.ID, h.ParentID)
}

func TestEditAndActorStamp(t *testing.T) {
	s := NewStore("alice")
	a, _ := s.Add("10.0.0.0/8", "Corp", Active, nil, "")
	require.NoError(t, s.Edit(a.ID, "Corp HQ", Reserved, []string{"x"}, "note"))
	got := s.records[a.ID]
	require.Equal(t, "Corp HQ", got.Name)
	require.Equal(t, Reserved, got.Status)
	require.Equal(t, "note", got.Notes)
	require.Equal(t, "alice", got.ChangedBy)
	require.Error(t, s.Edit("nope", "x", Active, nil, ""))
}

func TestDeleteRemoves(t *testing.T) {
	s := NewStore("a")
	a, _ := s.Add("10.0.0.0/8", "Corp", Active, nil, "")
	require.Len(t, s.Records(), 1)
	require.NoError(t, s.Delete(a.ID))
	require.Len(t, s.Records(), 0)
	require.Error(t, s.Delete("nope"))
}

func TestUndoRevertsLastAdd(t *testing.T) {
	s := NewStore("a")
	s.Add("10.0.0.0/8", "Corp", Active, nil, "")
	s.Add("10.1.0.0/16", "HQ", Active, nil, "")
	require.Len(t, s.Records(), 2)
	require.NoError(t, s.Undo())
	require.Len(t, s.Records(), 1)
}

func TestUndoEmptyIsError(t *testing.T) {
	require.Error(t, NewStore("a").Undo())
}
