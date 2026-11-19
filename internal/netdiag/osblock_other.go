//go:build !darwin && !linux && !windows

package netdiag

// No OS-block check on this platform: osBlockProvider stays nil, so
// OSBlockSupported() is false.
