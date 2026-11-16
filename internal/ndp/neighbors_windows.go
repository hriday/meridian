//go:build windows

package ndp

import (
	"context"
	"os/exec"
)

func init() {
	neighborsFn = windowsNeighbors
	defaultRoutersFn = windowsRouters
}

func windowsNeighbors(ctx context.Context) ([]Neighbor, error) {
	out, err := exec.CommandContext(ctx, "netsh", "interface", "ipv6", "show", "neighbors").Output()
	if err != nil {
		return nil, err
	}
	return parseWindowsNeighbors(string(out)), nil
}

func windowsRouters(ctx context.Context) ([]Router, error) {
	out, err := exec.CommandContext(ctx, "netsh", "interface", "ipv6", "show", "route").Output()
	if err != nil {
		return nil, err
	}
	return parseWindowsRouters(string(out)), nil
}
