//go:build darwin

package ndp

import (
	"context"
	"os/exec"
)

func init() {
	neighborsFn = darwinNeighbors
	defaultRoutersFn = darwinRouters
}

func darwinNeighbors(ctx context.Context) ([]Neighbor, error) {
	out, err := exec.CommandContext(ctx, "ndp", "-an").Output()
	if err != nil {
		return nil, err
	}
	return parseDarwinNeighbors(string(out)), nil
}

func darwinRouters(ctx context.Context) ([]Router, error) {
	out, err := exec.CommandContext(ctx, "ndp", "-rn").Output()
	if err != nil {
		return nil, err
	}
	return parseDarwinRouters(string(out)), nil
}
