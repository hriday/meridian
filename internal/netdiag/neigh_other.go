//go:build !darwin && !linux && !windows

package netdiag

// No neighbor-table source on this platform: neighborsRaw stays nil, so
// NeighborsSupported() is false.
