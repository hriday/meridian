//go:build !darwin && !linux && !windows

package linkpath

// readLink is the fallback for platforms where Meridian can't read link
// speed/duplex. The read is simply "unknown", and assess() degrades to the
// LLDP-only signals with an honest caveat.
func readLink(_ string) linkState { return linkState{} }
