//go:build linux

package netdiag

import (
	"context"
	"os"
)

func init() {
	neighborsRaw = func(ctx context.Context) (string, error) {
		b, err := os.ReadFile("/proc/net/arp")
		return string(b), err
	}
}
