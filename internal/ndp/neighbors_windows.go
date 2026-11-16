//go:build windows

package ndp

import (
	"context"
	"os/exec"
)

func init() { neighborsFn = windowsNeighbors }

func windowsNeighbors(ctx context.Context) ([]Neighbor, error) {
	out, err := exec.CommandContext(ctx, "netsh", "interface", "ipv6", "show", "neighbors").Output()
	if err != nil {
		return nil, err
	}
	return parseWindowsNeighbors(string(out)), nil
}
