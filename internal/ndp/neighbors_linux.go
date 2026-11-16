//go:build linux

package ndp

import (
	"context"
	"os/exec"
)

func init() { neighborsFn = linuxNeighbors }

func linuxNeighbors(ctx context.Context) ([]Neighbor, error) {
	out, err := exec.CommandContext(ctx, "ip", "-6", "neighbor", "show").Output()
	if err != nil {
		return nil, err
	}
	return parseLinuxNeighbors(string(out)), nil
}
