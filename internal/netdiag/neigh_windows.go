//go:build windows

package netdiag

import (
	"context"
	"os/exec"
)

func init() {
	neighborsRaw = func(ctx context.Context) (string, error) {
		out, err := exec.CommandContext(ctx, "arp", "-a").Output()
		return string(out), err
	}
}
