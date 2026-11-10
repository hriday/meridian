package snmp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeClient struct {
	get  map[string]any
	walk map[string][]PDU
}

func (f *fakeClient) Get(oids []string) ([]PDU, error) {
	out := make([]PDU, len(oids))
	for i, o := range oids {
		out[i] = PDU{OID: o, Value: f.get[o]} // missing -> nil (noSuchObject)
	}
	return out, nil
}
func (f *fakeClient) Walk(root string) ([]PDU, error) { return f.walk[root], nil }
func (f *fakeClient) Close() error                    { return nil }

func col(root string, entries map[int]any) []PDU {
	var out []PDU
	for idx, v := range entries {
		out = append(out, PDU{OID: root + "." + itoa(idx), Value: v})
	}
	return out
}
func itoa(i int) string { return string(rune('0' + i)) } // single-digit indices in tests

func TestListInterfaces(t *testing.T) {
	c := &fakeClient{walk: map[string][]PDU{
		oidIfDescr:      col(oidIfDescr, map[int]any{1: "Gi0/1", 2: "Tunnel0"}),
		oidIfAlias:      col(oidIfAlias, map[int]any{1: "uplink-core", 2: ""}),
		oidIfType:       col(oidIfType, map[int]any{1: 6, 2: 131}),
		oidIfHighSpeed:  col(oidIfHighSpeed, map[int]any{1: uint(1000), 2: uint(0)}),
		oidIfSpeed:      col(oidIfSpeed, map[int]any{1: uint(1000000000), 2: uint(0)}),
		oidIfOperStatus: col(oidIfOperStatus, map[int]any{1: 1, 2: 2}),
	}}
	ifs, err := ListInterfaces(c)
	require.NoError(t, err)
	require.Len(t, ifs, 2)
	byIdx := map[int]Interface{}
	for _, i := range ifs {
		byIdx[i.Index] = i
	}
	require.Equal(t, "Gi0/1", byIdx[1].Descr)
	require.Equal(t, "uplink-core", byIdx[1].Alias)
	require.Equal(t, uint64(1000000000), byIdx[1].SpeedBps) // ifHighSpeed 1000 Mbps
	require.Equal(t, "up", byIdx[1].OperStatus)
	require.Equal(t, uint64(0), byIdx[2].SpeedBps) // tunnel, no speed
	require.Equal(t, "down", byIdx[2].OperStatus)
}

func TestInterfaceCountersHC(t *testing.T) {
	idx := 3
	c := &fakeClient{get: map[string]any{
		oidIfHCInOctets + ".3":  uint64(1000),
		oidIfHCOutOctets + ".3": uint64(2000),
		oidIfInErrors + ".3":    uint(5),
		oidIfOutErrors + ".3":   uint(1),
		oidIfOperStatus + ".3":  1,
	}}
	ctr, err := InterfaceCounters(c, idx)
	require.NoError(t, err)
	require.Equal(t, 64, ctr.Bits)
	require.Equal(t, uint64(1000), ctr.InOctets)
	require.Equal(t, uint64(2000), ctr.OutOctets)
	require.Equal(t, uint64(5), ctr.InErrors)
	require.Equal(t, "up", ctr.OperStatus)
}

func TestInterfaceCounters32Fallback(t *testing.T) {
	c := &fakeClient{get: map[string]any{
		// HC absent (nil) -> fall back
		oidIfInOctets + ".3":   uint(100),
		oidIfOutOctets + ".3":  uint(200),
		oidIfOperStatus + ".3": 1,
	}}
	ctr, err := InterfaceCounters(c, 3)
	require.NoError(t, err)
	require.Equal(t, 32, ctr.Bits)
	require.Equal(t, uint64(100), ctr.InOctets)
	require.Equal(t, uint64(200), ctr.OutOctets)
}

func TestResourceUsagePresentAndAbsent(t *testing.T) {
	// present: two cores 40/60 -> mean 50; RAM 500/1000 units*1 -> 50%
	c := &fakeClient{walk: map[string][]PDU{
		oidHrProcessorLoad: col(oidHrProcessorLoad, map[int]any{1: 40, 2: 60}),
		oidHrStorageType:   col(oidHrStorageType, map[int]any{1: oidHrStorageRAM}),
	}, get: map[string]any{
		oidHrStorageAllocUnits + ".1": 1,
		oidHrStorageSize + ".1":       1000,
		oidHrStorageUsed + ".1":       500,
	}}
	r, err := ResourceUsage(c)
	require.NoError(t, err)
	require.True(t, r.HasCPU)
	require.InDelta(t, 50, r.CPUPct, 0.1)
	require.True(t, r.HasMem)
	require.InDelta(t, 50, r.MemPct, 0.1)

	empty := &fakeClient{walk: map[string][]PDU{}}
	r2, err := ResourceUsage(empty)
	require.NoError(t, err)
	require.False(t, r2.HasCPU)
	require.False(t, r2.HasMem)
}

func TestResourceUsageClampsMemOver100(t *testing.T) {
	// some agents report used > size; a usage percent must never exceed 100.
	c := &fakeClient{walk: map[string][]PDU{
		oidHrStorageType: col(oidHrStorageType, map[int]any{1: oidHrStorageRAM}),
	}, get: map[string]any{
		oidHrStorageAllocUnits + ".1": 1,
		oidHrStorageSize + ".1":       1000,
		oidHrStorageUsed + ".1":       1270,
	}}
	r, err := ResourceUsage(c)
	require.NoError(t, err)
	require.True(t, r.HasMem)
	require.Equal(t, 100.0, r.MemPct)
}
