//go:build darwin

package ndp

import (
	"context"
	"os/exec"
)

func init() { neighborsFn = darwinNeighbors }

func darwinNeighbors(ctx context.Context) ([]Neighbor, error) {
	out, err := exec.CommandContext(ctx, "ndp", "-an").Output()
	if err != nil {
		return nil, err
	}
	return parseDarwinNeighbors(string(out)), nil
}
