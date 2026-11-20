//go:build darwin

package netdiag

import (
	"context"
	"os/exec"
)

func init() {
	neighborsRaw = func(ctx context.Context) (string, error) {
		out, err := exec.CommandContext(ctx, "arp", "-an").Output()
		return string(out), err
	}
}
