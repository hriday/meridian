//go:build !windows

package netdiag

func isRefusedPlatform(error) bool { return false }
